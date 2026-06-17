package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// mockAuthRepo is a controllable stub for the Repository interface.
type mockAuthRepo struct {
	user                *UserRecord
	userErr             error
	refreshRec          *RefreshTokenRecord
	refreshErr          error
	deleteRefreshErr    error
	storeRefreshErr     error
	deleteUserTokensErr error
}

func (m *mockAuthRepo) FindUserByEmail(_ context.Context, _ string) (*UserRecord, error) {
	return m.user, m.userErr
}
func (m *mockAuthRepo) FindUserByRefreshToken(_ context.Context, _ string) (*RefreshTokenRecord, error) {
	return m.refreshRec, m.refreshErr
}
func (m *mockAuthRepo) DeleteRefreshToken(_ context.Context, _ string) error {
	return m.deleteRefreshErr
}
func (m *mockAuthRepo) StoreRefreshToken(_ context.Context, _, _ string, _ time.Time) error {
	return m.storeRefreshErr
}
func (m *mockAuthRepo) DeleteUserRefreshTokens(_ context.Context, _ string) error {
	return m.deleteUserTokensErr
}

func activeUser(hash string) *UserRecord {
	return &UserRecord{ID: "user-1", OrgID: "org-1", Role: "operator", Hash: hash, IsActive: true}
}

func quickHash(t *testing.T, password string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	hash := quickHash(t, "correct-pass")
	repo := &mockAuthRepo{user: activeUser(hash)}
	svc := NewService(repo)

	pair, err := svc.Login(context.Background(), "user@example.com", "correct-pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh_token is empty")
	}
	if pair.ExpiresIn != 86400 {
		t.Errorf("expires_in = %d, want 86400", pair.ExpiresIn)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	repo := &mockAuthRepo{userErr: errors.New("not found")}
	svc := NewService(repo)

	_, err := svc.Login(context.Background(), "unknown@example.com", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	hash := quickHash(t, "pass")
	repo := &mockAuthRepo{
		user: &UserRecord{ID: "u2", Hash: hash, IsActive: false},
	}
	svc := NewService(repo)

	_, err := svc.Login(context.Background(), "inactive@example.com", "pass")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	hash := quickHash(t, "correct-pass")
	repo := &mockAuthRepo{user: activeUser(hash)}
	svc := NewService(repo)

	_, err := svc.Login(context.Background(), "user@example.com", "wrong-pass")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_StoreTokenError(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	hash := quickHash(t, "pass")
	repo := &mockAuthRepo{user: activeUser(hash), storeRefreshErr: errors.New("db error")}
	svc := NewService(repo)

	_, err := svc.Login(context.Background(), "user@example.com", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	repo := &mockAuthRepo{
		refreshRec: &RefreshTokenRecord{
			UserID:    "user-1",
			OrgID:     "org-1",
			Role:      "operator",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}
	svc := NewService(repo)

	pair, err := svc.Refresh(context.Background(), "some-raw-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("access_token is empty after refresh")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh_token is empty after refresh")
	}
}

func TestRefresh_TokenNotFound(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	repo := &mockAuthRepo{refreshErr: errors.New("not found")}
	svc := NewService(repo)

	_, err := svc.Refresh(context.Background(), "unknown-token")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v, want ErrTokenInvalid", err)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	repo := &mockAuthRepo{
		refreshRec: &RefreshTokenRecord{
			UserID:    "user-1",
			OrgID:     "org-1",
			Role:      "operator",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		},
	}
	svc := NewService(repo)

	_, err := svc.Refresh(context.Background(), "expired-token")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("got %v, want ErrTokenInvalid", err)
	}
}

func TestRefresh_DeleteOldTokenError_StillSucceeds(t *testing.T) {
	// Rotation failure on the old token should log a warning but not abort the refresh.
	t.Setenv("JWT_SECRET", "test-secret-key-long-enough")

	repo := &mockAuthRepo{
		refreshRec: &RefreshTokenRecord{
			UserID:    "user-1",
			OrgID:     "org-1",
			Role:      "operator",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		deleteRefreshErr: errors.New("transient db error"),
	}
	svc := NewService(repo)

	pair, err := svc.Refresh(context.Background(), "some-token")
	if err != nil {
		t.Fatalf("expected success despite delete error, got: %v", err)
	}
	if pair == nil || pair.AccessToken == "" {
		t.Error("expected valid token pair")
	}
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	repo := &mockAuthRepo{}
	svc := NewService(repo)

	if err := svc.Logout(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogout_RepoError(t *testing.T) {
	repo := &mockAuthRepo{deleteUserTokensErr: errors.New("db error")}
	svc := NewService(repo)

	err := svc.Logout(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// HashPassword helper
// ---------------------------------------------------------------------------

func TestHashPassword_IsVerifiable(t *testing.T) {
	hash, err := HashPassword("my-secret")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("my-secret")); err != nil {
		t.Errorf("hash does not match password: %v", err)
	}
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Error("expected different hashes due to salt, got identical")
	}
}
