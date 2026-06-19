package orgadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

var ErrSelfDeactivation = errors.New("não é possível desativar o próprio usuário")

type ServiceInterface interface {
	ListUsers(ctx context.Context, orgID string) ([]domain.OrgUser, error)
	PatchUser(ctx context.Context, userID, orgID, actorID string, isActive *bool, creditLimit *int) error
	DeleteUser(ctx context.Context, userID, orgID, actorID string) error
	CreateInvitation(ctx context.Context, orgID, email, role, actorID string) (*domain.Invitation, error)
	ListInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error)
	DeleteInvitation(ctx context.Context, invitationID, orgID string) error
	GetUserHistory(ctx context.Context, userID, orgID string, days, page, limit int) (*UserHistory, error)
	GetOrgCosts(ctx context.Context, orgID string, days int) (*domain.OrgCosts, error)
	GetOrgCredits(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error)
	GetSellerProfile(ctx context.Context, userID string) (*domain.SellerProfile, error)
	UpdateProfile(ctx context.Context, userID, name, currentPassword, newPassword string) error
	ListSellerSearches(ctx context.Context, userID, orgID string, days, page, limit int) ([]SearchSummary, int, error)
}

type Service struct {
	repo       Repository
	creditsSvc credits.ServiceInterface
}

func NewService(repo Repository, creditsSvc credits.ServiceInterface) *Service {
	return &Service{repo: repo, creditsSvc: creditsSvc}
}

func (s *Service) ListUsers(ctx context.Context, orgID string) ([]domain.OrgUser, error) {
	return s.repo.ListUsers(ctx, orgID)
}

func (s *Service) PatchUser(ctx context.Context, userID, orgID, actorID string, isActive *bool, creditLimit *int) error {
	if isActive != nil && !*isActive && userID == actorID {
		return ErrSelfDeactivation
	}
	if err := s.repo.PatchUser(ctx, userID, orgID, isActive, creditLimit); err != nil {
		return err
	}
	if isActive != nil && !*isActive {
		targetID := userID
		orgIDStr := orgID
		if err := s.repo.WriteAuditLog(ctx, &orgIDStr, actorID, "user.deactivate", &targetID, map[string]any{
			"user_id": userID,
		}); err != nil {
			slog.WarnContext(ctx, "audit log write failed", "action", "user.deactivate", "user_id", userID, "error", err)
		}
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, userID, orgID, actorID string) error {
	if userID == actorID {
		return ErrSelfDeactivation
	}
	if err := s.repo.SoftDeleteUser(ctx, userID, orgID); err != nil {
		return err
	}
	targetID := userID
	orgIDStr := orgID
	if err := s.repo.WriteAuditLog(ctx, &orgIDStr, actorID, "user.delete", &targetID, map[string]any{
		"user_id": userID,
	}); err != nil {
		slog.WarnContext(ctx, "audit log write failed", "action", "user.delete", "user_id", userID, "error", err)
	}
	return nil
}

func (s *Service) CreateInvitation(ctx context.Context, orgID, email, role, actorID string) (*domain.Invitation, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	inv, err := s.repo.CreateInvitation(ctx, orgID, email, role, token, actorID)
	if err != nil {
		return nil, err
	}

	orgIDStr := orgID
	invID := inv.ID
	if err := s.repo.WriteAuditLog(ctx, &orgIDStr, actorID, "user.invite", &invID, map[string]any{
		"email": email,
		"role":  role,
	}); err != nil {
		slog.WarnContext(ctx, "audit log write failed", "action", "user.invite", "email", email, "error", err)
	}
	return inv, nil
}

func (s *Service) ListInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error) {
	return s.repo.ListInvitations(ctx, orgID)
}

func (s *Service) DeleteInvitation(ctx context.Context, invitationID, orgID string) error {
	return s.repo.DeleteInvitation(ctx, invitationID, orgID)
}

func (s *Service) GetUserHistory(ctx context.Context, userID, orgID string, days, page, limit int) (*UserHistory, error) {
	return s.repo.GetUserHistory(ctx, userID, orgID, days, page, limit)
}

func (s *Service) GetOrgCosts(ctx context.Context, orgID string, days int) (*domain.OrgCosts, error) {
	return s.repo.GetOrgCosts(ctx, orgID, days)
}

func (s *Service) GetOrgCredits(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	return s.creditsSvc.GetBalance(ctx, orgID)
}

func (s *Service) GetSellerProfile(ctx context.Context, userID string) (*domain.SellerProfile, error) {
	return s.repo.GetSellerProfile(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID, name, currentPassword, newPassword string) error {
	var newHash *string
	if newPassword != "" {
		if currentPassword == "" {
			return fmt.Errorf("senha atual é obrigatória para alterar a senha")
		}
		h, err := auth.HashPassword(newPassword)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		newHash = &h
	}
	return s.repo.UpdateProfile(ctx, userID, name, newHash)
}

func (s *Service) ListSellerSearches(ctx context.Context, userID, orgID string, days, page, limit int) ([]SearchSummary, int, error) {
	return s.repo.ListSellerSearches(ctx, userID, orgID, days, page, limit)
}

// CreditBalanceResponse is a local alias to avoid importing credits package in domain.
type CreditBalanceResponse = domain.CreditBalanceResponse
