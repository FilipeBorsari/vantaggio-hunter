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
}

func (m *mockAdminRepo) ListPlans(_ context.Context) ([]domain.Plan, error) {
	return m.plans, m.plansErr
}
func (m *mockAdminRepo) CreateOrg(_ context.Context, _ string, _ *string) (*domain.Org, error) {
	return m.org, m.orgErr
}
func (m *mockAdminRepo) CreateUser(_ context.Context, _, _, _, _ string) (*domain.User, error) {
	return m.user, m.userErr
}
func (m *mockAdminRepo) ListOrgs(_ context.Context, _, _ int) ([]domain.Org, error) {
	return m.orgs, m.orgsErr
}
func (m *mockAdminRepo) CountOrgs(_ context.Context) (int, error) {
	return m.count, m.countErr
}
func (m *mockAdminRepo) SetUserActive(_ context.Context, _ string, _ bool) error {
	return m.setActiveErr
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
	svc := NewService(repo)

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
	svc := NewService(repo)

	_, err := svc.ListPlans(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateOrg
// ---------------------------------------------------------------------------

func TestCreateOrg_Success(t *testing.T) {
	planID := "plan-1"
	org := &domain.Org{ID: "org-1", Name: "ACME Corp", PlanID: &planID, IsActive: true}
	repo := &mockAdminRepo{org: org}
	svc := NewService(repo)

	result, err := svc.CreateOrg(context.Background(), "ACME Corp", &planID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "ACME Corp" {
		t.Errorf("name = %q, want ACME Corp", result.Name)
	}
}

func TestCreateOrg_NoPlan(t *testing.T) {
	org := &domain.Org{ID: "org-2", Name: "No Plan Org"}
	repo := &mockAdminRepo{org: org}
	svc := NewService(repo)

	result, err := svc.CreateOrg(context.Background(), "No Plan Org", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PlanID != nil {
		t.Errorf("plan_id = %v, want nil", result.PlanID)
	}
}

func TestCreateOrg_RepoError(t *testing.T) {
	repo := &mockAdminRepo{orgErr: errors.New("db error")}
	svc := NewService(repo)

	_, err := svc.CreateOrg(context.Background(), "Fail Org", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser_Success(t *testing.T) {
	user := &domain.User{ID: "user-1", Email: "admin@acme.com", Role: "admin"}
	repo := &mockAdminRepo{user: user}
	svc := NewService(repo)

	result, err := svc.CreateUser(context.Background(), "org-1", "admin@acme.com", "P@ssw0rd!", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Email != "admin@acme.com" {
		t.Errorf("email = %q, want admin@acme.com", result.Email)
	}
	if result.Role != "admin" {
		t.Errorf("role = %q, want admin", result.Role)
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	repo := &mockAdminRepo{userErr: ErrEmailAlreadyExists}
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), "org-1", "dup@acme.com", "pass", "operator")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("got %v, want ErrEmailAlreadyExists", err)
	}
}

func TestCreateUser_RepoError(t *testing.T) {
	repo := &mockAdminRepo{userErr: errors.New("constraint violation")}
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), "org-1", "u@e.com", "pass", "operator")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateUser_PasswordIsHashed(t *testing.T) {
	// Verify the service never stores the plaintext password.
	var storedHash string
	repo := &mockAdminRepo{}
	repo.user = &domain.User{ID: "u1", Email: "u@e.com", Role: "operator"}
	// Override CreateUser to capture the hash
	captureRepo := &hashCaptureRepo{mockAdminRepo: repo, capturedHash: &storedHash}
	svc := NewService(captureRepo)

	_, err := svc.CreateUser(context.Background(), "org-1", "u@e.com", "plaintext-password", "operator")
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

// hashCaptureRepo wraps mockAdminRepo and captures the password hash passed to CreateUser.
type hashCaptureRepo struct {
	*mockAdminRepo
	capturedHash *string
}

func (r *hashCaptureRepo) CreateUser(_ context.Context, _, _, hash, _ string) (*domain.User, error) {
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
	svc := NewService(repo)

	result, err := svc.ListOrgs(context.Background(), 1, 20)
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
	svc := NewService(repo)

	_, err := svc.ListOrgs(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListOrgs_CountError(t *testing.T) {
	repo := &mockAdminRepo{orgs: []domain.Org{}, countErr: errors.New("count failed")}
	svc := NewService(repo)

	_, err := svc.ListOrgs(context.Background(), 1, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// SetUserActive
// ---------------------------------------------------------------------------

func TestSetUserActive_Activate(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewService(repo)

	if err := svc.SetUserActive(context.Background(), "user-1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserActive_Deactivate(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewService(repo)

	if err := svc.SetUserActive(context.Background(), "user-1", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserActive_RepoError(t *testing.T) {
	repo := &mockAdminRepo{setActiveErr: errors.New("not found")}
	svc := NewService(repo)

	if err := svc.SetUserActive(context.Background(), "nonexistent", true); err == nil {
		t.Fatal("expected error, got nil")
	}
}
