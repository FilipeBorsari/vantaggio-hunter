package credits

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	GetBalance(ctx context.Context, orgID string) (int, error)
	// Deduct debits amount credits atomically within the provided transaction.
	// Returns ErrInsufficientCredits if balance < amount.
	Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error
	AddCredits(ctx context.Context, orgID string, amount int, desc string) error
	ListTransactions(ctx context.Context, orgID string, page, limit int) ([]domain.CreditTransaction, int, error)
	BeginTx(ctx context.Context) (pgx.Tx, error)
}
