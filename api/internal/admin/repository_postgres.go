package admin

import (
	"context"
	"encoding/json"
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

func (r *postgresRepo) CreateUser(ctx context.Context, orgID, name, email, passwordHash, role string) (*domain.User, error) {
	var id string
	err := r.db.QueryRow(ctx,
		`INSERT INTO tb_users (org_id, name, email, password_hash, role)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id`,
		orgID, name, email, passwordHash, role,
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

func (r *postgresRepo) ListOrgs(ctx context.Context, page, limit int, q string) ([]domain.Org, error) {
	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.name, o.plan_id, p.name, o.is_active, o.created_at,
		        COUNT(u.id) AS user_count
		 FROM tb_organizations o
		 LEFT JOIN tb_plans p ON p.id = o.plan_id
		 LEFT JOIN tb_users u ON u.org_id = o.id AND u.deleted_at IS NULL
		 WHERE ($3 = '' OR o.name ILIKE '%' || $3 || '%')
		 GROUP BY o.id, p.name
		 ORDER BY o.created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset, q,
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

func (r *postgresRepo) GetOrgDetail(ctx context.Context, orgID string) (*domain.OrgDetail, error) {
	var o domain.Org
	var createdAt time.Time
	err := r.db.QueryRow(ctx,
		`SELECT o.id, o.name, o.plan_id, p.name, o.is_active, o.created_at,
		        COUNT(u.id) AS user_count
		 FROM tb_organizations o
		 LEFT JOIN tb_plans p ON p.id = o.plan_id
		 LEFT JOIN tb_users u ON u.org_id = o.id AND u.deleted_at IS NULL
		 WHERE o.id = $1
		 GROUP BY o.id, p.name`,
		orgID,
	).Scan(&o.ID, &o.Name, &o.PlanID, &o.PlanName, &o.IsActive, &createdAt, &o.UserCount)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	o.CreatedAt = createdAt.Format(time.RFC3339)

	// Stats
	var stats domain.OrgStats
	if err := r.db.QueryRow(ctx,
		`SELECT COALESCE(b.balance, 0),
		        COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - INTERVAL '30 days'),
		        COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - INTERVAL '30 days')
		 FROM tb_organizations org
		 LEFT JOIN tb_credit_balances b ON b.org_id = org.id
		 LEFT JOIN tb_searches s ON s.org_id = org.id
		 LEFT JOIN tb_export_queue e ON e.org_id = org.id
		 WHERE org.id = $1
		 GROUP BY b.balance`,
		orgID,
	).Scan(&stats.Balance, &stats.TotalSearches, &stats.Exports); err != nil {
		return nil, fmt.Errorf("get org stats: %w", err)
	}

	// Users
	rows, err := r.db.Query(ctx,
		`SELECT u.id, COALESCE(u.name, u.email), u.email, u.role, u.is_active, u.credit_limit,
		        COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - INTERVAL '30 days'),
		        COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - INTERVAL '30 days'),
		        COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0 AND ct.created_at > now() - INTERVAL '30 days'), 0),
		        MAX(s.created_at)
		 FROM tb_users u
		 LEFT JOIN tb_searches s ON s.user_id = u.id
		 LEFT JOIN tb_export_queue e ON e.user_id = u.id
		 LEFT JOIN tb_credit_transactions ct ON ct.user_id = u.id
		 WHERE u.org_id = $1 AND u.deleted_at IS NULL
		 GROUP BY u.id
		 ORDER BY u.created_at`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("get org users: %w", err)
	}
	defer rows.Close()

	users := []domain.OrgUser{}
	for rows.Next() {
		var u domain.OrgUser
		var lastActive *time.Time
		if err := rows.Scan(
			&u.UserID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreditLimit,
			&u.SearchesThisMonth, &u.ExportsThisMonth, &u.CreditsConsumed, &lastActive,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if lastActive != nil {
			s := lastActive.Format(time.RFC3339)
			u.LastActiveAt = &s
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &domain.OrgDetail{Org: o, Stats: stats, Users: users}, nil
}

func (r *postgresRepo) PatchOrg(ctx context.Context, orgID string, isActive *bool, planID *string) error {
	if isActive == nil && planID == nil {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE tb_organizations
		 SET is_active  = COALESCE($2, is_active),
		     plan_id    = COALESCE($3::uuid, plan_id),
		     updated_at = now()
		 WHERE id = $1`,
		orgID, isActive, planID,
	)
	if err != nil {
		return fmt.Errorf("patch org: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetAdminDashboard(ctx context.Context, days int) (*domain.AdminDashboard, error) {
	interval := fmt.Sprintf("%d days", days)
	var d domain.AdminDashboard
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE is_active),
		        (SELECT COALESCE(COUNT(*),0) FROM tb_searches WHERE created_at > now() - $1::interval),
		        (SELECT COALESCE(COUNT(*),0) FROM tb_export_queue WHERE created_at > now() - $1::interval),
		        (SELECT COALESCE(SUM(ABS(amount)),0) FROM tb_credit_transactions WHERE amount < 0 AND created_at > now() - $1::interval)
		 FROM tb_organizations`,
		interval,
	).Scan(&d.TotalOrgs, &d.ActiveOrgs, &d.TotalSearches, &d.TotalExports, &d.TotalCreditsConsumed); err != nil {
		return nil, fmt.Errorf("admin dashboard: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.name, o.is_active,
		        COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - $1::interval),
		        COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - $1::interval),
		        COALESCE(b.balance, 0)
		 FROM tb_organizations o
		 LEFT JOIN tb_searches s ON s.org_id = o.id
		 LEFT JOIN tb_export_queue e ON e.org_id = o.id
		 LEFT JOIN tb_credit_balances b ON b.org_id = o.id
		 GROUP BY o.id, b.balance
		 ORDER BY o.created_at DESC`,
		interval,
	)
	if err != nil {
		return nil, fmt.Errorf("admin dashboard orgs: %w", err)
	}
	defer rows.Close()

	d.Orgs = []domain.OrgSummary{}
	for rows.Next() {
		var s domain.OrgSummary
		if err := rows.Scan(&s.OrgID, &s.Name, &s.IsActive, &s.Searches, &s.Exports, &s.Balance); err != nil {
			return nil, fmt.Errorf("scan org summary: %w", err)
		}
		d.Orgs = append(d.Orgs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return &d, nil
}

func (r *postgresRepo) WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error {
	meta, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	if _, err := r.db.Exec(ctx,
		`INSERT INTO tb_audit_logs (org_id, actor_id, action, target_id, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		orgID, actorID, action, targetID, meta,
	); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
