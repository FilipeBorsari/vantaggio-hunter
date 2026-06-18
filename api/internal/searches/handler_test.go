package searches_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/searches"
)

// mockSvc implements searches.ServiceInterface for testing.
type mockSvc struct {
	createFn      func(ctx context.Context, orgID, userID string, mode domain.SearchMode, filters domain.SearchFilters, query *string) (*domain.Search, error)
	getFn         func(ctx context.Context, id, orgID string, page, limit int) (*domain.SearchResponse, error)
	listFn        func(ctx context.Context, orgID string, page, limit int) (*domain.SearchListResponse, error)
	searchCNAEsFn func(ctx context.Context, q string) ([]domain.CNAE, error)
}

func (m *mockSvc) Create(ctx context.Context, orgID, userID string, mode domain.SearchMode, filters domain.SearchFilters, query *string) (*domain.Search, error) {
	return m.createFn(ctx, orgID, userID, mode, filters, query)
}
func (m *mockSvc) Get(ctx context.Context, id, orgID string, page, limit int) (*domain.SearchResponse, error) {
	return m.getFn(ctx, id, orgID, page, limit)
}
func (m *mockSvc) List(ctx context.Context, orgID string, page, limit int) (*domain.SearchListResponse, error) {
	return m.listFn(ctx, orgID, page, limit)
}
func (m *mockSvc) SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error) {
	return m.searchCNAEsFn(ctx, q)
}

func newTestRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

func TestHandler_Create_InvalidMode(t *testing.T) {
	svc := &mockSvc{}
	h := searches.NewHandler(svc, newTestRedis())

	body := `{"mode":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/searches", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Create_InvalidInput(t *testing.T) {
	svc := &mockSvc{
		createFn: func(ctx context.Context, orgID, userID string, mode domain.SearchMode, filters domain.SearchFilters, query *string) (*domain.Search, error) {
			return nil, searches.ErrInvalidSearchInput
		},
	}
	h := searches.NewHandler(svc, newTestRedis())

	body := `{"mode":"structured","filters":{}}`
	req := httptest.NewRequest(http.MethodPost, "/searches", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockSvc{
		getFn: func(ctx context.Context, id, orgID string, page, limit int) (*domain.SearchResponse, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := searches.NewHandler(svc, newTestRedis())

	r := chi.NewRouter()
	r.Get("/searches/{id}", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/searches/no-id", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_List_OK(t *testing.T) {
	svc := &mockSvc{
		listFn: func(ctx context.Context, orgID string, page, limit int) (*domain.SearchListResponse, error) {
			return &domain.SearchListResponse{Data: []domain.Search{}, Total: 0}, nil
		},
	}
	h := searches.NewHandler(svc, newTestRedis())

	req := httptest.NewRequest(http.MethodGet, "/searches", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp domain.SearchListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestHandler_SearchCNAEs_OK(t *testing.T) {
	svc := &mockSvc{
		searchCNAEsFn: func(ctx context.Context, q string) ([]domain.CNAE, error) {
			return []domain.CNAE{{Code: "4520-0/01", Description: "Manutenção e reparação"}}, nil
		},
	}
	h := searches.NewHandler(svc, newTestRedis())

	req := httptest.NewRequest(http.MethodGet, "/cnaes?q=manutencao", nil)
	rec := httptest.NewRecorder()

	h.SearchCNAEs(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
