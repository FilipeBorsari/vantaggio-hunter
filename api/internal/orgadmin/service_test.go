package orgadmin

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
)


// ---------------------------------------------------------------------------
// mockRepo: stub for Repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	listUsersErr      error
	users             []domain.OrgUser
	getUserUser       *domain.OrgUser
	getUserErr        error
	patchUserErr      error
	softDeleteErr     error
	invitation        *domain.Invitation
	createInviteErr   error
	invitations       []domain.Invitation
	listInviteErr     error
	deleteInviteErr   error
	history           *UserHistory
	historyErr        error
	costs             *domain.OrgCosts
	costsErr          error
	profile           *domain.SellerProfile
	profileErr        error
	updateProfileErr  error
	searches          []SearchSummary
	searchCount       int
	searchesErr       error
	capturedIsActive  *bool
	capturedLimit     *int
	capturedPwdHash   *string
}

func (m *mockRepo) ListUsers(_ context.Context, _ string) ([]domain.OrgUser, error) {
	return m.users, m.listUsersErr
}
func (m *mockRepo) GetUser(_ context.Context, _, _ string) (*domain.OrgUser, error) {
	return m.getUserUser, m.getUserErr
}
func (m *mockRepo) PatchUser(_ context.Context, _, _ string, isActive *bool, creditLimit *int) error {
	m.capturedIsActive = isActive
	m.capturedLimit = creditLimit
	return m.patchUserErr
}
func (m *mockRepo) SoftDeleteUser(_ context.Context, _, _ string) error {
	return m.softDeleteErr
}
func (m *mockRepo) CreateInvitation(_ context.Context, _, _, _, _, _ string) (*domain.Invitation, error) {
	return m.invitation, m.createInviteErr
}
func (m *mockRepo) ListInvitations(_ context.Context, _ string) ([]domain.Invitation, error) {
	return m.invitations, m.listInviteErr
}
func (m *mockRepo) DeleteInvitation(_ context.Context, _, _ string) error {
	return m.deleteInviteErr
}
func (m *mockRepo) GetUserHistory(_ context.Context, _, _ string, _, _, _ int) (*UserHistory, error) {
	return m.history, m.historyErr
}
func (m *mockRepo) GetOrgCosts(_ context.Context, _ string, _ int) (*domain.OrgCosts, error) {
	return m.costs, m.costsErr
}
func (m *mockRepo) GetSellerProfile(_ context.Context, _ string) (*domain.SellerProfile, error) {
	return m.profile, m.profileErr
}
func (m *mockRepo) UpdateProfile(_ context.Context, _ string, _ string, pwdHash *string) error {
	m.capturedPwdHash = pwdHash
	return m.updateProfileErr
}
func (m *mockRepo) ListSellerSearches(_ context.Context, _, _ string, _, _, _ int) ([]SearchSummary, int, error) {
	return m.searches, m.searchCount, m.searchesErr
}
func (m *mockRepo) WriteAuditLog(_ context.Context, _ *string, _, _ string, _ *string, _ map[string]any) error {
	return nil
}

// mockCreditsSvc: stub for credits.ServiceInterface
type mockCreditsSvc struct {
	balance    int
	balanceErr error
}

func (m *mockCreditsSvc) GetBalance(_ context.Context, orgID string) (*domain.CreditBalanceResponse, error) {
	if m.balanceErr != nil {
		return nil, m.balanceErr
	}
	return &domain.CreditBalanceResponse{Balance: m.balance, OrgID: orgID}, nil
}
func (m *mockCreditsSvc) Deduct(_ context.Context, _ pgx.Tx, _, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error {
	return nil
}
func (m *mockCreditsSvc) BeginTx(_ context.Context) (pgx.Tx, error) { return nil, nil }
func (m *mockCreditsSvc) AddCredits(_ context.Context, _ string, _ int, _ string) error {
	return nil
}
func (m *mockCreditsSvc) ListTransactions(_ context.Context, _ string, _, _ int) (*domain.CreditTransactionsResponse, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsers_Success(t *testing.T) {
	users := []domain.OrgUser{
		{UserID: "u1", Name: "Ana", Email: "ana@co.com", Role: "seller", IsActive: true},
		{UserID: "u2", Name: "Bob", Email: "bob@co.com", Role: "seller", IsActive: false},
	}
	svc := NewService(&mockRepo{users: users}, nil)

	got, err := svc.ListUsers(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
	if got[0].Name != "Ana" {
		t.Errorf("name = %q, want Ana", got[0].Name)
	}
}

func TestListUsers_RepoError(t *testing.T) {
	svc := NewService(&mockRepo{listUsersErr: errors.New("db error")}, nil)
	_, err := svc.ListUsers(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PatchUser
// ---------------------------------------------------------------------------

func TestPatchUser_Deactivate_Success(t *testing.T) {
	active := false
	repo := &mockRepo{}
	svc := NewService(repo, nil)

	if err := svc.PatchUser(context.Background(), "u1", "org-1", "actor", &active, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedIsActive == nil || *repo.capturedIsActive != false {
		t.Error("expected is_active=false to be passed to repo")
	}
}

func TestPatchUser_SelfDeactivation_Blocked(t *testing.T) {
	active := false
	svc := NewService(&mockRepo{}, nil)

	// userID == actorID with is_active=false → must fail
	err := svc.PatchUser(context.Background(), "same", "org-1", "same", &active, nil)
	if !errors.Is(err, ErrSelfDeactivation) {
		t.Errorf("expected ErrSelfDeactivation, got %v", err)
	}
}

func TestPatchUser_Activate_AllowedOnSelf(t *testing.T) {
	// activating self is allowed
	active := true
	repo := &mockRepo{}
	svc := NewService(repo, nil)

	if err := svc.PatchUser(context.Background(), "same", "org-1", "same", &active, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchUser_SetCreditLimit(t *testing.T) {
	limit := 500
	repo := &mockRepo{}
	svc := NewService(repo, nil)

	if err := svc.PatchUser(context.Background(), "u1", "org-1", "actor", nil, &limit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedLimit == nil || *repo.capturedLimit != 500 {
		t.Error("expected credit_limit=500 to be passed to repo")
	}
}

func TestPatchUser_NotFound(t *testing.T) {
	active := false
	svc := NewService(&mockRepo{patchUserErr: domain.ErrNotFound}, nil)

	err := svc.PatchUser(context.Background(), "u9", "org-1", "actor", &active, nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteUser
// ---------------------------------------------------------------------------

func TestDeleteUser_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	if err := svc.DeleteUser(context.Background(), "u1", "org-1", "actor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUser_SelfDelete_Blocked(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	err := svc.DeleteUser(context.Background(), "same", "org-1", "same")
	if !errors.Is(err, ErrSelfDeactivation) {
		t.Errorf("expected ErrSelfDeactivation, got %v", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{softDeleteErr: domain.ErrNotFound}, nil)
	err := svc.DeleteUser(context.Background(), "u9", "org-1", "actor")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateInvitation
// ---------------------------------------------------------------------------

func TestCreateInvitation_Success(t *testing.T) {
	inv := &domain.Invitation{ID: "inv-1", Email: "v@co.com", Role: "seller", ExpiresAt: "2030-01-01T00:00:00Z"}
	svc := NewService(&mockRepo{invitation: inv}, nil)

	got, err := svc.CreateInvitation(context.Background(), "org-1", "v@co.com", "seller", "actor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "inv-1" {
		t.Errorf("id = %q, want inv-1", got.ID)
	}
}

func TestCreateInvitation_RepoError(t *testing.T) {
	svc := NewService(&mockRepo{createInviteErr: errors.New("db error")}, nil)
	_, err := svc.CreateInvitation(context.Background(), "org-1", "v@co.com", "seller", "actor")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListInvitations
// ---------------------------------------------------------------------------

func TestListInvitations_Success(t *testing.T) {
	invs := []domain.Invitation{
		{ID: "inv-1", Email: "a@co.com"},
		{ID: "inv-2", Email: "b@co.com"},
	}
	svc := NewService(&mockRepo{invitations: invs}, nil)

	got, err := svc.ListInvitations(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ---------------------------------------------------------------------------
// DeleteInvitation
// ---------------------------------------------------------------------------

func TestDeleteInvitation_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	if err := svc.DeleteInvitation(context.Background(), "inv-1", "org-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInvitation_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{deleteInviteErr: domain.ErrNotFound}, nil)
	err := svc.DeleteInvitation(context.Background(), "inv-9", "org-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetOrgCosts
// ---------------------------------------------------------------------------

func TestGetOrgCosts_Success(t *testing.T) {
	costs := &domain.OrgCosts{
		Period:               "30d",
		TotalCreditsConsumed: 1200,
		BySeller: []domain.SellerCost{
			{UserID: "u1", Name: "Ana", Searches: 20, CreditsConsumed: 800},
		},
	}
	svc := NewService(&mockRepo{costs: costs}, nil)

	got, err := svc.GetOrgCosts(context.Background(), "org-1", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCreditsConsumed != 1200 {
		t.Errorf("total = %d, want 1200", got.TotalCreditsConsumed)
	}
	if len(got.BySeller) != 1 {
		t.Errorf("by_seller len = %d, want 1", len(got.BySeller))
	}
}

func TestGetOrgCosts_RepoError(t *testing.T) {
	svc := NewService(&mockRepo{costsErr: errors.New("db error")}, nil)
	_, err := svc.GetOrgCosts(context.Background(), "org-1", 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetOrgCredits
// ---------------------------------------------------------------------------

func TestGetOrgCredits_Success(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockCreditsSvc{balance: 4200})

	got, err := svc.GetOrgCredits(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Balance != 4200 {
		t.Errorf("balance = %d, want 4200", got.Balance)
	}
}

func TestGetOrgCredits_ServiceError(t *testing.T) {
	svc := NewService(&mockRepo{}, &mockCreditsSvc{balanceErr: errors.New("db error")})
	_, err := svc.GetOrgCredits(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetSellerProfile
// ---------------------------------------------------------------------------

func TestGetSellerProfile_Success(t *testing.T) {
	p := &domain.SellerProfile{UserID: "u1", Name: "Ana", OrgName: "Corp", CreditsConsumedMonth: 42}
	svc := NewService(&mockRepo{profile: p}, nil)

	got, err := svc.GetSellerProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Ana" {
		t.Errorf("name = %q, want Ana", got.Name)
	}
}

func TestGetSellerProfile_NotFound(t *testing.T) {
	svc := NewService(&mockRepo{profileErr: domain.ErrNotFound}, nil)
	_, err := svc.GetSellerProfile(context.Background(), "u9")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateProfile
// ---------------------------------------------------------------------------

func TestUpdateProfile_NameOnly(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, nil)

	if err := svc.UpdateProfile(context.Background(), "u1", "Novo Nome", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No password change — hash should not be passed
	if repo.capturedPwdHash != nil {
		t.Error("expected nil password hash for name-only update")
	}
}

func TestUpdateProfile_PasswordChange_NoCurrentPassword(t *testing.T) {
	svc := NewService(&mockRepo{}, nil)
	err := svc.UpdateProfile(context.Background(), "u1", "", "", "novasenha123")
	if err == nil {
		t.Fatal("expected error when new_password set without current_password")
	}
}

func TestUpdateProfile_PasswordChange_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, nil)

	if err := svc.UpdateProfile(context.Background(), "u1", "", "senhaAtual", "novaSenha123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Hash should have been generated and passed to repo
	if repo.capturedPwdHash == nil || *repo.capturedPwdHash == "novaSenha123" {
		t.Error("expected bcrypt hash, not plaintext password")
	}
}

// ---------------------------------------------------------------------------
// ListSellerSearches
// ---------------------------------------------------------------------------

func TestListSellerSearches_Success(t *testing.T) {
	searches := []SearchSummary{
		{SearchID: "s1", Query: "distribuidoras SP", ResultsCount: 200, CreditsUsed: 200},
	}
	svc := NewService(&mockRepo{searches: searches, searchCount: 1}, nil)

	got, total, err := svc.ListSellerSearches(context.Background(), "u1", "org-1", 30, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(got) != 1 || got[0].Query != "distribuidoras SP" {
		t.Errorf("unexpected searches: %v", got)
	}
}

func TestListSellerSearches_RepoError(t *testing.T) {
	svc := NewService(&mockRepo{searchesErr: errors.New("db error")}, nil)
	_, _, err := svc.ListSellerSearches(context.Background(), "u1", "org-1", 30, 1, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
