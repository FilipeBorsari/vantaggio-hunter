package credits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return tx, nil
}

func (r *postgresRepo) GetBalance(ctx context.Context, orgID string) (int, error) {
	var balance int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(balance, 0) FROM tb_credit_balances WHERE org_id = $1`,
		orgID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (r *postgresRepo) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, referenceID *string, desc string) error {
	var balance int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(balance, 0) FROM tb_credit_balances WHERE org_id = $1 FOR UPDATE`,
		orgID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInsufficientCredits
		}
		return fmt.Errorf("lock balance: %w", err)
	}
	if balance < amount {
		return domain.ErrInsufficientCredits
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO tb_credit_transactions (org_id, user_id, type, amount, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6::uuid)`,
		orgID, userID, txType, -amount, desc, referenceID,
	)
	if err != nil {
		return fmt.Errorf("insert debit transaction: %w", err)
	}
	return nil
}

func (r *postgresRepo) AddCredits(ctx context.Context, orgID string, amount int, desc string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO tb_credit_transactions (org_id, type, amount, description)
		 VALUES ($1, $2, $3, $4)`,
		orgID, domain.CreditTxPurchase, amount, desc,
	)
	if err != nil {
		return fmt.Errorf("add credits: %w", err)
	}
	return nil
}

func (r *postgresRepo) ListTransactions(ctx context.Context, orgID string, page, limit int) ([]domain.CreditTransaction, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tb_credit_transactions WHERE org_id = $1`, orgID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, user_id, type, amount, description, reference_id, created_at
		 FROM tb_credit_transactions
		 WHERE org_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var result []domain.CreditTransaction
	for rows.Next() {
		var t domain.CreditTransaction
		var createdAt time.Time
		var refID *string
		if err := rows.Scan(
			&t.ID, &t.OrgID, &t.UserID, &t.Type, &t.Amount,
			&t.Description, &refID, &createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		if refID != nil {
			t.ReferenceID = refID
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	if result == nil {
		result = []domain.CreditTransaction{}
	}
	return result, total, nil
}
