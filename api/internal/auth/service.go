package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrTokenInvalid       = errors.New("token inválido ou expirado")
)

type Claims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type ServiceInterface interface {
	Login(ctx context.Context, email, password string) (*TokenPair, error)
	Refresh(ctx context.Context, rawToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	u, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.generateTokenPair(ctx, u.ID, u.OrgID, u.Role)
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	h := sha256hex(rawToken)
	rec, err := s.repo.FindUserByRefreshToken(ctx, h)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, ErrTokenInvalid
	}
	// rotate: delete old token before issuing new one; log but don't fail on cleanup error
	if err := s.repo.DeleteRefreshToken(ctx, h); err != nil {
		slog.WarnContext(ctx, "delete old refresh token during rotation", "error", err)
	}
	return s.generateTokenPair(ctx, rec.UserID, rec.OrgID, rec.Role)
}

func (s *Service) Logout(ctx context.Context, userID string) error {
	if err := s.repo.DeleteUserRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

func (s *Service) generateTokenPair(ctx context.Context, userID, orgID, role string) (*TokenPair, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	now := time.Now()
	const expiresIn = 24 * 60 * 60

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	accessStr, err := accessToken.SignedString(secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshHash := sha256hex(rawRefresh)
	expiresAt := now.Add(30 * 24 * time.Hour)

	if err := s.repo.StoreRefreshToken(ctx, userID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: rawRefresh,
		ExpiresIn:    expiresIn,
	}, nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
