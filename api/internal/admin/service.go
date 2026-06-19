package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

var ErrEmailAlreadyExists = errors.New("email já existe")

type ServiceInterface interface {
	ListPlans(ctx context.Context) ([]domain.Plan, error)
	CreateOrgWithAdmin(ctx context.Context, name string, planID *string, adminEmail, adminName string) (*CreateOrgResult, error)
	CreateUser(ctx context.Context, orgID, name, email, password, role string) (*domain.User, error)
	ListOrgs(ctx context.Context, page, limit int, q string) (*domain.OrgListResponse, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
	GetOrgDetail(ctx context.Context, orgID string) (*domain.OrgDetail, error)
	PatchOrg(ctx context.Context, orgID string, isActive *bool, planID *string) error
	GetAdminDashboard(ctx context.Context, days int) (*domain.AdminDashboard, error)
	AddCreditsToOrg(ctx context.Context, orgID string, amount int, desc, actorID string) (int, error)
	Impersonate(ctx context.Context, orgID, actorID string) (string, error)
	WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error
}

type CreateOrgResult struct {
	OrgID        string `json:"org_id"`
	UserID       string `json:"user_id"`
	TempPassword string `json:"temp_password"`
}

type Service struct {
	repo       Repository
	creditsSvc credits.ServiceInterface
}

func NewService(repo Repository, creditsSvc credits.ServiceInterface) *Service {
	return &Service{repo: repo, creditsSvc: creditsSvc}
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *Service) CreateOrgWithAdmin(ctx context.Context, name string, planID *string, adminEmail, adminName string) (*CreateOrgResult, error) {
	org, err := s.repo.CreateOrg(ctx, name, planID)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	tempPass, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate temp password: %w", err)
	}
	hash, err := auth.HashPassword(tempPass)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, org.ID, adminName, adminEmail, hash, "org_admin")
	if err != nil {
		return nil, fmt.Errorf("create org admin: %w", err)
	}

	return &CreateOrgResult{
		OrgID:        org.ID,
		UserID:       user.ID,
		TempPassword: tempPass,
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, orgID, name, email, password, role string) (*domain.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return s.repo.CreateUser(ctx, orgID, name, email, hash, role)
}

func (s *Service) ListOrgs(ctx context.Context, page, limit int, q string) (*domain.OrgListResponse, error) {
	orgs, err := s.repo.ListOrgs(ctx, page, limit, q)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountOrgs(ctx)
	if err != nil {
		return nil, fmt.Errorf("count orgs: %w", err)
	}
	return &domain.OrgListResponse{Data: orgs, Total: total}, nil
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	return s.repo.SetUserActive(ctx, userID, isActive)
}

func (s *Service) GetOrgDetail(ctx context.Context, orgID string) (*domain.OrgDetail, error) {
	return s.repo.GetOrgDetail(ctx, orgID)
}

func (s *Service) PatchOrg(ctx context.Context, orgID string, isActive *bool, planID *string) error {
	return s.repo.PatchOrg(ctx, orgID, isActive, planID)
}

func (s *Service) GetAdminDashboard(ctx context.Context, days int) (*domain.AdminDashboard, error) {
	return s.repo.GetAdminDashboard(ctx, days)
}

func (s *Service) AddCreditsToOrg(ctx context.Context, orgID string, amount int, desc, actorID string) (int, error) {
	if err := s.creditsSvc.AddCredits(ctx, orgID, amount, desc); err != nil {
		return 0, fmt.Errorf("add credits: %w", err)
	}
	bal, err := s.creditsSvc.GetBalance(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}

	orgIDStr := orgID
	targetIDStr := orgID
	if logErr := s.repo.WriteAuditLog(ctx, &orgIDStr, actorID, "credits.add", &targetIDStr, map[string]any{
		"amount":      amount,
		"description": desc,
	}); logErr != nil {
		slog.WarnContext(ctx, "audit log write failed", "action", "credits.add", "org_id", orgID, "error", logErr)
	}

	return bal.Balance, nil
}

func (s *Service) Impersonate(ctx context.Context, orgID, actorID string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		OrgID:          orgID,
		UserID:         actorID,
		Role:           "org_admin",
		ImpersonatedBy: actorID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign impersonation token: %w", err)
	}

	orgIDStr := orgID
	targetIDStr := orgID
	if logErr := s.repo.WriteAuditLog(ctx, &orgIDStr, actorID, "org.impersonate", &targetIDStr, map[string]any{
		"impersonated_org": orgID,
	}); logErr != nil {
		slog.WarnContext(ctx, "audit log write failed", "action", "org.impersonate", "org_id", orgID, "error", logErr)
	}

	return signed, nil
}

func (s *Service) WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error {
	return s.repo.WriteAuditLog(ctx, orgID, actorID, action, targetID, metadata)
}
