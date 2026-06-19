package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// mockAdminSvc is a controllable stub for the ServiceInterface.
type mockAdminSvc struct {
	plans        []domain.Plan
	plansErr     error
	createResult *CreateOrgResult
	createOrgErr error
	user         *domain.User
	userErr      error
	orgList      *domain.OrgListResponse
	orgListErr   error
	setActiveErr error
	orgDetail    *domain.OrgDetail
	orgDetailErr error
	dashboard    *domain.AdminDashboard
	dashErr      error
}

func (m *mockAdminSvc) ListPlans(_ context.Context) ([]domain.Plan, error) {
	return m.plans, m.plansErr
}
func (m *mockAdminSvc) CreateOrgWithAdmin(_ context.Context, _ string, _ *string, _, _ string) (*CreateOrgResult, error) {
	return m.createResult, m.createOrgErr
}
func (m *mockAdminSvc) CreateUser(_ context.Context, _, _, _, _, _ string) (*domain.User, error) {
	return m.user, m.userErr
}
func (m *mockAdminSvc) ListOrgs(_ context.Context, _, _ int, _ string) (*domain.OrgListResponse, error) {
	return m.orgList, m.orgListErr
}
func (m *mockAdminSvc) SetUserActive(_ context.Context, _ string, _ bool) error {
	return m.setActiveErr
}
func (m *mockAdminSvc) GetOrgDetail(_ context.Context, _ string) (*domain.OrgDetail, error) {
	return m.orgDetail, m.orgDetailErr
}
func (m *mockAdminSvc) PatchOrg(_ context.Context, _ string, _ *bool, _ *string) error {
	return nil
}
func (m *mockAdminSvc) GetAdminDashboard(_ context.Context, _ int) (*domain.AdminDashboard, error) {
	return m.dashboard, m.dashErr
}
func (m *mockAdminSvc) AddCreditsToOrg(_ context.Context, _ string, _ int, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockAdminSvc) Impersonate(_ context.Context, _, _ string) (string, error) {
	return "tok", nil
}
func (m *mockAdminSvc) WriteAuditLog(_ context.Context, _ *string, _, _ string, _ *string, _ map[string]any) error {
	return nil
}

// withChiParam injects a chi route param into the request context.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withChiParams injects multiple chi route params into the request context.
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// ListPlans handler
// ---------------------------------------------------------------------------

func TestListPlansHandler_Success(t *testing.T) {
	plans := []domain.Plan{
		{ID: "p1", Name: "Starter", Credits: 100, PriceCents: 4900},
	}
	h := NewHandler(&mockAdminSvc{plans: plans})
	r := httptest.NewRequest(http.MethodGet, "/admin/plans", nil)
	w := httptest.NewRecorder()
	h.ListPlans(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body []domain.Plan
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].Name != "Starter" {
		t.Errorf("body = %v", body)
	}
}

func TestListPlansHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockAdminSvc{plansErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/admin/plans", nil)
	w := httptest.NewRecorder()
	h.ListPlans(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateOrg handler
// ---------------------------------------------------------------------------

func TestCreateOrgHandler_Success(t *testing.T) {
	result := &CreateOrgResult{OrgID: "org-1", UserID: "u-1", TempPassword: "tmp"}
	h := NewHandler(&mockAdminSvc{createResult: result})

	body := `{"name":"New Corp","admin_email":"admin@corp.com"}`
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateOrg(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestCreateOrgHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockAdminSvc{})
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h.CreateOrg(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateOrgHandler_MissingName(t *testing.T) {
	h := NewHandler(&mockAdminSvc{})
	body := `{"name":"","admin_email":"a@b.com"}`
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateOrg(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateOrgHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockAdminSvc{createOrgErr: errors.New("db error")})
	body := `{"name":"Fail Corp","admin_email":"a@b.com"}`
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateOrg(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateUser handler
// ---------------------------------------------------------------------------

func TestCreateUserHandler_Success(t *testing.T) {
	user := &domain.User{ID: "u1", Email: "user@acme.com", Role: "seller"}
	h := NewHandler(&mockAdminSvc{user: user})

	body := `{"email":"user@acme.com","password":"P@ss1234","role":"seller"}`
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations/org-1/users", bytes.NewBufferString(body))
	r = withChiParam(r, "id", "org-1")
	w := httptest.NewRecorder()
	h.CreateUser(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestCreateUserHandler_EmailAlreadyExists(t *testing.T) {
	h := NewHandler(&mockAdminSvc{userErr: ErrEmailAlreadyExists})
	body := `{"email":"dup@acme.com","password":"pass","role":"seller"}`
	r := httptest.NewRequest(http.MethodPost, "/admin/organizations/org-1/users", bytes.NewBufferString(body))
	r = withChiParam(r, "id", "org-1")
	w := httptest.NewRecorder()
	h.CreateUser(w, r)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ListOrgs handler
// ---------------------------------------------------------------------------

func TestListOrgsHandler_Success(t *testing.T) {
	resp := &domain.OrgListResponse{
		Data:  []domain.Org{{ID: "org-1", Name: "ACME"}},
		Total: 1,
	}
	h := NewHandler(&mockAdminSvc{orgList: resp})
	r := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	w := httptest.NewRecorder()
	h.ListOrgs(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body domain.OrgListResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 {
		t.Errorf("total = %d, want 1", body.Total)
	}
}

func TestListOrgsHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockAdminSvc{orgListErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	w := httptest.NewRecorder()
	h.ListOrgs(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// SetUserActive handler
// ---------------------------------------------------------------------------

func TestSetUserActiveHandler_Activate(t *testing.T) {
	h := NewHandler(&mockAdminSvc{})
	body := `{"is_active":true}`
	r := httptest.NewRequest(http.MethodPatch, "/admin/organizations/org-1/users/user-1", bytes.NewBufferString(body))
	r = withChiParams(r, map[string]string{"id": "org-1", "userId": "user-1"})
	w := httptest.NewRecorder()
	h.SetUserActive(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestSetUserActiveHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockAdminSvc{setActiveErr: errors.New("user not found")})
	body := `{"is_active":true}`
	r := httptest.NewRequest(http.MethodPatch, "/admin/organizations/org-1/users/user-1", bytes.NewBufferString(body))
	r = withChiParams(r, map[string]string{"id": "org-1", "userId": "user-1"})
	w := httptest.NewRecorder()
	h.SetUserActive(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// intParam
// ---------------------------------------------------------------------------

func TestIntParam(t *testing.T) {
	cases := []struct {
		input string
		def   int
		want  int
	}{
		{"", 1, 1},
		{"0", 10, 10},
		{"-5", 1, 1},
		{"abc", 1, 1},
		{"42", 1, 42},
		{"100", 20, 100},
	}
	for _, tc := range cases {
		got := intParam(tc.input, tc.def)
		if got != tc.want {
			t.Errorf("intParam(%q, %d) = %d, want %d", tc.input, tc.def, got, tc.want)
		}
	}
}
