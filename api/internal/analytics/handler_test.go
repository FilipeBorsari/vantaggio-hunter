package analytics_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/analytics"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// ─── Mock Service ─────────────────────────────────────────────────────────────

type mockSvc struct {
	getKPIsFn   func(ctx context.Context, orgID, period string, from, to time.Time) (*domain.AnalyticsKPIs, error)
	getDailyFn  func(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error)
	getTopFn    func(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error)
	getFunnelFn func(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error)
	runETLFn    func(ctx context.Context) error
}

func (m *mockSvc) GetKPIs(ctx context.Context, orgID, period string, from, to time.Time) (*domain.AnalyticsKPIs, error) {
	return m.getKPIsFn(ctx, orgID, period, from, to)
}
func (m *mockSvc) GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error) {
	return m.getDailyFn(ctx, orgID, from, to)
}
func (m *mockSvc) GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error) {
	return m.getTopFn(ctx, orgID, from, to, limit)
}
func (m *mockSvc) GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error) {
	return m.getFunnelFn(ctx, orgID, from, to)
}
func (m *mockSvc) RunETL(ctx context.Context) error {
	return m.runETLFn(ctx)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func requestWithOrg(method, target, orgID string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), authpkg.ContextKeyOrgID, orgID)
	return req.WithContext(ctx)
}

// ─── GetKPIs ─────────────────────────────────────────────────────────────────

func TestHandler_GetKPIs_OK(t *testing.T) {
	svc := &mockSvc{
		getKPIsFn: func(_ context.Context, orgID, period string, _, _ time.Time) (*domain.AnalyticsKPIs, error) {
			return &domain.AnalyticsKPIs{
				Period:          period,
				CreditsConsumed: 2000,
				LeadsExtracted:  2000,
				SearchesCount:   8,
			}, nil
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetKPIs(rec, requestWithOrg(http.MethodGet, "/analytics/kpis?period=30d", "org-1"))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp domain.AnalyticsKPIs
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CreditsConsumed != 2000 {
		t.Errorf("expected 2000 credits consumed, got %d", resp.CreditsConsumed)
	}
	if resp.Period != "30d" {
		t.Errorf("expected period 30d, got %s", resp.Period)
	}
}

func TestHandler_GetKPIs_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getKPIsFn: func(_ context.Context, _, _ string, _, _ time.Time) (*domain.AnalyticsKPIs, error) {
			return nil, errors.New("db error")
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetKPIs(rec, requestWithOrg(http.MethodGet, "/analytics/kpis", "org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandler_GetKPIs_InvalidCustomPeriod(t *testing.T) {
	svc := &mockSvc{}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetKPIs(rec, requestWithOrg(http.MethodGet, "/analytics/kpis?period=custom&from=bad&to=also-bad", "org-1"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetKPIs_ReversedCustomPeriod(t *testing.T) {
	svc := &mockSvc{}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetKPIs(rec, requestWithOrg(http.MethodGet, "/analytics/kpis?period=custom&from=2026-06-17&to=2026-06-01", "org-1"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for reversed dates, got %d", rec.Code)
	}
}

func TestHandler_GetKPIs_MissingOrgID(t *testing.T) {
	svc := &mockSvc{}
	h := analytics.NewHandler(svc)

	// Request sem org_id no contexto (simula ausência do middleware)
	req := httptest.NewRequest(http.MethodGet, "/analytics/kpis", nil)
	rec := httptest.NewRecorder()
	h.GetKPIs(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when orgID missing, got %d", rec.Code)
	}
}

// ─── GetDailyConsumption ─────────────────────────────────────────────────────

func TestHandler_GetDailyConsumption_OK(t *testing.T) {
	svc := &mockSvc{
		getDailyFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPoint, error) {
			return []domain.DailyPoint{
				{Date: "2026-06-01", Credits: 150, Leads: 150},
				{Date: "2026-06-02", Credits: 200, Leads: 200},
			}, nil
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetDailyConsumption(rec, requestWithOrg(http.MethodGet, "/analytics/daily-consumption?period=7d", "org-1"))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp []domain.DailyPoint
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 points, got %d", len(resp))
	}
}

func TestHandler_GetDailyConsumption_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getDailyFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPoint, error) {
			return nil, errors.New("db error")
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetDailyConsumption(rec, requestWithOrg(http.MethodGet, "/analytics/daily-consumption", "org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ─── GetTopCNAEs ─────────────────────────────────────────────────────────────

func TestHandler_GetTopCNAEs_OK(t *testing.T) {
	svc := &mockSvc{
		getTopFn: func(_ context.Context, _ string, _, _ time.Time, limit int) ([]domain.TopCNAE, error) {
			return []domain.TopCNAE{
				{CNAECode: "4711-3/01", Description: "Supermercados", Leads: 800},
			}, nil
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetTopCNAEs(rec, requestWithOrg(http.MethodGet, "/analytics/top-cnaes?period=30d&limit=10", "org-1"))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp []domain.TopCNAE
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 cnae, got %d", len(resp))
	}
	if resp[0].Leads != 800 {
		t.Errorf("expected 800 leads, got %d", resp[0].Leads)
	}
}

func TestHandler_GetTopCNAEs_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getTopFn: func(_ context.Context, _ string, _, _ time.Time, _ int) ([]domain.TopCNAE, error) {
			return nil, errors.New("db error")
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetTopCNAEs(rec, requestWithOrg(http.MethodGet, "/analytics/top-cnaes", "org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ─── GetFunnel ───────────────────────────────────────────────────────────────

func TestHandler_GetFunnel_OK(t *testing.T) {
	svc := &mockSvc{
		getFunnelFn: func(_ context.Context, _ string, _, _ time.Time) (*domain.FunnelResponse, error) {
			return &domain.FunnelResponse{
				Stages: []domain.FunnelStage{
					{Name: "Extraídos", Count: 5000},
					{Name: "Qualificados", Count: 0},
					{Name: "Exportados", Count: 0},
				},
			}, nil
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetFunnel(rec, requestWithOrg(http.MethodGet, "/analytics/funnel?period=30d", "org-1"))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp domain.FunnelResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(resp.Stages))
	}
	if resp.Stages[0].Count != 5000 {
		t.Errorf("expected 5000 extraídos, got %d", resp.Stages[0].Count)
	}
	// leads_qualificados and leads_exportados are 0 until STEP-07/06
	if resp.Stages[1].Count != 0 {
		t.Errorf("expected 0 qualificados (STEP-07 not done), got %d", resp.Stages[1].Count)
	}
}

func TestHandler_GetFunnel_ServiceError(t *testing.T) {
	svc := &mockSvc{
		getFunnelFn: func(_ context.Context, _ string, _, _ time.Time) (*domain.FunnelResponse, error) {
			return nil, errors.New("db error")
		},
	}
	h := analytics.NewHandler(svc)

	rec := httptest.NewRecorder()
	h.GetFunnel(rec, requestWithOrg(http.MethodGet, "/analytics/funnel", "org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
