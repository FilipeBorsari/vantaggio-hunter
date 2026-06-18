package searches

import (
	"context"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, s *domain.Search) error
	GetByID(ctx context.Context, id, orgID string) (*domain.Search, error)
	GetByIDForWorker(ctx context.Context, id string) (*domain.Search, error)
	List(ctx context.Context, orgID string, page, limit int) ([]domain.Search, int, error)
	UpdateStatus(ctx context.Context, id string, status domain.SearchStatus, resultCount *int, errMsg *string) error
	// RecoverStaleSearches resets searches stuck in "processing" for more than
	// staleAfter minutes back to "queued" so the worker can retry them.
	RecoverStaleSearches(ctx context.Context, staleMinutes int) (int64, error)
	RunStructuredSearch(ctx context.Context, searchID string, f domain.SearchFilters) (int, error)
	RunSemanticSearch(ctx context.Context, searchID string, f domain.SearchFilters, queryVec []float32) (int, error)
	GetResults(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error)
	SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error)
}
