package auth

import (
	"context"
	"time"
)

type UserRecord struct {
	ID       string
	OrgID    string
	Role     string
	Hash     string
	IsActive bool
}

type RefreshTokenRecord struct {
	UserID    string
	OrgID     string
	Role      string
	ExpiresAt time.Time
}

type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*UserRecord, error)
	FindUserByRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	DeleteUserRefreshTokens(ctx context.Context, userID string) error
}
