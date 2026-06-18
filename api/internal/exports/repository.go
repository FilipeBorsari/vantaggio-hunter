package exports

import (
	"context"
	"time"

	"github.com/vantaggio/prospect-api/internal/domain"
)

// Repository handles persistence for CRM integrations and export jobs.
type Repository interface {
	// Integration
	SaveIntegration(ctx context.Context, orgID, crmType, baseURL, encAPIKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error)
	GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error)
	// GetIntegrationRaw returns the encrypted api_key and account_id for internal use by the worker.
	GetIntegrationRaw(ctx context.Context, orgID string) (baseURL, encAPIKey string, inboxID *int, accountID int, err error)

	// Export jobs
	CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error)
	GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error)
	ListExports(ctx context.Context, orgID string, page, limit int) ([]domain.ExportJob, int, error)
	// MarkProcessing sets status=processing and increments attempt, returning the new attempt number.
	MarkProcessing(ctx context.Context, id string) (attempt int, err error)
	UpdateExportResult(ctx context.Context, id string, status domain.ExportStatus, successCount, failCount int, errorLog []domain.ExportErrorEntry, nextRetryAt *time.Time) error
	// IncrFunnelExported updates leads_exportados in fato_funil_leads for the given search.
	IncrFunnelExported(ctx context.Context, searchID, orgID string, count int) error
}
