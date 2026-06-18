package exports_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/exports"
)

// ── mock repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	saveIntegrationFn   func(ctx context.Context, orgID, crmType, baseURL, encAPIKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error)
	getIntegrationFn    func(ctx context.Context, orgID string) (*domain.CRMIntegration, error)
	getIntegrationRawFn func(ctx context.Context, orgID string) (string, string, *int, int, error)
	createExportFn      func(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error)
	getExportFn         func(ctx context.Context, id, orgID string) (*domain.ExportJob, error)
	listExportsFn       func(ctx context.Context, orgID string, page, limit int) ([]domain.ExportJob, int, error)
	markProcessingFn    func(ctx context.Context, id string) (int, error)
	updateResultFn      func(ctx context.Context, id string, status domain.ExportStatus, ok, fail int, log []domain.ExportErrorEntry, retry *time.Time) error
	incrFunnelFn        func(ctx context.Context, searchID, orgID string, count int) error
}

func (m *mockRepo) SaveIntegration(ctx context.Context, orgID, crmType, baseURL, encAPIKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error) {
	return m.saveIntegrationFn(ctx, orgID, crmType, baseURL, encAPIKey, inboxID, accountID)
}
func (m *mockRepo) GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error) {
	return m.getIntegrationFn(ctx, orgID)
}
func (m *mockRepo) GetIntegrationRaw(ctx context.Context, orgID string) (string, string, *int, int, error) {
	return m.getIntegrationRawFn(ctx, orgID)
}
func (m *mockRepo) CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error) {
	return m.createExportFn(ctx, orgID, userID, searchID, cnpjs)
}
func (m *mockRepo) GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error) {
	return m.getExportFn(ctx, id, orgID)
}
func (m *mockRepo) ListExports(ctx context.Context, orgID string, page, limit int) ([]domain.ExportJob, int, error) {
	return m.listExportsFn(ctx, orgID, page, limit)
}
func (m *mockRepo) MarkProcessing(ctx context.Context, id string) (int, error) {
	return m.markProcessingFn(ctx, id)
}
func (m *mockRepo) UpdateExportResult(ctx context.Context, id string, status domain.ExportStatus, ok, fail int, log []domain.ExportErrorEntry, retry *time.Time) error {
	return m.updateResultFn(ctx, id, status, ok, fail, log, retry)
}
func (m *mockRepo) IncrFunnelExported(ctx context.Context, searchID, orgID string, count int) error {
	return m.incrFunnelFn(ctx, searchID, orgID, count)
}

// ── mock credit service ───────────────────────────────────────────────────────

type mockCreditSvc struct {
	getBalanceFn func(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error)
}

func (m *mockCreditSvc) GetBalance(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	return m.getBalanceFn(ctx, orgID)
}
func (m *mockCreditSvc) Deduct(_ context.Context, _ pgx.Tx, _, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error {
	return nil
}
func (m *mockCreditSvc) BeginTx(_ context.Context) (pgx.Tx, error)   { return nil, nil }
func (m *mockCreditSvc) AddCredits(_ context.Context, _ string, _ int, _ string) error { return nil }
func (m *mockCreditSvc) ListTransactions(_ context.Context, _ string, _, _ int) (*domain.CreditTransactionsResponse, error) {
	return nil, nil
}

var _ credits.ServiceInterface = (*mockCreditSvc)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestKey() []byte { return make([]byte, 32) } // 32 zero bytes = valid AES-256 key

func integrationFound() func(ctx context.Context, orgID string) (*domain.CRMIntegration, error) {
	return func(_ context.Context, _ string) (*domain.CRMIntegration, error) {
		return &domain.CRMIntegration{ID: "intg-1", CRMType: "chatwoot", IsActive: true}, nil
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestService_CreateIntegration_OK(t *testing.T) {
	saved := false
	repo := &mockRepo{
		saveIntegrationFn: func(_ context.Context, orgID, crmType, baseURL, encAPIKey string, _ *int, _ int) (*domain.CRMIntegration, error) {
			saved = true
			if orgID != "org-1" {
				t.Errorf("wrong orgID: %s", orgID)
			}
			if encAPIKey == "raw-api-key" {
				t.Error("api_key was not encrypted before saving")
			}
			return &domain.CRMIntegration{ID: "i-1", CRMType: crmType, BaseURL: baseURL, IsActive: true}, nil
		},
	}
	creditSvc := &mockCreditSvc{}
	svc := exports.NewService(repo, creditSvc, newTestKey())

	intg, err := svc.CreateIntegration(context.Background(), "org-1", "chatwoot", "https://app.chatwoot.com", "raw-api-key", nil, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !saved {
		t.Error("repo.SaveIntegration was not called")
	}
	if intg.ID != "i-1" {
		t.Errorf("wrong id: %s", intg.ID)
	}
}

func TestService_GetIntegration_NotFound(t *testing.T) {
	repo := &mockRepo{
		getIntegrationFn: func(_ context.Context, _ string) (*domain.CRMIntegration, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := exports.NewService(repo, &mockCreditSvc{}, newTestKey())

	_, err := svc.GetIntegration(context.Background(), "org-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_CreateExport_EmptyCNPJs(t *testing.T) {
	svc := exports.NewService(&mockRepo{}, &mockCreditSvc{}, newTestKey())

	_, err := svc.CreateExport(context.Background(), "org-1", "user-1", nil, []string{})
	if err == nil {
		t.Fatal("expected error for empty CNPJ list")
	}
}

func TestService_CreateExport_TooManyCNPJs(t *testing.T) {
	cnpjs := make([]string, 501)
	for i := range cnpjs {
		cnpjs[i] = "12345678000100"
	}
	svc := exports.NewService(&mockRepo{}, &mockCreditSvc{}, newTestKey())

	_, err := svc.CreateExport(context.Background(), "org-1", "user-1", nil, cnpjs)
	if err == nil {
		t.Fatal("expected error for > 500 CNPJs")
	}
}

func TestService_CreateExport_NoCRMIntegration(t *testing.T) {
	repo := &mockRepo{
		getIntegrationFn: func(_ context.Context, _ string) (*domain.CRMIntegration, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := exports.NewService(repo, &mockCreditSvc{}, newTestKey())

	_, err := svc.CreateExport(context.Background(), "org-1", "user-1", nil, []string{"12345678000100"})
	if !errors.Is(err, exports.ErrNoCRMIntegration) {
		t.Errorf("expected ErrNoCRMIntegration, got %v", err)
	}
}

func TestService_CreateExport_InsufficientCredits(t *testing.T) {
	repo := &mockRepo{getIntegrationFn: integrationFound()}
	creditSvc := &mockCreditSvc{
		getBalanceFn: func(_ context.Context, _ string) (*domain.CreditBalanceResponse, error) {
			return &domain.CreditBalanceResponse{Balance: 0}, nil
		},
	}
	svc := exports.NewService(repo, creditSvc, newTestKey())

	_, err := svc.CreateExport(context.Background(), "org-1", "user-1", nil, []string{"12345678000100"})
	if !errors.Is(err, domain.ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got %v", err)
	}
}

func TestService_CreateExport_OK(t *testing.T) {
	created := false
	cnpjs := []string{"12345678000100", "98765432000199"}
	repo := &mockRepo{
		getIntegrationFn: integrationFound(),
		createExportFn: func(_ context.Context, orgID, userID string, _ *string, got []string) (*domain.ExportJob, error) {
			created = true
			if len(got) != 2 {
				t.Errorf("expected 2 cnpjs, got %d", len(got))
			}
			return &domain.ExportJob{
				ID: "exp-1", OrgID: orgID, Status: domain.ExportStatusPending,
				TotalCount: len(got), CNPJs: got,
			}, nil
		},
	}
	creditSvc := &mockCreditSvc{
		getBalanceFn: func(_ context.Context, _ string) (*domain.CreditBalanceResponse, error) {
			return &domain.CreditBalanceResponse{Balance: 10}, nil
		},
	}
	svc := exports.NewService(repo, creditSvc, newTestKey())

	job, err := svc.CreateExport(context.Background(), "org-1", "user-1", nil, cnpjs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("repo.CreateExport was not called")
	}
	if job.Status != domain.ExportStatusPending {
		t.Errorf("expected pending, got %s", job.Status)
	}
}

func TestService_ListExports_EmptyReturnsSlice(t *testing.T) {
	repo := &mockRepo{
		listExportsFn: func(_ context.Context, _ string, _, _ int) ([]domain.ExportJob, int, error) {
			return nil, 0, nil
		},
	}
	svc := exports.NewService(repo, &mockCreditSvc{}, newTestKey())

	resp, err := svc.ListExports(context.Background(), "org-1", 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestService_GetExport_NotFound(t *testing.T) {
	repo := &mockRepo{
		getExportFn: func(_ context.Context, _, _ string) (*domain.ExportJob, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := exports.NewService(repo, &mockCreditSvc{}, newTestKey())

	_, err := svc.GetExport(context.Background(), "exp-999", "org-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
