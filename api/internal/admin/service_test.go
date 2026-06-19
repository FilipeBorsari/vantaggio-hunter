package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/vantaggio/prospect-api/internal/domain"
)

// mockAdminRepo is a controllable stub for the Repository interface.
type mockAdminRepo struct {
	plans        []domain.Plan
	plansErr     error
	org          *domain.Org
	orgErr       error
	user         *domain.User
	userErr      error
	orgs         []domain.Org
	orgsErr      error
	count        int
	countErr     error
	setActiveErr error
	auditErr     error
}

func (m *mockAdminRepo) ListPlans(_ context.Context) ([]domain.Plan, error) {
	return m.plans, m.plansErr
}
func (m *mockAdminRepo) CreateOrg(_ context.Context, _ string, _ *string) (*domain.Org, error) {
	return m.org, m.orgErr
}
func (m *mockAdminRepo) CreateUser(_ context.Context, _, _, _, _, _ string) (*domain.User, error) {
	return m.user, m.userErr
}
func (m *mockAdminRepo) ListOrgs(_ context.Context, _, _ int, _ string) ([]domain.Org, error) {
	return m.orgs, m.orgsErr
}
func (m *mockAdminRepo) CountOrgs(_ context.Context) (int, error) {
	return m.count, m.countErr
}
func (m *mockAdminRepo) SetUserActive(_ context.Context, _ string, _ bool) error {
	return m.setActiveErr
}
func (m *mockAdminRepo) GetOrgDetail(_ context.Context, _ string) (*domain.OrgDetail, error) {
	return nil, nil
}
func (m *mockAdminRepo) PatchOrg(_ context.Context, _ string, _ *bool, _ *string) error {
	return nil
}
func (m *mockAdminRepo) GetAdminDashboard(_ context.Context, _ int) (*domain.AdminDashboard, error) {
	return &domain.AdminDashboard{Orgs: []domain.OrgSummary{}}, nil
}
func (m *mockAdminRepo) WriteAuditLog(_ context.Context, _ *string, _, _ string, _ *string, _ map[string]any) error {
	return m.auditErr
}

// ---------------------------------------------------------------------------
// ListPlans
// ---------------------------------------------------------------------------

func TestListPlans_ReturnsPlans(t *testing.T) {
	plans := []domain.Plan{
		{ID: "plan-1", Name: "Starter", Credits: 100, PriceCents: 4900},
		{ID: "plan-2", Name: "Pro", Credits: 1000, PriceCents: 19900},
	}
	repo := &mockAdminRepo{plans: plans}
	svc := NewService(repo, nil)

	result, err := svc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
	if result[0].Name != "Starter" {
		t.Errorf("name = %q, want Starter", result[0].Name)
	}
}

func TestListPlans_RepoError(t *testing.T) {
	repo := &mockAdminRepo{plansErr: errors.New("db error")}
	svc := NewService(repo, nil)

	_, err := svc.ListPlans(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser_Success(t *testing.T) {
	user := &domain.User{ID: "user-1", Email: "admin@acme.com", Role: "org_admin"}
	repo := &mockAdminRepo{user: user}
	svc := NewService(repo, nil)

	result, err := svc.CreateUser(context.Background(), "org-1", "Admin", "admin@acme.com", "P@ssw0rd!", "org_admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Email != "admin@acme.com" {
		t.Errorf("email = %q, want admin@acme.com", result.Email)
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	repo := &mockAdminRepo{userErr: ErrEmailAlreadyExists}
	svc := NewService(repo, nil)

	_, err := svc.CreateUser(context.Background(), "org-1", "Admin", "dup@acme.com", "pass", "seller")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("got %v, want ErrEmailAlreadyExists", err)
	}
}

func TestCreateUser_PasswordIsHashed(t *testing.T) {
	var storedHash string
	repo := &mockAdminRepo{user: &domain.User{ID: "u1", Email: "u@e.com", Role: "seller"}}
	captureRepo := &hashCaptureRepo{mockAdminRepo: repo, capturedHash: &storedHash}
	svc := NewService(captureRepo, nil)

	_, err := svc.CreateUser(context.Background(), "org-1", "User", "u@e.com", "plaintext-password", "seller")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storedHash == "plaintext-password" {
		t.Error("password was stored in plaintext — must be hashed")
	}
	if storedHash == "" {
		t.Error("hash is empty")
	}
}

type hashCaptureRepo struct {
	*mockAdminRepo
	capturedHash *string
}

func (r *hashCaptureRepo) CreateUser(_ context.Context, _, _, _, hash, _ string) (*domain.User, error) {
	*r.capturedHash = hash
	return r.mockAdminRepo.user, r.mockAdminRepo.userErr
}

// ---------------------------------------------------------------------------
// ListOrgs
// ---------------------------------------------------------------------------

func TestListOrgs_Success(t *testing.T) {
	orgs := []domain.Org{
		{ID: "org-1", Name: "ACME Corp", IsActive: true},
		{ID: "org-2", Name: "Beta Inc", IsActive: false},
	}
	repo := &mockAdminRepo{orgs: orgs, count: 2}
	svc := NewService(repo, nil)

	result, err := svc.ListOrgs(context.Background(), 1, 20, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(result.Data))
	}
}

func TestListOrgs_ListError(t *testing.T) {
	repo := &mockAdminRepo{orgsErr: errors.New("db error")}
	svc := NewService(repo, nil)

	_, err := svc.ListOrgs(context.Background(), 1, 20, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListOrgs_CountError(t *testing.T) {
	repo := &mockAdminRepo{orgs: []domain.Org{}, countErr: errors.New("count failed")}
	svc := NewService(repo, nil)

	_, err := svc.ListOrgs(context.Background(), 1, 20, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// SetUserActive
// ---------------------------------------------------------------------------

func TestSetUserActive_Activate(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewService(repo, nil)

	if err := svc.SetUserActive(context.Background(), "user-1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserActive_RepoError(t *testing.T) {
	repo := &mockAdminRepo{setActiveErr: errors.New("not found")}
	svc := NewService(repo, nil)

	if err := svc.SetUserActive(context.Background(), "nonexistent", true); err == nil {
		t.Fatal("expected error, got nil")
	}
}
