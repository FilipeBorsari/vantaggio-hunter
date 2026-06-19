// Package orgadmin implementa as rotas de gestão intra-organização:
// org_admin gerencia seus sellers e visualiza custos/créditos;
// sellers acessam o próprio perfil e histórico de buscas.
// Todas as queries são isoladas por org_id do JWT — nenhum dado
// de outra organização é acessível.
package orgadmin

import (
	"context"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	ListUsers(ctx context.Context, orgID string) ([]domain.OrgUser, error)
	GetUser(ctx context.Context, userID, orgID string) (*domain.OrgUser, error)
	PatchUser(ctx context.Context, userID, orgID string, isActive *bool, creditLimit *int) error
	SoftDeleteUser(ctx context.Context, userID, orgID string) error
	CreateInvitation(ctx context.Context, orgID, email, role, token, invitedBy string) (*domain.Invitation, error)
	ListInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error)
	DeleteInvitation(ctx context.Context, invitationID, orgID string) error
	GetUserHistory(ctx context.Context, userID, orgID string, days, page, limit int) (*UserHistory, error)
	GetOrgCosts(ctx context.Context, orgID string, days int) (*domain.OrgCosts, error)
	GetSellerProfile(ctx context.Context, userID string) (*domain.SellerProfile, error)
	UpdateProfile(ctx context.Context, userID, name string, passwordHash *string) error
	ListSellerSearches(ctx context.Context, userID, orgID string, days, page, limit int) ([]SearchSummary, int, error)
	WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error
}

type UserHistory struct {
	User     domain.OrgUser  `json:"user"`
	Stats    UserHistStats   `json:"stats"`
	Searches []SearchSummary `json:"searches"`
	Page     int             `json:"page"`
	Total    int             `json:"total"`
}

type UserHistStats struct {
	Searches        int `json:"searches"`
	Exports         int `json:"exports"`
	CreditsConsumed int `json:"credits_consumed"`
}

type SearchSummary struct {
	SearchID     string `json:"search_id"`
	Query        string `json:"query"`
	ResultsCount int    `json:"results_count"`
	CreditsUsed  int    `json:"credits_used"`
	CreatedAt    string `json:"created_at"`
}
