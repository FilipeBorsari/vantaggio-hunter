package ia_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/ia"
)

// mockService implements ia.ServiceInterface for handler tests.
type mockService struct {
	qualifyFn            func(ctx context.Context, orgID, userID, cnpj string) (*ia.QualifyResult, error)
	listQualificationsFn func(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error)
}

func (m *mockService) Qualify(ctx context.Context, orgID, userID, cnpj string) (*ia.QualifyResult, error) {
	return m.qualifyFn(ctx, orgID, userID, cnpj)
}
func (m *mockService) ListQualifications(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error) {
	return m.listQualificationsFn(ctx, orgID, cnpj)
}

func ctxWithAuth(orgID, userID string) context.Context {
	ctx := context.WithValue(context.Background(), authpkg.ContextKeyOrgID, orgID)
	return context.WithValue(ctx, authpkg.ContextKeyUserID, userID)
}

func routerWithCNPJ(h *ia.Handler, cnpj string) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/ia/qualify/{cnpj}", h.Qualify)
	r.Get("/ia/qualifications", h.ListQualifications)
	return r
}

func TestHandler_Qualify_InvalidCNPJ(t *testing.T) {
	h := ia.NewHandler(&mockService{})
	r := chi.NewRouter()
	r.Post("/ia/qualify/{cnpj}", h.Qualify)

	req := httptest.NewRequest(http.MethodPost, "/ia/qualify/123", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Qualify_InsufficientCredits(t *testing.T) {
	svc := &mockService{
		qualifyFn: func(_ context.Context, _, _, _ string) (*ia.QualifyResult, error) {
			return nil, ia.ErrInsufficientCredits
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "12345678000195")

	req := httptest.NewRequest(http.MethodPost, "/ia/qualify/12345678000195", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402, got %d", rec.Code)
	}
}

func TestHandler_Qualify_CompanyNotFound(t *testing.T) {
	svc := &mockService{
		qualifyFn: func(_ context.Context, _, _, _ string) (*ia.QualifyResult, error) {
			return nil, ia.ErrCompanyNotFound
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "12345678000195")

	req := httptest.NewRequest(http.MethodPost, "/ia/qualify/12345678000195", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_Qualify_Success(t *testing.T) {
	svc := &mockService{
		qualifyFn: func(_ context.Context, _, _, _ string) (*ia.QualifyResult, error) {
			return &ia.QualifyResult{
				QualificationID: "uuid",
				CNPJ:            "12345678000195",
				Score:           82,
				Justification:   "Empresa sólida",
				Model:           "gpt-4o-mini",
				CreditsUsed:     10,
			}, nil
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "12345678000195")

	req := httptest.NewRequest(http.MethodPost, "/ia/qualify/12345678000195", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestHandler_Qualify_CacheHit_Returns200(t *testing.T) {
	svc := &mockService{
		qualifyFn: func(_ context.Context, _, _, _ string) (*ia.QualifyResult, error) {
			return &ia.QualifyResult{
				QualificationID: "cached-uuid",
				Score:           70,
				FromCache:       true,
			}, nil
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "12345678000195")

	req := httptest.NewRequest(http.MethodPost, "/ia/qualify/12345678000195", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for cache hit, got %d", rec.Code)
	}
}

func TestHandler_ListQualifications_OK(t *testing.T) {
	svc := &mockService{
		listQualificationsFn: func(_ context.Context, _ string, _ *string) ([]domain.AIQualification, error) {
			return []domain.AIQualification{
				{ID: "q1", CNPJ: "12345678000195", Score: 80, CreatedAt: time.Now()},
			}, nil
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "")

	req := httptest.NewRequest(http.MethodGet, "/ia/qualifications", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_ListQualifications_InternalError(t *testing.T) {
	svc := &mockService{
		listQualificationsFn: func(_ context.Context, _ string, _ *string) ([]domain.AIQualification, error) {
			return nil, errors.New("db error")
		},
	}
	h := ia.NewHandler(svc)
	r := routerWithCNPJ(h, "")

	req := httptest.NewRequest(http.MethodGet, "/ia/qualifications", nil)
	req = req.WithContext(ctxWithAuth("org1", "user1"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
