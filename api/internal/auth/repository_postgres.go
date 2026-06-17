package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) FindUserByEmail(ctx context.Context, email string) (*UserRecord, error) {
	var u UserRecord
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, role, password_hash, is_active
		 FROM tb_users
		 WHERE email=$1 AND deleted_at IS NULL`,
		email,
	).Scan(&u.ID, &u.OrgID, &u.Role, &u.Hash, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}

func (r *postgresRepo) FindUserByRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error) {
	var rec RefreshTokenRecord
	err := r.db.QueryRow(ctx,
		`SELECT u.id, u.org_id, u.role, rt.expires_at
		 FROM tb_refresh_tokens rt
		 JOIN tb_users u ON u.id = rt.user_id
		 WHERE rt.token_hash=$1 AND u.is_active=true AND u.deleted_at IS NULL`,
		tokenHash,
	).Scan(&rec.UserID, &rec.OrgID, &rec.Role, &rec.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	return &rec, nil
}

func (r *postgresRepo) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM tb_refresh_tokens WHERE token_hash=$1`, tokenHash); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

func (r *postgresRepo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO tb_refresh_tokens (user_id, token_hash, expires_at) VALUES ($1,$2,$3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *postgresRepo) DeleteUserRefreshTokens(ctx context.Context, userID string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM tb_refresh_tokens WHERE user_id=$1`, userID); err != nil {
		return fmt.Errorf("delete user refresh tokens: %w", err)
	}
	return nil
}
