package orgadmin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

func (r *postgresRepo) ListUsers(ctx context.Context, orgID string) ([]domain.OrgUser, error) {
	rows, err := r.db.Query(ctx,
		`SELECT u.id, COALESCE(u.name, u.email), u.email, u.role, u.is_active, u.credit_limit,
		        COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - INTERVAL '30 days'),
		        COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - INTERVAL '30 days'),
		        COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0 AND ct.created_at > now() - INTERVAL '30 days'), 0),
		        MAX(s.created_at)
		 FROM tb_users u
		 LEFT JOIN tb_searches s ON s.user_id = u.id
		 LEFT JOIN tb_export_queue  e ON e.user_id = u.id
		 LEFT JOIN tb_credit_transactions ct ON ct.user_id = u.id
		 WHERE u.org_id = $1 AND u.deleted_at IS NULL
		 GROUP BY u.id
		 ORDER BY u.created_at`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
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
	return users, rows.Err()
}

func (r *postgresRepo) GetUser(ctx context.Context, userID, orgID string) (*domain.OrgUser, error) {
	var u domain.OrgUser
	var lastActive *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT u.id, COALESCE(u.name, u.email), u.email, u.role, u.is_active, u.credit_limit,
		        COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - INTERVAL '30 days'),
		        COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - INTERVAL '30 days'),
		        COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0 AND ct.created_at > now() - INTERVAL '30 days'), 0),
		        MAX(s.created_at)
		 FROM tb_users u
		 LEFT JOIN tb_searches s ON s.user_id = u.id
		 LEFT JOIN tb_export_queue  e ON e.user_id = u.id
		 LEFT JOIN tb_credit_transactions ct ON ct.user_id = u.id
		 WHERE u.id = $1 AND u.org_id = $2 AND u.deleted_at IS NULL
		 GROUP BY u.id`,
		userID, orgID,
	).Scan(
		&u.UserID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreditLimit,
		&u.SearchesThisMonth, &u.ExportsThisMonth, &u.CreditsConsumed, &lastActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if lastActive != nil {
		s := lastActive.Format(time.RFC3339)
		u.LastActiveAt = &s
	}
	return &u, nil
}

func (r *postgresRepo) PatchUser(ctx context.Context, userID, orgID string, isActive *bool, creditLimit *int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE tb_users
		 SET is_active    = COALESCE($3, is_active),
		     credit_limit = COALESCE($4, credit_limit),
		     updated_at   = now()
		 WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`,
		userID, orgID, isActive, creditLimit,
	)
	if err != nil {
		return fmt.Errorf("patch user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) SoftDeleteUser(ctx context.Context, userID, orgID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE tb_users
		 SET is_active = false, deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`,
		userID, orgID,
	)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) CreateInvitation(ctx context.Context, orgID, email, role, token, invitedBy string) (*domain.Invitation, error) {
	var inv domain.Invitation
	var expiresAt, createdAt time.Time
	err := r.db.QueryRow(ctx,
		`INSERT INTO tb_invitations (org_id, email, role, token, invited_by)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, org_id, email, role, token, invited_by::text, expires_at, created_at`,
		orgID, email, role, token, invitedBy,
	).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy, &expiresAt, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}
	inv.ExpiresAt = expiresAt.Format(time.RFC3339)
	inv.CreatedAt = createdAt.Format(time.RFC3339)
	return &inv, nil
}

func (r *postgresRepo) ListInvitations(ctx context.Context, orgID string) ([]domain.Invitation, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, email, role, invited_by::text,
		        accepted_at, expires_at, created_at
		 FROM tb_invitations
		 WHERE org_id = $1
		 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	invs := []domain.Invitation{}
	for rows.Next() {
		var inv domain.Invitation
		var acceptedAt *time.Time
		var expiresAt, createdAt time.Time
		if err := rows.Scan(
			&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.InvitedBy,
			&acceptedAt, &expiresAt, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		if acceptedAt != nil {
			s := acceptedAt.Format(time.RFC3339)
			inv.AcceptedAt = &s
		}
		inv.ExpiresAt = expiresAt.Format(time.RFC3339)
		inv.CreatedAt = createdAt.Format(time.RFC3339)
		invs = append(invs, inv)
	}
	return invs, rows.Err()
}

func (r *postgresRepo) DeleteInvitation(ctx context.Context, invitationID, orgID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM tb_invitations
		 WHERE id = $1 AND org_id = $2 AND accepted_at IS NULL`,
		invitationID, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) GetUserHistory(ctx context.Context, userID, orgID string, days, page, limit int) (*UserHistory, error) {
	u, err := r.GetUser(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	interval := fmt.Sprintf("%d days", days)
	var stats UserHistStats
	if err := r.db.QueryRow(ctx,
		`SELECT
		   COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > now() - $3::interval),
		   COUNT(DISTINCT e.id) FILTER (WHERE e.created_at > now() - $3::interval),
		   COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0 AND ct.created_at > now() - $3::interval), 0)
		 FROM tb_users u
		 LEFT JOIN tb_searches s ON s.user_id = u.id
		 LEFT JOIN tb_export_queue e ON e.user_id = u.id
		 LEFT JOIN tb_credit_transactions ct ON ct.user_id = u.id
		 WHERE u.id = $1 AND u.org_id = $2`,
		userID, orgID, interval,
	).Scan(&stats.Searches, &stats.Exports, &stats.CreditsConsumed); err != nil {
		return nil, fmt.Errorf("user history stats: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT s.id,
		        COALESCE(s.query_text, array_to_string(ARRAY(SELECT c FROM jsonb_array_elements_text(s.filters->'cnaes') c), ', ')),
		        COALESCE(s.result_count, 0),
		        COALESCE(ABS(ct.amount), 0),
		        s.created_at
		 FROM tb_searches s
		 LEFT JOIN tb_credit_transactions ct ON ct.reference_id = s.id AND ct.amount < 0
		 WHERE s.user_id = $1 AND s.org_id = $2
		   AND s.created_at > now() - $3::interval
		 ORDER BY s.created_at DESC
		 LIMIT $4 OFFSET $5`,
		userID, orgID, interval, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("user history searches: %w", err)
	}
	defer rows.Close()

	searches := []SearchSummary{}
	for rows.Next() {
		var ss SearchSummary
		var createdAt time.Time
		if err := rows.Scan(&ss.SearchID, &ss.Query, &ss.ResultsCount, &ss.CreditsUsed, &createdAt); err != nil {
			return nil, fmt.Errorf("scan search: %w", err)
		}
		ss.CreatedAt = createdAt.Format(time.RFC3339)
		searches = append(searches, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &UserHistory{User: *u, Stats: stats, Searches: searches, Page: page, Total: stats.Searches}, nil
}

func (r *postgresRepo) GetOrgCosts(ctx context.Context, orgID string, days int) (*domain.OrgCosts, error) {
	interval := fmt.Sprintf("%d days", days)
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(ABS(amount)), 0)
		 FROM tb_credit_transactions
		 WHERE org_id = $1 AND amount < 0
		   AND created_at > now() - $2::interval`,
		orgID, interval,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("org costs total: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT u.id, COALESCE(u.name, u.email),
		        COUNT(DISTINCT s.id),
		        COUNT(DISTINCT e.id),
		        COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0), 0)
		 FROM tb_users u
		 LEFT JOIN tb_searches s ON s.user_id = u.id
		   AND s.created_at > now() - $2::interval
		 LEFT JOIN tb_export_queue e ON e.user_id = u.id
		   AND e.created_at > now() - $2::interval
		 LEFT JOIN tb_credit_transactions ct ON ct.user_id = u.id
		   AND ct.created_at > now() - $2::interval
		 WHERE u.org_id = $1 AND u.deleted_at IS NULL
		 GROUP BY u.id
		 ORDER BY COALESCE(SUM(ABS(ct.amount)) FILTER (WHERE ct.amount < 0), 0) DESC`,
		orgID, interval,
	)
	if err != nil {
		return nil, fmt.Errorf("org costs by seller: %w", err)
	}
	defer rows.Close()

	sellers := []domain.SellerCost{}
	for rows.Next() {
		var sc domain.SellerCost
		if err := rows.Scan(&sc.UserID, &sc.Name, &sc.Searches, &sc.Exports, &sc.CreditsConsumed); err != nil {
			return nil, fmt.Errorf("scan seller cost: %w", err)
		}
		sellers = append(sellers, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	period := fmt.Sprintf("%dd", days)
	return &domain.OrgCosts{Period: period, TotalCreditsConsumed: total, BySeller: sellers}, nil
}

func (r *postgresRepo) GetSellerProfile(ctx context.Context, userID string) (*domain.SellerProfile, error) {
	var p domain.SellerProfile
	err := r.db.QueryRow(ctx,
		`SELECT u.id, COALESCE(u.name, u.email), u.email, o.name, u.credit_limit,
		        COALESCE((SELECT SUM(ABS(amount)) FROM tb_credit_transactions
		                  WHERE user_id = u.id AND amount < 0
		                    AND created_at > now() - INTERVAL '30 days'), 0)
		 FROM tb_users u
		 JOIN tb_organizations o ON o.id = u.org_id
		 WHERE u.id = $1`,
		userID,
	).Scan(&p.UserID, &p.Name, &p.Email, &p.OrgName, &p.CreditLimit, &p.CreditsConsumedMonth)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seller profile: %w", err)
	}
	return &p, nil
}

func (r *postgresRepo) UpdateProfile(ctx context.Context, userID, name string, passwordHash *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tb_users
		 SET name          = CASE WHEN $2 != '' THEN $2 ELSE name END,
		     password_hash = COALESCE($3, password_hash),
		     updated_at    = now()
		 WHERE id = $1`,
		userID, name, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *postgresRepo) ListSellerSearches(ctx context.Context, userID, orgID string, days, page, limit int) ([]SearchSummary, int, error) {
	interval := fmt.Sprintf("%d days", days)
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tb_searches
		 WHERE user_id = $1 AND org_id = $2
		   AND created_at > now() - $3::interval`,
		userID, orgID, interval,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count searches: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT s.id,
		        COALESCE(s.query_text, ''),
		        COALESCE(s.result_count, 0),
		        COALESCE(ABS(ct.amount), 0),
		        s.created_at
		 FROM tb_searches s
		 LEFT JOIN tb_credit_transactions ct ON ct.reference_id = s.id AND ct.amount < 0
		 WHERE s.user_id = $1 AND s.org_id = $2
		   AND s.created_at > now() - $3::interval
		 ORDER BY s.created_at DESC
		 LIMIT $4 OFFSET $5`,
		userID, orgID, interval, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list seller searches: %w", err)
	}
	defer rows.Close()

	searches := []SearchSummary{}
	for rows.Next() {
		var ss SearchSummary
		var createdAt time.Time
		if err := rows.Scan(&ss.SearchID, &ss.Query, &ss.ResultsCount, &ss.CreditsUsed, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan search: %w", err)
		}
		ss.CreatedAt = createdAt.Format(time.RFC3339)
		searches = append(searches, ss)
	}
	return searches, total, rows.Err()
}

func (r *postgresRepo) WriteAuditLog(ctx context.Context, orgID *string, actorID, action string, targetID *string, metadata map[string]any) error {
	meta, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	if _, err := r.db.Exec(ctx,
		`INSERT INTO tb_audit_logs (org_id, actor_id, action, target_id, metadata)
		 VALUES ($1,$2,$3,$4,$5)`,
		orgID, actorID, action, targetID, meta,
	); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
