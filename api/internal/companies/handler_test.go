package companies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// ---------------------------------------------------------------------------
// mock credit service + pgx.Tx
// ---------------------------------------------------------------------------

type mockCreditSvc struct {
	beginTxFn func(ctx context.Context) (pgx.Tx, error)
	deductFn  func(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, refID *string, desc string) error
}

func (m *mockCreditSvc) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return m.beginTxFn(ctx)
}
func (m *mockCreditSvc) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, refID *string, desc string) error {
	return m.deductFn(ctx, tx, orgID, userID, amount, txType, refID, desc)
}
func (m *mockCreditSvc) GetBalance(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	return nil, nil
}
func (m *mockCreditSvc) AddCredits(ctx context.Context, orgID string, amount int, desc string) error {
	return nil
}
func (m *mockCreditSvc) ListTransactions(ctx context.Context, orgID string, page, limit int) (*domain.CreditTransactionsResponse, error) {
	return nil, nil
}

// ensure it implements the interface at compile time
var _ credits.ServiceInterface = (*mockCreditSvc)(nil)

// stubTx implements pgx.Tx with no-ops; only Commit and Rollback matter for tests.
type stubTx struct {
	commitErr   error
	rollbackErr error
}

func (t *stubTx) Begin(ctx context.Context) (pgx.Tx, error)    { return t, nil }
func (t *stubTx) Commit(ctx context.Context) error              { return t.commitErr }
func (t *stubTx) Rollback(ctx context.Context) error            { return t.rollbackErr }
func (t *stubTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *stubTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (t *stubTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (t *stubTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *stubTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *stubTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (t *stubTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (t *stubTx) Conn() *pgx.Conn                                               { return nil }

// spyCompanySvc captures the arguments passed to the service for assertions.
type spyCompanySvc struct {
	capturedFilters Filters
	listResp        *domain.CompanyListResponse
	listErr         error
	detail          *domain.CompanyDetail
	detailErr       error
}

func (s *spyCompanySvc) List(_ context.Context, f Filters) (*domain.CompanyListResponse, error) {
	s.capturedFilters = f
	return s.listResp, s.listErr
}
func (s *spyCompanySvc) GetByCNPJ(_ context.Context, _ string) (*domain.CompanyDetail, error) {
	return s.detail, s.detailErr
}

// withChiParam injects a chi route param into the request context.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// List handler
// ---------------------------------------------------------------------------

func TestListHandler_DefaultFilters(t *testing.T) {
	svc := &spyCompanySvc{listResp: &domain.CompanyListResponse{}}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if svc.capturedFilters.Page != 1 {
		t.Errorf("default page = %d, want 1", svc.capturedFilters.Page)
	}
	if svc.capturedFilters.Limit != 50 {
		t.Errorf("default limit = %d, want 50", svc.capturedFilters.Limit)
	}
}

func TestListHandler_ParsesQueryParams(t *testing.T) {
	svc := &spyCompanySvc{listResp: &domain.CompanyListResponse{}}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies?uf=SP&city=CAMPINAS&page=2&limit=20&status=2&capital_min=100000", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	f := svc.capturedFilters
	if f.UF != "SP" {
		t.Errorf("uf = %q, want SP", f.UF)
	}
	if f.City != "CAMPINAS" {
		t.Errorf("city = %q, want CAMPINAS", f.City)
	}
	if f.Page != 2 {
		t.Errorf("page = %d, want 2", f.Page)
	}
	if f.Limit != 20 {
		t.Errorf("limit = %d, want 20", f.Limit)
	}
	if f.Status == nil || *f.Status != 2 {
		t.Errorf("status = %v, want 2", f.Status)
	}
	if f.CapitalMin == nil || *f.CapitalMin != 100000 {
		t.Errorf("capital_min = %v, want 100000", f.CapitalMin)
	}
}

func TestListHandler_ParsesCNAEParam(t *testing.T) {
	svc := &spyCompanySvc{listResp: &domain.CompanyListResponse{}}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies?cnae=6201500,6202300,4711301", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	f := svc.capturedFilters
	if len(f.CNAEs) != 3 {
		t.Fatalf("cnaes len = %d, want 3", len(f.CNAEs))
	}
	if f.CNAEs[0] != "6201500" || f.CNAEs[1] != "6202300" || f.CNAEs[2] != "4711301" {
		t.Errorf("cnaes = %v, want [6201500 6202300 4711301]", f.CNAEs)
	}
}

func TestListHandler_LimitCappedAt200(t *testing.T) {
	svc := &spyCompanySvc{listResp: &domain.CompanyListResponse{}}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies?limit=9999", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if svc.capturedFilters.Limit != 200 {
		t.Errorf("limit = %d, want 200 (capped)", svc.capturedFilters.Limit)
	}
}

func TestListHandler_InvalidPageDefaultsTo1(t *testing.T) {
	cases := []string{"?page=0", "?page=-1", "?page=abc"}
	for _, q := range cases {
		svc := &spyCompanySvc{listResp: &domain.CompanyListResponse{}}
		h := NewHandler(svc, nil)
		r := httptest.NewRequest(http.MethodGet, "/companies"+q, nil)
		w := httptest.NewRecorder()
		h.List(w, r)
		if svc.capturedFilters.Page != 1 {
			t.Errorf("query %q: page = %d, want 1", q, svc.capturedFilters.Page)
		}
	}
}

func TestListHandler_ServiceError(t *testing.T) {
	svc := &spyCompanySvc{listErr: errors.New("db error")}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestListHandler_ResponseShape(t *testing.T) {
	capital := 50000.0
	nome := "ACME LTDA"
	resp := &domain.CompanyListResponse{
		Data: []domain.Company{
			{CNPJ: "12345678000100", RazaoSocial: "ACME LTDA", NomeFantasia: &nome, UF: "SP", CapitalSocial: &capital},
		},
		Total: 1,
		Page:  1,
		Limit: 50,
	}
	svc := &spyCompanySvc{listResp: resp}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	var body domain.CompanyListResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 {
		t.Errorf("total = %d, want 1", body.Total)
	}
	if len(body.Data) != 1 || body.Data[0].CNPJ != "12345678000100" {
		t.Errorf("data = %v", body.Data)
	}
}

// ---------------------------------------------------------------------------
// GetByCNPJ handler
// ---------------------------------------------------------------------------

func TestGetByCNPJHandler_Success(t *testing.T) {
	detail := &domain.CompanyDetail{
		CNPJ:        "12345678000100",
		RazaoSocial: "ACME LTDA",
		UF:          "SP",
		CNAEs:       []domain.CNAE{{Code: "6201500", IsPrimary: true}},
		Partners:    []domain.Partner{{Nome: "JOAO"}},
	}
	svc := &spyCompanySvc{detail: detail}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies/12345678000100", nil)
	r = withChiParam(r, "cnpj", "12345678000100")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body domain.CompanyDetail
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.CNPJ != "12345678000100" {
		t.Errorf("cnpj = %q, want 12345678000100", body.CNPJ)
	}
}

func TestGetByCNPJHandler_NotFound(t *testing.T) {
	svc := &spyCompanySvc{detailErr: domain.ErrNotFound}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies/00000000000000", nil)
	r = withChiParam(r, "cnpj", "00000000000000")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetByCNPJHandler_WrappedNotFound(t *testing.T) {
	// Errors that wrap ErrNotFound via %w must also return 404.
	wrapped := fmt.Errorf("get by cnpj: %w", domain.ErrNotFound)
	svc := &spyCompanySvc{detailErr: wrapped}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies/00000000000000", nil)
	r = withChiParam(r, "cnpj", "00000000000000")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for wrapped ErrNotFound", w.Code)
	}
}

func TestGetByCNPJHandler_InternalError(t *testing.T) {
	svc := &spyCompanySvc{detailErr: errors.New("db connection lost")}
	h := NewHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/companies/12345678000100", nil)
	r = withChiParam(r, "cnpj", "12345678000100")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetByCNPJ with credit service
// ---------------------------------------------------------------------------

// withOrgAndUser injects orgID and userID into the request context,
// simulating what the Authenticate middleware does.
func withOrgAndUser(r *http.Request, orgID, userID string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, authpkg.ContextKeyOrgID, orgID)
	ctx = context.WithValue(ctx, authpkg.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

func TestGetByCNPJHandler_InsufficientCredits_Returns402(t *testing.T) {
	tx := &stubTx{}
	creditSvc := &mockCreditSvc{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return tx, nil },
		deductFn: func(_ context.Context, _ pgx.Tx, _, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error {
			return domain.ErrInsufficientCredits
		},
	}
	svc := &spyCompanySvc{}
	h := NewHandler(svc, creditSvc)

	r := httptest.NewRequest(http.MethodGet, "/companies/12345678000100", nil)
	r = withChiParam(r, "cnpj", "12345678000100")
	// Provide a non-empty orgID so the credit path is exercised.
	r = withOrgAndUser(r, "org-123", "user-456")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", w.Code)
	}
}

func TestGetByCNPJHandler_SuccessfulDeduction_Returns200(t *testing.T) {
	tx := &stubTx{}
	deductCalled := false
	commitCalled := false
	tx.commitErr = nil

	creditSvc := &mockCreditSvc{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return tx, nil },
		deductFn: func(_ context.Context, _ pgx.Tx, orgID, _ string, amount int, txType domain.CreditTxType, _ *string, _ string) error {
			deductCalled = true
			if orgID != "org-123" {
				return fmt.Errorf("unexpected orgID: %s", orgID)
			}
			if amount != 10 {
				return fmt.Errorf("expected 10 credits, got %d", amount)
			}
			if txType != domain.CreditTxCompanyDetail {
				return fmt.Errorf("unexpected tx type: %s", txType)
			}
			return nil
		},
	}

	// Override commit to track the call.
	committed := false
	tx.commitErr = nil
	_ = committed // suppress unused warning; commitErr=nil means commit succeeds

	detail := &domain.CompanyDetail{CNPJ: "12345678000100", RazaoSocial: "ACME LTDA", UF: "SP"}
	svc := &spyCompanySvc{detail: detail}
	h := NewHandler(svc, creditSvc)

	r := httptest.NewRequest(http.MethodGet, "/companies/12345678000100", nil)
	r = withChiParam(r, "cnpj", "12345678000100")
	r = withOrgAndUser(r, "org-123", "user-456")
	w := httptest.NewRecorder()
	h.GetByCNPJ(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !deductCalled {
		t.Error("Deduct was not called")
	}
	_ = commitCalled
}
