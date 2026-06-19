package invitations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

func (r *postgresRepo) FindByToken(ctx context.Context, token string) (*InvitationRecord, error) {
	var rec InvitationRecord
	var expiresAt time.Time
	err := r.db.QueryRow(ctx,
		`SELECT i.id, i.org_id, o.name, i.email, i.role, i.token, i.expires_at
		 FROM tb_invitations i
		 JOIN tb_organizations o ON o.id = i.org_id
		 WHERE i.token = $1 AND i.accepted_at IS NULL`,
		token,
	).Scan(&rec.ID, &rec.OrgID, &rec.OrgName, &rec.Email, &rec.Role, &rec.Token, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invitation: %w", err)
	}
	rec.ExpiresAt = expiresAt.Format(time.RFC3339)
	return &rec, nil
}

func (r *postgresRepo) AcceptInvitation(ctx context.Context, token, userID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE tb_invitations
		 SET accepted_at = now()
		 WHERE token = $1 AND accepted_at IS NULL`,
		token,
	)
	if err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) CreateUserFromInvite(ctx context.Context, orgID, name, email, passwordHash, role string) (string, error) {
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
			return "", domain.ErrConflict
		}
		return "", fmt.Errorf("create user from invite: %w", err)
	}
	return id, nil
}
