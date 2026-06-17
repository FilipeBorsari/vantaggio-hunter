package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, credits, price_cents FROM tb_plans WHERE active=true ORDER BY price_cents`)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()
	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.Credits, &p.PriceCents); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return plans, nil
}

func (r *postgresRepo) CreateOrg(ctx context.Context, name string, planID *string) (*domain.Org, error) {
	var o domain.Org
	var createdAt time.Time
	err := r.db.QueryRow(ctx,
		`INSERT INTO tb_organizations (name, plan_id)
		 VALUES ($1,$2)
		 RETURNING id, name, plan_id, is_active, created_at`,
		name, planID,
	).Scan(&o.ID, &o.Name, &o.PlanID, &o.IsActive, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}
	o.CreatedAt = createdAt.Format(time.RFC3339)
	return &o, nil
}

func (r *postgresRepo) CreateUser(ctx context.Context, orgID, email, passwordHash, role string) (*domain.User, error) {
	var id string
	err := r.db.QueryRow(ctx,
		`INSERT INTO tb_users (org_id, email, password_hash, role)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id`,
		orgID, email, passwordHash, role,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &domain.User{ID: id, Email: email, Role: role}, nil
}

func (r *postgresRepo) ListOrgs(ctx context.Context, page, limit int) ([]domain.Org, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.name, o.plan_id, p.name, o.is_active, o.created_at,
		        COUNT(u.id) AS user_count
		 FROM tb_organizations o
		 LEFT JOIN tb_plans p ON p.id = o.plan_id
		 LEFT JOIN tb_users u ON u.org_id = o.id AND u.deleted_at IS NULL
		 GROUP BY o.id, p.name
		 ORDER BY o.created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list orgs: %w", err)
	}
	defer rows.Close()

	orgs := []domain.Org{}
	for rows.Next() {
		var o domain.Org
		var createdAt time.Time
		if err := rows.Scan(&o.ID, &o.Name, &o.PlanID, &o.PlanName, &o.IsActive, &createdAt, &o.UserCount); err != nil {
			return nil, fmt.Errorf("scan org: %w", err)
		}
		o.CreatedAt = createdAt.Format(time.RFC3339)
		orgs = append(orgs, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return orgs, nil
}

func (r *postgresRepo) CountOrgs(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tb_organizations`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count orgs: %w", err)
	}
	return total, nil
}

func (r *postgresRepo) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	if _, err := r.db.Exec(ctx,
		`UPDATE tb_users SET is_active=$1, updated_at=now() WHERE id=$2`,
		isActive, userID,
	); err != nil {
		return fmt.Errorf("set user active: %w", err)
	}
	return nil
}
