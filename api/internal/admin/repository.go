package admin

import (
	"context"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	ListPlans(ctx context.Context) ([]domain.Plan, error)
	CreateOrg(ctx context.Context, name string, planID *string) (*domain.Org, error)
	CreateUser(ctx context.Context, orgID, email, passwordHash, role string) (*domain.User, error)
	ListOrgs(ctx context.Context, page, limit int) ([]domain.Org, error)
	CountOrgs(ctx context.Context) (int, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
}
