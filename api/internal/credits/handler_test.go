package credits_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type mockSvc struct {
	getBalanceFn      func(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error)
	deductFn          func(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error
	beginTxFn         func(ctx context.Context) (pgx.Tx, error)
	addCreditsFn      func(ctx context.Context, orgID string, amount int, desc string) error
	listTransactionsFn func(ctx context.Context, orgID string, page, limit int) (*domain.CreditTransactionsResponse, error)
}

func (m *mockSvc) GetBalance(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	return m.getBalanceFn(ctx, orgID)
}
func (m *mockSvc) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error {
	return m.deductFn(ctx, tx, orgID, userID, amount, txType, referenceID, desc)
}
func (m *mockSvc) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return m.beginTxFn(ctx)
}
func (m *mockSvc) AddCredits(ctx context.Context, orgID string, amount int, desc string) error {
	return m.addCreditsFn(ctx, orgID, amount, desc)
}
func (m *mockSvc) ListTransactions(ctx context.Context, orgID string, page, limit int) (*domain.CreditTransactionsResponse, error) {
	return m.listTransactionsFn(ctx, orgID, page, limit)
}

func TestHandler_GetBalance_OK(t *testing.T) {
	svc := &mockSvc{
		getBalanceFn: func(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
			return &domain.CreditBalanceResponse{Balance: 250, OrgID: "org-1"}, nil
		},
	}
	h := credits.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/credits/balance", nil)
	rec := httptest.NewRecorder()

	h.GetBalance(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp domain.CreditBalanceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Balance != 250 {
		t.Errorf("expected 250, got %d", resp.Balance)
	}
}

func TestHandler_AdminAddCredits_InvalidBody(t *testing.T) {
	svc := &mockSvc{}
	h := credits.NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/admin/credits/add", bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()

	h.AdminAddCredits(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_AdminAddCredits_MissingOrgID(t *testing.T) {
	svc := &mockSvc{}
	h := credits.NewHandler(svc)

	body := `{"amount": 500}`
	req := httptest.NewRequest(http.MethodPost, "/admin/credits/add", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.AdminAddCredits(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_AdminAddCredits_NegativeAmount(t *testing.T) {
	svc := &mockSvc{}
	h := credits.NewHandler(svc)

	body := `{"org_id":"org-1","amount":-10}`
	req := httptest.NewRequest(http.MethodPost, "/admin/credits/add", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.AdminAddCredits(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_AdminAddCredits_OK(t *testing.T) {
	svc := &mockSvc{
		addCreditsFn: func(ctx context.Context, orgID string, amount int, desc string) error {
			return nil
		},
	}
	h := credits.NewHandler(svc)

	body := `{"org_id":"org-1","amount":1000,"description":"Compra plano Pro"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/credits/add", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.AdminAddCredits(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}
