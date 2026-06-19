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
	// ListQueuedSearchIDs returns the IDs of all searches currently in "queued"
	// status, so the worker can re-push them to Redis after a restart or stale recovery.
	ListQueuedSearchIDs(ctx context.Context) ([]string, error)
	RunStructuredSearch(ctx context.Context, searchID string, f domain.SearchFilters) (int, error)
	RunSemanticSearch(ctx context.Context, searchID string, f domain.SearchFilters, queryVec []float32, queryText string) (int, error)
	GetResults(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error)
	SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error)
	// GetCompanyEmbedInputs returns up to limit companies from the search results
	// that do not yet have an embedding, with pre-built embedding text.
	GetCompanyEmbedInputs(ctx context.Context, searchID string, limit int) ([]domain.CompanyEmbedInput, error)
	// SaveEmbeddings persists generated vectors back into tb_companies.
	SaveEmbeddings(ctx context.Context, embeddings []domain.CompanyEmbedding) error
	// EstimateCount returns the number of companies that would be returned by a
	// search with the given mode and filters, without creating the search.
	EstimateCount(ctx context.Context, mode domain.SearchMode, f domain.SearchFilters, queryText string) (int, error)
}
