package credits_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type mockRepo struct {
	getBalanceFn      func(ctx context.Context, orgID string) (int, error)
	deductFn          func(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error
	addCreditsFn      func(ctx context.Context, orgID string, amount int, desc string) error
	listTxFn          func(ctx context.Context, orgID string, page, limit int) ([]domain.CreditTransaction, int, error)
	beginTxFn         func(ctx context.Context) (pgx.Tx, error)
}

func (m *mockRepo) GetBalance(ctx context.Context, orgID string) (int, error) {
	return m.getBalanceFn(ctx, orgID)
}
func (m *mockRepo) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error {
	return m.deductFn(ctx, tx, orgID, userID, amount, txType, referenceID, desc)
}
func (m *mockRepo) AddCredits(ctx context.Context, orgID string, amount int, desc string) error {
	return m.addCreditsFn(ctx, orgID, amount, desc)
}
func (m *mockRepo) ListTransactions(ctx context.Context, orgID string, page, limit int) ([]domain.CreditTransaction, int, error) {
	return m.listTxFn(ctx, orgID, page, limit)
}
func (m *mockRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return m.beginTxFn(ctx)
}

func TestService_GetBalance(t *testing.T) {
	repo := &mockRepo{
		getBalanceFn: func(ctx context.Context, orgID string) (int, error) {
			return 450, nil
		},
	}
	svc := credits.NewService(repo)

	resp, err := svc.GetBalance(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Balance != 450 {
		t.Errorf("expected balance 450, got %d", resp.Balance)
	}
	if resp.OrgID != "org-1" {
		t.Errorf("expected org-1, got %s", resp.OrgID)
	}
}

func TestService_AddCredits_NegativeAmount(t *testing.T) {
	repo := &mockRepo{}
	svc := credits.NewService(repo)

	err := svc.AddCredits(context.Background(), "org-1", -10, "test")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestService_AddCredits_OK(t *testing.T) {
	called := false
	repo := &mockRepo{
		addCreditsFn: func(ctx context.Context, orgID string, amount int, desc string) error {
			called = true
			if amount != 500 {
				t.Errorf("expected 500, got %d", amount)
			}
			return nil
		},
	}
	svc := credits.NewService(repo)

	if err := svc.AddCredits(context.Background(), "org-1", 500, "compra plano"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.AddCredits not called")
	}
}

func TestService_Deduct_InsufficientCredits(t *testing.T) {
	repo := &mockRepo{
		deductFn: func(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error {
			return domain.ErrInsufficientCredits
		},
	}
	svc := credits.NewService(repo)

	err := svc.Deduct(context.Background(), nil, "org-1", "user-1", 100, domain.CreditTxSearch, nil, "busca")
	if !errors.Is(err, domain.ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got %v", err)
	}
}

func TestService_ListTransactions(t *testing.T) {
	repo := &mockRepo{
		listTxFn: func(ctx context.Context, orgID string, page, limit int) ([]domain.CreditTransaction, int, error) {
			return []domain.CreditTransaction{
				{ID: "tx-1", Amount: -100, Type: domain.CreditTxSearch, CreatedAt: "2026-01-01T00:00:00Z"},
			}, 1, nil
		},
		getBalanceFn: func(ctx context.Context, orgID string) (int, error) {
			return 400, nil
		},
	}
	svc := credits.NewService(repo)

	resp, err := svc.ListTransactions(context.Background(), "org-1", 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if resp.Balance != 400 {
		t.Errorf("expected balance 400, got %d", resp.Balance)
	}
}
