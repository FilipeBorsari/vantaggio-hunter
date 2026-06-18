package analytics_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vantaggio/prospect-api/internal/analytics"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// ─── Mock Repository ─────────────────────────────────────────────────────────

type mockRepo struct {
	getKPIsFn            func(ctx context.Context, orgID string, from, to time.Time) (*domain.AnalyticsKPIs, error)
	getDailyFn           func(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error)
	getTopCNAEsFn        func(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error)
	getFunnelFn          func(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error)
	runETLFn             func(ctx context.Context) error
}

func (m *mockRepo) GetKPIs(ctx context.Context, orgID string, from, to time.Time) (*domain.AnalyticsKPIs, error) {
	return m.getKPIsFn(ctx, orgID, from, to)
}
func (m *mockRepo) GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error) {
	return m.getDailyFn(ctx, orgID, from, to)
}
func (m *mockRepo) GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error) {
	return m.getTopCNAEsFn(ctx, orgID, from, to, limit)
}
func (m *mockRepo) GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error) {
	return m.getFunnelFn(ctx, orgID, from, to)
}
func (m *mockRepo) RunETL(ctx context.Context) error {
	return m.runETLFn(ctx)
}

// ─── ParsePeriod ─────────────────────────────────────────────────────────────

func TestParsePeriod_7d(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=7d", nil)
	label, from, to, err := analytics.ParsePeriod(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != "7d" {
		t.Errorf("expected label 7d, got %s", label)
	}
	diff := to.Sub(from)
	if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
		t.Errorf("expected ~7 day range, got %v", diff)
	}
}

func TestParsePeriod_30d_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil) // no period param
	label, from, to, err := analytics.ParsePeriod(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != "30d" {
		t.Errorf("expected label 30d, got %s", label)
	}
	diff := to.Sub(from)
	if diff < 29*24*time.Hour || diff > 31*24*time.Hour {
		t.Errorf("expected ~30 day range, got %v", diff)
	}
}

func TestParsePeriod_90d(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=90d", nil)
	label, from, to, err := analytics.ParsePeriod(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != "90d" {
		t.Errorf("expected label 90d, got %s", label)
	}
	diff := to.Sub(from)
	if diff < 89*24*time.Hour || diff > 91*24*time.Hour {
		t.Errorf("expected ~90 day range, got %v", diff)
	}
}

func TestParsePeriod_Custom_OK(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=custom&from=2026-01-01&to=2026-01-31", nil)
	label, from, to, err := analytics.ParsePeriod(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != "custom" {
		t.Errorf("expected label custom, got %s", label)
	}
	if from.Year() != 2026 || from.Month() != 1 || from.Day() != 1 {
		t.Errorf("unexpected from: %v", from)
	}
	// to é retornado como meia-noite do último dia; tempoID extrai apenas a data
	if to.Year() != 2026 || to.Month() != 1 || to.Day() != 31 {
		t.Errorf("unexpected to: %v", to)
	}
	// sanity: tempoID(to) deve ser 20260131, não 20260201
	if to.After(from.AddDate(0, 1, 0)) {
		t.Errorf("to ultrapassou o fim do período: %v", to)
	}
}

func TestParsePeriod_Custom_InvalidFrom(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=custom&from=not-a-date&to=2026-01-31", nil)
	_, _, _, err := analytics.ParsePeriod(req)
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
}

func TestParsePeriod_Custom_InvalidTo(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=custom&from=2026-01-01&to=oops", nil)
	_, _, _, err := analytics.ParsePeriod(req)
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
}

func TestParsePeriod_Custom_ReversedDates(t *testing.T) {
	req := httptest.NewRequest("GET", "/?period=custom&from=2026-06-17&to=2026-06-01", nil)
	_, _, _, err := analytics.ParsePeriod(req)
	if err == nil {
		t.Fatal("expected error when from > to")
	}
}

// ─── GetKPIs ─────────────────────────────────────────────────────────────────

func TestService_GetKPIs_OK(t *testing.T) {
	repo := &mockRepo{
		getKPIsFn: func(_ context.Context, orgID string, _, _ time.Time) (*domain.AnalyticsKPIs, error) {
			return &domain.AnalyticsKPIs{
				CreditsConsumed:  1000,
				CreditsPurchased: 5000,
				LeadsExtracted:   1000,
				LeadsExported:    50,
				SearchesCount:    5,
			}, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	kpis, err := svc.GetKPIs(context.Background(), "org-1", "30d", now.AddDate(0, 0, -30), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kpis.Period != "30d" {
		t.Errorf("expected period 30d, got %s", kpis.Period)
	}
	if kpis.CreditsConsumed != 1000 {
		t.Errorf("expected 1000 credits consumed, got %d", kpis.CreditsConsumed)
	}
	// conversion rate = 50/1000 = 0.05
	if kpis.ConversionRate < 0.049 || kpis.ConversionRate > 0.051 {
		t.Errorf("expected conversion rate ~0.05, got %f", kpis.ConversionRate)
	}
}

func TestService_GetKPIs_ZeroLeads_NoConversion(t *testing.T) {
	repo := &mockRepo{
		getKPIsFn: func(_ context.Context, _ string, _, _ time.Time) (*domain.AnalyticsKPIs, error) {
			return &domain.AnalyticsKPIs{LeadsExtracted: 0, LeadsExported: 0}, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	kpis, err := svc.GetKPIs(context.Background(), "org-1", "7d", now.AddDate(0, 0, -7), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kpis.ConversionRate != 0 {
		t.Errorf("expected 0 conversion rate when no leads, got %f", kpis.ConversionRate)
	}
}

func TestService_GetKPIs_RepoError(t *testing.T) {
	repo := &mockRepo{
		getKPIsFn: func(_ context.Context, _ string, _, _ time.Time) (*domain.AnalyticsKPIs, error) {
			return nil, errors.New("db down")
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	_, err := svc.GetKPIs(context.Background(), "org-1", "30d", now.AddDate(0, 0, -30), now)
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// ─── GetDailyConsumption ─────────────────────────────────────────────────────

func TestService_GetDailyConsumption_OK(t *testing.T) {
	repo := &mockRepo{
		getDailyFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPoint, error) {
			return []domain.DailyPoint{
				{Date: "2026-06-01", Credits: 100, Leads: 100},
			}, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	points, err := svc.GetDailyConsumption(context.Background(), "org-1", now.AddDate(0, 0, -7), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 1 {
		t.Errorf("expected 1 point, got %d", len(points))
	}
}

func TestService_GetDailyConsumption_NilReturnsEmpty(t *testing.T) {
	repo := &mockRepo{
		getDailyFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.DailyPoint, error) {
			return nil, nil // repo returns nil when no data
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	points, err := svc.GetDailyConsumption(context.Background(), "org-1", now.AddDate(0, 0, -7), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if points == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(points) != 0 {
		t.Errorf("expected empty slice, got %d items", len(points))
	}
}

// ─── GetTopCNAEs ─────────────────────────────────────────────────────────────

func TestService_GetTopCNAEs_OK(t *testing.T) {
	var capturedLimit int
	repo := &mockRepo{
		getTopCNAEsFn: func(_ context.Context, _ string, _, _ time.Time, limit int) ([]domain.TopCNAE, error) {
			capturedLimit = limit
			return []domain.TopCNAE{{CNAECode: "4711-3/01", Description: "Comércio varejista", Leads: 500}}, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	results, err := svc.GetTopCNAEs(context.Background(), "org-1", now.AddDate(0, 0, -30), now, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 5 {
		t.Errorf("expected limit 5 passed to repo, got %d", capturedLimit)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestService_GetTopCNAEs_LimitZeroDefaultsTen(t *testing.T) {
	var capturedLimit int
	repo := &mockRepo{
		getTopCNAEsFn: func(_ context.Context, _ string, _, _ time.Time, limit int) ([]domain.TopCNAE, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	_, _ = svc.GetTopCNAEs(context.Background(), "org-1", now.AddDate(0, 0, -30), now, 0)
	if capturedLimit != 10 {
		t.Errorf("expected limit clamped to 10 for 0, got %d", capturedLimit)
	}
}

func TestService_GetTopCNAEs_LimitAbove50ClampsTo50(t *testing.T) {
	var capturedLimit int
	repo := &mockRepo{
		getTopCNAEsFn: func(_ context.Context, _ string, _, _ time.Time, limit int) ([]domain.TopCNAE, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	_, _ = svc.GetTopCNAEs(context.Background(), "org-1", now.AddDate(0, 0, -30), now, 99)
	if capturedLimit != 50 {
		t.Errorf("expected limit clamped to max 50 for 99, got %d", capturedLimit)
	}
}

func TestService_GetTopCNAEs_NilReturnsEmpty(t *testing.T) {
	repo := &mockRepo{
		getTopCNAEsFn: func(_ context.Context, _ string, _, _ time.Time, _ int) ([]domain.TopCNAE, error) {
			return nil, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	results, err := svc.GetTopCNAEs(context.Background(), "org-1", now.AddDate(0, 0, -30), now, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
}

// ─── GetFunnel ───────────────────────────────────────────────────────────────

func TestService_GetFunnel_OK(t *testing.T) {
	repo := &mockRepo{
		getFunnelFn: func(_ context.Context, _ string, _, _ time.Time) (*domain.FunnelResponse, error) {
			return &domain.FunnelResponse{
				Stages: []domain.FunnelStage{
					{Name: "Extraídos", Count: 1000},
					{Name: "Qualificados", Count: 50},
					{Name: "Exportados", Count: 20},
				},
			}, nil
		},
	}
	svc := analytics.NewService(repo)
	now := time.Now()
	funnel, err := svc.GetFunnel(context.Background(), "org-1", now.AddDate(0, 0, -30), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(funnel.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(funnel.Stages))
	}
	if funnel.Stages[0].Count != 1000 {
		t.Errorf("expected 1000 extraídos, got %d", funnel.Stages[0].Count)
	}
}

// ─── RunETL ──────────────────────────────────────────────────────────────────

func TestService_RunETL_OK(t *testing.T) {
	called := false
	repo := &mockRepo{
		runETLFn: func(_ context.Context) error {
			called = true
			return nil
		},
	}
	svc := analytics.NewService(repo)
	if err := svc.RunETL(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.RunETL not called")
	}
}

func TestService_RunETL_Error(t *testing.T) {
	repo := &mockRepo{
		runETLFn: func(_ context.Context) error {
			return errors.New("etl failure")
		},
	}
	svc := analytics.NewService(repo)
	if err := svc.RunETL(context.Background()); err == nil {
		t.Fatal("expected error from ETL")
	}
}
