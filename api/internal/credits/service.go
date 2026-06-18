package credits

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type ServiceInterface interface {
	GetBalance(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error)
	// Deduct debits amount credits atomically within the given transaction.
	// Callers must commit or rollback the transaction.
	Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error
	BeginTx(ctx context.Context) (pgx.Tx, error)
	AddCredits(ctx context.Context, orgID string, amount int, desc string) error
	ListTransactions(ctx context.Context, orgID string, page, limit int) (*domain.CreditTransactionsResponse, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetBalance(ctx context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	balance, err := s.repo.GetBalance(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return &domain.CreditBalanceResponse{Balance: balance, OrgID: orgID}, nil
}

func (s *Service) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.repo.BeginTx(ctx)
}

func (s *Service) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error {
	return s.repo.Deduct(ctx, tx, orgID, userID, amount, txType, referenceID, desc)
}

func (s *Service) AddCredits(ctx context.Context, orgID string, amount int, desc string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if err := s.repo.AddCredits(ctx, orgID, amount, desc); err != nil {
		return fmt.Errorf("add credits: %w", err)
	}
	return nil
}

func (s *Service) ListTransactions(ctx context.Context, orgID string, page, limit int) (*domain.CreditTransactionsResponse, error) {
	txs, total, err := s.repo.ListTransactions(ctx, orgID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	balance, err := s.repo.GetBalance(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return &domain.CreditTransactionsResponse{
		Data:    txs,
		Total:   total,
		Balance: balance,
	}, nil
}
