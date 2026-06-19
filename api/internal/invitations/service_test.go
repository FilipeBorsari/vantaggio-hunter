package invitations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// ---------------------------------------------------------------------------
// mockInvRepo: stub para Repository
// ---------------------------------------------------------------------------

type mockInvRepo struct {
	record         *InvitationRecord
	findErr        error
	acceptErr      error
	createUserID   string
	createUserErr  error
}

func (m *mockInvRepo) FindByToken(_ context.Context, _ string) (*InvitationRecord, error) {
	return m.record, m.findErr
}
func (m *mockInvRepo) AcceptInvitation(_ context.Context, _, _ string) error {
	return m.acceptErr
}
func (m *mockInvRepo) CreateUserFromInvite(_ context.Context, _, _, _, _, _ string) (string, error) {
	return m.createUserID, m.createUserErr
}

// ---------------------------------------------------------------------------
// mockAuthSvc: stub para auth.ServiceInterface
// ---------------------------------------------------------------------------

type mockAuthSvc struct {
	pair *auth.TokenPair
	err  error
}

func (m *mockAuthSvc) Login(_ context.Context, _, _ string) (*auth.TokenPair, error) {
	return m.pair, m.err
}
func (m *mockAuthSvc) Refresh(_ context.Context, _ string) (*auth.TokenPair, error) {
	return m.pair, m.err
}
func (m *mockAuthSvc) Logout(_ context.Context, _ string) error { return m.err }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func futureExpiry() string {
	return time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
}

func pastExpiry() string {
	return time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// ValidateToken
// ---------------------------------------------------------------------------

func TestValidateToken_Success(t *testing.T) {
	rec := &InvitationRecord{
		ID:      "inv-1",
		OrgID:   "org-1",
		OrgName: "Corp",
		Email:   "v@co.com",
		Role:    "seller",
		Token:   "tok123",
		ExpiresAt: futureExpiry(),
	}
	svc := NewService(&mockInvRepo{record: rec}, nil)

	got, err := svc.ValidateToken(context.Background(), "tok123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "v@co.com" {
		t.Errorf("email = %q, want v@co.com", got.Email)
	}
	if got.OrgName != "Corp" {
		t.Errorf("org_name = %q, want Corp", got.OrgName)
	}
}

func TestValidateToken_NotFound(t *testing.T) {
	svc := NewService(&mockInvRepo{findErr: domain.ErrNotFound}, nil)
	_, err := svc.ValidateToken(context.Background(), "bad-token")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	rec := &InvitationRecord{
		ID:        "inv-expired",
		Email:     "v@co.com",
		ExpiresAt: pastExpiry(),
	}
	svc := NewService(&mockInvRepo{record: rec}, nil)
	_, err := svc.ValidateToken(context.Background(), "exp-token")
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_RepoError(t *testing.T) {
	svc := NewService(&mockInvRepo{findErr: errors.New("db error")}, nil)
	_, err := svc.ValidateToken(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Accept
// ---------------------------------------------------------------------------

func TestAccept_Success(t *testing.T) {
	rec := &InvitationRecord{
		ID: "inv-1", OrgID: "org-1", Email: "v@co.com", Role: "seller",
		ExpiresAt: futureExpiry(),
	}
	pair := &auth.TokenPair{AccessToken: "jwt-access", RefreshToken: "ref", ExpiresIn: 86400}
	svc := NewService(
		&mockInvRepo{record: rec, createUserID: "user-new"},
		&mockAuthSvc{pair: pair},
	)

	token, err := svc.Accept(context.Background(), "tok123", "Vendedor", "senhaForte!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "jwt-access" {
		t.Errorf("token = %q, want jwt-access", token)
	}
}

func TestAccept_ExpiredToken(t *testing.T) {
	rec := &InvitationRecord{ExpiresAt: pastExpiry(), Email: "v@co.com"}
	svc := NewService(&mockInvRepo{record: rec}, nil)

	_, err := svc.Accept(context.Background(), "tok", "Nome", "senha")
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestAccept_NotFound(t *testing.T) {
	svc := NewService(&mockInvRepo{findErr: domain.ErrNotFound}, nil)

	_, err := svc.Accept(context.Background(), "bad-tok", "Nome", "senha")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAccept_EmailConflict(t *testing.T) {
	// E-mail já registrado — CreateUserFromInvite retorna ErrConflict.
	rec := &InvitationRecord{
		OrgID: "org-1", Email: "dup@co.com", Role: "seller",
		ExpiresAt: futureExpiry(),
	}
	svc := NewService(
		&mockInvRepo{record: rec, createUserErr: domain.ErrConflict},
		nil,
	)

	_, err := svc.Accept(context.Background(), "tok", "Nome", "senha")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestAccept_LoginFails(t *testing.T) {
	// Usuário criado com sucesso, mas login falha.
	rec := &InvitationRecord{
		OrgID: "org-1", Email: "v@co.com", Role: "seller",
		ExpiresAt: futureExpiry(),
	}
	svc := NewService(
		&mockInvRepo{record: rec, createUserID: "u-new"},
		&mockAuthSvc{err: auth.ErrInvalidCredentials},
	)

	_, err := svc.Accept(context.Background(), "tok", "Nome", "senha")
	if err == nil {
		t.Fatal("expected error when login fails, got nil")
	}
}

func TestAccept_AcceptMarkFails(t *testing.T) {
	// Usuário criado, mas marcar convite como aceito falha.
	rec := &InvitationRecord{
		OrgID: "org-1", Email: "v@co.com", Role: "seller",
		ExpiresAt: futureExpiry(),
	}
	svc := NewService(
		&mockInvRepo{record: rec, createUserID: "u-new", acceptErr: errors.New("db error")},
		nil,
	)

	_, err := svc.Accept(context.Background(), "tok", "Nome", "senha")
	if err == nil {
		t.Fatal("expected error when accept mark fails, got nil")
	}
}
