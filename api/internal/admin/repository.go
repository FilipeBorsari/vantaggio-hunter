package admin

import (
	"context"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	ListPlans(ctx context.Context) ([]domain.Plan, error)
	CreateOrg(ctx context.Context, name string, planID *string) (*domain.Org, error)
	CreateUser(ctx context.Context, orgID, name, email, passwordHash, role string) (*domain.User, error)
	ListOrgs(ctx context.Context, page, limit int, q string) ([]domain.Org, error)
	CountOrgs(ctx context.Context) (int, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
	GetOrgDetail(ctx context.Context, orgID string) (*domain.OrgDetail, error)
	PatchOrg(ctx context.Context, orgID string, isActive *bool, planID *string) error
	GetAdminDashboard(ctx context.Context, days int) (*domain.AdminDashboard, error)
	WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error
}
