package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
)

var ErrEmailAlreadyExists = errors.New("email já existe")

type ServiceInterface interface {
	ListPlans(ctx context.Context) ([]domain.Plan, error)
	CreateOrg(ctx context.Context, name string, planID *string) (*domain.Org, error)
	CreateUser(ctx context.Context, orgID, email, password, role string) (*domain.User, error)
	ListOrgs(ctx context.Context, page, limit int) (*domain.OrgListResponse, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *Service) CreateOrg(ctx context.Context, name string, planID *string) (*domain.Org, error) {
	return s.repo.CreateOrg(ctx, name, planID)
}

func (s *Service) CreateUser(ctx context.Context, orgID, email, password, role string) (*domain.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return s.repo.CreateUser(ctx, orgID, email, hash, role)
}

func (s *Service) ListOrgs(ctx context.Context, page, limit int) (*domain.OrgListResponse, error) {
	orgs, err := s.repo.ListOrgs(ctx, page, limit)
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
