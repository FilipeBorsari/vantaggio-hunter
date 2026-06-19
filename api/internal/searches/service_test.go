package searches_test

import (
	"context"
	"errors"
	"testing"

	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/searches"
)

// mockRepo implements searches.Repository for testing.
type mockRepo struct {
	createFn                func(ctx context.Context, s *domain.Search) error
	getByIDFn               func(ctx context.Context, id, orgID string) (*domain.Search, error)
	getByIDForWorkerFn      func(ctx context.Context, id string) (*domain.Search, error)
	listFn                  func(ctx context.Context, orgID string, page, limit int) ([]domain.Search, int, error)
	updateStatusFn          func(ctx context.Context, id string, status domain.SearchStatus, resultCount *int, errMsg *string) error
	runStructuredFn         func(ctx context.Context, searchID string, f domain.SearchFilters) (int, error)
	runSemanticFn           func(ctx context.Context, searchID string, f domain.SearchFilters, vec []float32, queryText string) (int, error)
	getResultsFn            func(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error)
	searchCNAEsFn           func(ctx context.Context, q string) ([]domain.CNAE, error)
	recoverStaleSearchesFn  func(ctx context.Context, staleMinutes int) (int64, error)
	listQueuedSearchIDsFn      func(ctx context.Context) ([]string, error)
	getCompanyEmbedInputsFn    func(ctx context.Context, searchID string, limit int) ([]domain.CompanyEmbedInput, error)
	saveEmbeddingsFn           func(ctx context.Context, embeddings []domain.CompanyEmbedding) error
}

func (m *mockRepo) Create(ctx context.Context, s *domain.Search) error {
	return m.createFn(ctx, s)
}
func (m *mockRepo) GetByID(ctx context.Context, id, orgID string) (*domain.Search, error) {
	return m.getByIDFn(ctx, id, orgID)
}
func (m *mockRepo) GetByIDForWorker(ctx context.Context, id string) (*domain.Search, error) {
	return m.getByIDForWorkerFn(ctx, id)
}
func (m *mockRepo) List(ctx context.Context, orgID string, page, limit int) ([]domain.Search, int, error) {
	return m.listFn(ctx, orgID, page, limit)
}
func (m *mockRepo) UpdateStatus(ctx context.Context, id string, status domain.SearchStatus, resultCount *int, errMsg *string) error {
	return m.updateStatusFn(ctx, id, status, resultCount, errMsg)
}
func (m *mockRepo) RunStructuredSearch(ctx context.Context, searchID string, f domain.SearchFilters) (int, error) {
	return m.runStructuredFn(ctx, searchID, f)
}
func (m *mockRepo) RunSemanticSearch(ctx context.Context, searchID string, f domain.SearchFilters, vec []float32, queryText string) (int, error) {
	return m.runSemanticFn(ctx, searchID, f, vec, queryText)
}
func (m *mockRepo) GetResults(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error) {
	return m.getResultsFn(ctx, searchID, page, limit)
}
func (m *mockRepo) SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error) {
	return m.searchCNAEsFn(ctx, q)
}
func (m *mockRepo) RecoverStaleSearches(ctx context.Context, staleMinutes int) (int64, error) {
	if m.recoverStaleSearchesFn != nil {
		return m.recoverStaleSearchesFn(ctx, staleMinutes)
	}
	return 0, nil
}
func (m *mockRepo) ListQueuedSearchIDs(ctx context.Context) ([]string, error) {
	if m.listQueuedSearchIDsFn != nil {
		return m.listQueuedSearchIDsFn(ctx)
	}
	return nil, nil
}
func (m *mockRepo) GetCompanyEmbedInputs(ctx context.Context, searchID string, limit int) ([]domain.CompanyEmbedInput, error) {
	if m.getCompanyEmbedInputsFn != nil {
		return m.getCompanyEmbedInputsFn(ctx, searchID, limit)
	}
	return nil, nil
}
func (m *mockRepo) SaveEmbeddings(ctx context.Context, embeddings []domain.CompanyEmbedding) error {
	if m.saveEmbeddingsFn != nil {
		return m.saveEmbeddingsFn(ctx, embeddings)
	}
	return nil
}

func TestService_Create_Structured_OK(t *testing.T) {
	uf := "SP"
	repo := &mockRepo{
		createFn: func(ctx context.Context, s *domain.Search) error {
			s.ID = "abc-123"
			s.Status = domain.SearchStatusQueued
			s.CreatedAt = "2026-01-01T00:00:00Z"
			return nil
		},
	}
	svc := searches.NewService(repo)

	filters := domain.SearchFilters{UF: &uf}
	got, err := svc.Create(context.Background(), "org1", "user1", domain.SearchModeStructured, filters, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "abc-123" {
		t.Errorf("expected ID abc-123, got %s", got.ID)
	}
	if got.Status != domain.SearchStatusQueued {
		t.Errorf("expected status queued, got %s", got.Status)
	}
}

func TestService_Create_Structured_EmptyFilters(t *testing.T) {
	repo := &mockRepo{}
	svc := searches.NewService(repo)

	_, err := svc.Create(context.Background(), "org1", "user1", domain.SearchModeStructured, domain.SearchFilters{}, nil)
	if err == nil {
		t.Fatal("expected error for empty filters")
	}
	if !errors.Is(err, searches.ErrInvalidSearchInput) {
		t.Errorf("expected ErrInvalidSearchInput, got %v", err)
	}
}

func TestService_Create_Semantic_NoQuery(t *testing.T) {
	repo := &mockRepo{}
	svc := searches.NewService(repo)

	_, err := svc.Create(context.Background(), "org1", "user1", domain.SearchModeSemantic, domain.SearchFilters{}, nil)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if !errors.Is(err, searches.ErrInvalidSearchInput) {
		t.Errorf("expected ErrInvalidSearchInput, got %v", err)
	}
}

func TestService_Create_Semantic_OK(t *testing.T) {
	query := "padarias artesanais em Curitiba"
	repo := &mockRepo{
		createFn: func(ctx context.Context, s *domain.Search) error {
			s.ID = "sem-1"
			s.Status = domain.SearchStatusQueued
			s.CreatedAt = "2026-01-01T00:00:00Z"
			return nil
		},
	}
	svc := searches.NewService(repo)

	got, err := svc.Create(context.Background(), "org1", "user1", domain.SearchModeSemantic, domain.SearchFilters{}, &query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "sem-1" {
		t.Errorf("expected ID sem-1, got %s", got.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(ctx context.Context, id, orgID string) (*domain.Search, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := searches.NewService(repo)

	_, err := svc.Get(context.Background(), "no-id", "org1", 1, 100)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Get_Done_WithResults(t *testing.T) {
	count := 2
	repo := &mockRepo{
		getByIDFn: func(ctx context.Context, id, orgID string) (*domain.Search, error) {
			return &domain.Search{
				ID:          "s1",
				Status:      domain.SearchStatusDone,
				ResultCount: &count,
				CreatedAt:   "2026-01-01T00:00:00Z",
			}, nil
		},
		getResultsFn: func(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error) {
			return []domain.SearchResult{
				{CNPJ: "00000000000001", RazaoSocial: "Empresa A", UF: "SP", SituacaoCadastral: 2, CNAEs: []domain.CNAE{}},
				{CNPJ: "00000000000002", RazaoSocial: "Empresa B", UF: "SP", SituacaoCadastral: 2, CNAEs: []domain.CNAE{}},
			}, 2, nil
		},
	}
	svc := searches.NewService(repo)

	resp, err := svc.Get(context.Background(), "s1", "org1", 1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
	if *resp.Total != 2 {
		t.Errorf("expected total 2, got %d", *resp.Total)
	}
}

func TestService_SearchCNAEs_EmptyQuery(t *testing.T) {
	repo := &mockRepo{}
	svc := searches.NewService(repo)

	cnaes, err := svc.SearchCNAEs(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cnaes) != 0 {
		t.Errorf("expected 0 cnaes for empty query, got %d", len(cnaes))
	}
}
