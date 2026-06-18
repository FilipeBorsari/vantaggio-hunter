package exports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/exports"
)

// ── mock service ──────────────────────────────────────────────────────────────

type mockSvc struct {
	createIntegrationFn func(ctx context.Context, orgID, crmType, baseURL, apiKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error)
	getIntegrationFn    func(ctx context.Context, orgID string) (*domain.CRMIntegration, error)
	createExportFn      func(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error)
	getExportFn         func(ctx context.Context, id, orgID string) (*domain.ExportJob, error)
	listExportsFn       func(ctx context.Context, orgID string, page, limit int) (*domain.ExportListResponse, error)
}

func (m *mockSvc) CreateIntegration(ctx context.Context, orgID, crmType, baseURL, apiKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error) {
	return m.createIntegrationFn(ctx, orgID, crmType, baseURL, apiKey, inboxID, accountID)
}
func (m *mockSvc) GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error) {
	return m.getIntegrationFn(ctx, orgID)
}
func (m *mockSvc) CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error) {
	return m.createExportFn(ctx, orgID, userID, searchID, cnpjs)
}
func (m *mockSvc) GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error) {
	return m.getExportFn(ctx, id, orgID)
}
func (m *mockSvc) ListExports(ctx context.Context, orgID string, page, limit int) (*domain.ExportListResponse, error) {
	return m.listExportsFn(ctx, orgID, page, limit)
}

// ── mock queue ────────────────────────────────────────────────────────────────

type mockQueue struct{ pushed []string }

func (m *mockQueue) RPush(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	for _, v := range values {
		if s, ok := v.(string); ok {
			m.pushed = append(m.pushed, s)
		}
	}
	return redis.NewIntCmd(context.Background())
}

// ── helpers ───────────────────────────────────────────────────────────────────

func withAuth(r *http.Request, orgID, userID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), authpkg.ContextKeyOrgID, orgID)
	ctx = context.WithValue(ctx, authpkg.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, authpkg.ContextKeyRole, role)
	return r.WithContext(ctx)
}

// ── GetIntegration ────────────────────────────────────────────────────────────

func TestHandler_GetIntegration_NotFound(t *testing.T) {
	svc := &mockSvc{
		getIntegrationFn: func(_ context.Context, _ string) (*domain.CRMIntegration, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	req := withAuth(httptest.NewRequest(http.MethodGet, "/crm/integrations", nil), "org-1", "u-1", "manager")
	rec := httptest.NewRecorder()
	h.GetIntegration(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_GetIntegration_OK(t *testing.T) {
	svc := &mockSvc{
		getIntegrationFn: func(_ context.Context, orgID string) (*domain.CRMIntegration, error) {
			return &domain.CRMIntegration{ID: "i-1", CRMType: "chatwoot", BaseURL: "https://chat.example.com", IsActive: true}, nil
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	req := withAuth(httptest.NewRequest(http.MethodGet, "/crm/integrations", nil), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.GetIntegration(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var intg domain.CRMIntegration
	if err := json.NewDecoder(rec.Body).Decode(&intg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if intg.ID != "i-1" {
		t.Errorf("wrong id: %s", intg.ID)
	}
}

// ── CreateIntegration ─────────────────────────────────────────────────────────

func TestHandler_CreateIntegration_ForbiddenForOperator(t *testing.T) {
	h := exports.NewHandler(&mockSvc{}, &mockQueue{})

	body := `{"crm_type":"chatwoot","base_url":"https://x.com","api_key":"key123"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/crm/integrations", bytes.NewBufferString(body)), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.CreateIntegration(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestHandler_CreateIntegration_MissingFields(t *testing.T) {
	h := exports.NewHandler(&mockSvc{}, &mockQueue{})

	body := `{"crm_type":"chatwoot"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/crm/integrations", bytes.NewBufferString(body)), "org-1", "u-1", "admin")
	rec := httptest.NewRecorder()
	h.CreateIntegration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_CreateIntegration_OK(t *testing.T) {
	svc := &mockSvc{
		createIntegrationFn: func(_ context.Context, orgID, crmType, baseURL, apiKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error) {
			return &domain.CRMIntegration{ID: "i-new", CRMType: crmType, BaseURL: baseURL, IsActive: true}, nil
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	body := `{"crm_type":"chatwoot","base_url":"https://chat.example.com","api_key":"key123","account_id":1}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/crm/integrations", bytes.NewBufferString(body)), "org-1", "u-1", "manager")
	rec := httptest.NewRecorder()
	h.CreateIntegration(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ── CreateExport ──────────────────────────────────────────────────────────────

func TestHandler_CreateExport_EmptyBody(t *testing.T) {
	h := exports.NewHandler(&mockSvc{}, &mockQueue{})

	req := withAuth(httptest.NewRequest(http.MethodPost, "/exports", bytes.NewBufferString("{}")), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.CreateExport(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_CreateExport_NoCRM(t *testing.T) {
	svc := &mockSvc{
		createExportFn: func(_ context.Context, _, _ string, _ *string, _ []string) (*domain.ExportJob, error) {
			return nil, exports.ErrNoCRMIntegration
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	body := `{"cnpjs":["12345678000100"]}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/exports", bytes.NewBufferString(body)), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.CreateExport(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_CreateExport_InsufficientCredits(t *testing.T) {
	svc := &mockSvc{
		createExportFn: func(_ context.Context, _, _ string, _ *string, _ []string) (*domain.ExportJob, error) {
			return nil, domain.ErrInsufficientCredits
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	body := `{"cnpjs":["12345678000100"]}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/exports", bytes.NewBufferString(body)), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.CreateExport(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402, got %d", rec.Code)
	}
}

func TestHandler_CreateExport_OK_EnqueuesID(t *testing.T) {
	queue := &mockQueue{}
	svc := &mockSvc{
		createExportFn: func(_ context.Context, _, _ string, _ *string, cnpjs []string) (*domain.ExportJob, error) {
			return &domain.ExportJob{
				ID: "exp-42", Status: domain.ExportStatusPending, TotalCount: len(cnpjs),
			}, nil
		},
	}
	h := exports.NewHandler(svc, queue)

	body := `{"cnpjs":["12345678000100","98765432000199"]}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/exports", bytes.NewBufferString(body)), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.CreateExport(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Confirm the export ID was pushed to the Redis queue.
	if len(queue.pushed) == 0 || queue.pushed[0] != "exp-42" {
		t.Errorf("expected exp-42 in queue, got %v", queue.pushed)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["export_id"] != "exp-42" {
		t.Errorf("wrong export_id in response: %v", resp["export_id"])
	}
}

// ── ListExports ───────────────────────────────────────────────────────────────

func TestHandler_ListExports_OK(t *testing.T) {
	svc := &mockSvc{
		listExportsFn: func(_ context.Context, _ string, _, _ int) (*domain.ExportListResponse, error) {
			return &domain.ExportListResponse{
				Data:  []domain.ExportJob{{ID: "exp-1", Status: domain.ExportStatusDone}},
				Total: 1,
			}, nil
		},
	}
	h := exports.NewHandler(svc, &mockQueue{})

	req := withAuth(httptest.NewRequest(http.MethodGet, "/exports", nil), "org-1", "u-1", "operator")
	rec := httptest.NewRecorder()
	h.ListExports(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp domain.ExportListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}
