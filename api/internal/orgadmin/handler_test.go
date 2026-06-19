package orgadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
)

// ---------------------------------------------------------------------------
// mockSvc: stub para ServiceInterface
// ---------------------------------------------------------------------------

type mockSvc struct {
	users           []domain.OrgUser
	listUsersErr    error
	patchUserErr    error
	deleteUserErr   error
	history         *UserHistory
	historyErr      error
	invitation      *domain.Invitation
	createInvErr    error
	invitations     []domain.Invitation
	listInvErr      error
	deleteInvErr    error
	costs           *domain.OrgCosts
	costsErr        error
	credits         *domain.CreditBalanceResponse
	creditsErr      error
	profile         *domain.SellerProfile
	profileErr      error
	updateProfErr   error
	searches        []SearchSummary
	searchCount     int
	searchesErr     error
}

func (m *mockSvc) ListUsers(_ context.Context, _ string) ([]domain.OrgUser, error) {
	return m.users, m.listUsersErr
}
func (m *mockSvc) PatchUser(_ context.Context, _, _, _ string, _ *bool, _ *int) error {
	return m.patchUserErr
}
func (m *mockSvc) DeleteUser(_ context.Context, _, _, _ string) error {
	return m.deleteUserErr
}
func (m *mockSvc) CreateInvitation(_ context.Context, _, _, _, _ string) (*domain.Invitation, error) {
	return m.invitation, m.createInvErr
}
func (m *mockSvc) ListInvitations(_ context.Context, _ string) ([]domain.Invitation, error) {
	return m.invitations, m.listInvErr
}
func (m *mockSvc) DeleteInvitation(_ context.Context, _, _ string) error {
	return m.deleteInvErr
}
func (m *mockSvc) GetUserHistory(_ context.Context, _, _ string, _, _, _ int) (*UserHistory, error) {
	return m.history, m.historyErr
}
func (m *mockSvc) GetOrgCosts(_ context.Context, _ string, _ int) (*domain.OrgCosts, error) {
	return m.costs, m.costsErr
}
func (m *mockSvc) GetOrgCredits(_ context.Context, _ string) (*domain.CreditBalanceResponse, error) {
	return m.credits, m.creditsErr
}
func (m *mockSvc) GetSellerProfile(_ context.Context, _ string) (*domain.SellerProfile, error) {
	return m.profile, m.profileErr
}
func (m *mockSvc) UpdateProfile(_ context.Context, _, _, _, _ string) error {
	return m.updateProfErr
}
func (m *mockSvc) ListSellerSearches(_ context.Context, _, _ string, _, _, _ int) ([]SearchSummary, int, error) {
	return m.searches, m.searchCount, m.searchesErr
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// withAuthCtx injeta orgID e userID no contexto (simula autenticação).
func withAuthCtx(r *http.Request, orgID, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.ContextKeyOrgID, orgID)
	ctx = context.WithValue(ctx, auth.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// withChiParam injeta um parâmetro de rota chi no contexto.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withChiAndAuth combina injeção de rota e auth.
func withChiAndAuth(r *http.Request, orgID, userID, paramKey, paramVal string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramKey, paramVal)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.ContextKeyOrgID, orgID)
	ctx = context.WithValue(ctx, auth.ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsersHandler_Success(t *testing.T) {
	users := []domain.OrgUser{{UserID: "u1", Name: "Ana", IsActive: true}}
	h := NewHandler(&mockSvc{users: users})

	r := httptest.NewRequest(http.MethodGet, "/org/users", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.ListUsers(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body []domain.OrgUser
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].Name != "Ana" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestListUsersHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{listUsersErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/org/users", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.ListUsers(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// PatchUser
// ---------------------------------------------------------------------------

func TestPatchUserHandler_Success(t *testing.T) {
	h := NewHandler(&mockSvc{})
	body := `{"is_active":false}`
	r := httptest.NewRequest(http.MethodPatch, "/org/users/u1", bytes.NewBufferString(body))
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u1")
	w := httptest.NewRecorder()
	h.PatchUser(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestPatchUserHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockSvc{})
	r := httptest.NewRequest(http.MethodPatch, "/org/users/u1", bytes.NewBufferString("bad-json"))
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u1")
	w := httptest.NewRecorder()
	h.PatchUser(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPatchUserHandler_SelfDeactivation(t *testing.T) {
	h := NewHandler(&mockSvc{patchUserErr: ErrSelfDeactivation})
	body := `{"is_active":false}`
	r := httptest.NewRequest(http.MethodPatch, "/org/users/same", bytes.NewBufferString(body))
	r = withChiAndAuth(r, "org-1", "same", "userId", "same")
	w := httptest.NewRecorder()
	h.PatchUser(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestPatchUserHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockSvc{patchUserErr: domain.ErrNotFound})
	body := `{"is_active":true}`
	r := httptest.NewRequest(http.MethodPatch, "/org/users/u9", bytes.NewBufferString(body))
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u9")
	w := httptest.NewRecorder()
	h.PatchUser(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// DeleteUser
// ---------------------------------------------------------------------------

func TestDeleteUserHandler_Success(t *testing.T) {
	h := NewHandler(&mockSvc{})
	r := httptest.NewRequest(http.MethodDelete, "/org/users/u1", nil)
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u1")
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestDeleteUserHandler_SelfDelete(t *testing.T) {
	h := NewHandler(&mockSvc{deleteUserErr: ErrSelfDeactivation})
	r := httptest.NewRequest(http.MethodDelete, "/org/users/same", nil)
	r = withChiAndAuth(r, "org-1", "same", "userId", "same")
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestDeleteUserHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockSvc{deleteUserErr: domain.ErrNotFound})
	r := httptest.NewRequest(http.MethodDelete, "/org/users/u9", nil)
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u9")
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetUserHistory
// ---------------------------------------------------------------------------

func TestGetUserHistoryHandler_Success(t *testing.T) {
	hist := &UserHistory{
		User:     domain.OrgUser{UserID: "u1", Name: "Ana"},
		Stats:    UserHistStats{Searches: 10, CreditsConsumed: 100},
		Searches: []SearchSummary{},
	}
	h := NewHandler(&mockSvc{history: hist})
	r := httptest.NewRequest(http.MethodGet, "/org/users/u1/history", nil)
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u1")
	w := httptest.NewRecorder()
	h.GetUserHistory(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetUserHistoryHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockSvc{historyErr: domain.ErrNotFound})
	r := httptest.NewRequest(http.MethodGet, "/org/users/u9/history", nil)
	r = withChiAndAuth(r, "org-1", "actor", "userId", "u9")
	w := httptest.NewRecorder()
	h.GetUserHistory(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateInvitation
// ---------------------------------------------------------------------------

func TestCreateInvitationHandler_Success(t *testing.T) {
	inv := &domain.Invitation{ID: "inv-1", ExpiresAt: "2030-01-01T00:00:00Z"}
	h := NewHandler(&mockSvc{invitation: inv})
	body := `{"email":"v@co.com","role":"seller"}`
	r := httptest.NewRequest(http.MethodPost, "/org/invitations", bytes.NewBufferString(body))
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.CreateInvitation(w, r)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestCreateInvitationHandler_MissingEmail(t *testing.T) {
	h := NewHandler(&mockSvc{})
	body := `{"email":"","role":"seller"}`
	r := httptest.NewRequest(http.MethodPost, "/org/invitations", bytes.NewBufferString(body))
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.CreateInvitation(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateInvitationHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockSvc{})
	r := httptest.NewRequest(http.MethodPost, "/org/invitations", bytes.NewBufferString("bad"))
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.CreateInvitation(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ListInvitations
// ---------------------------------------------------------------------------

func TestListInvitationsHandler_Success(t *testing.T) {
	invs := []domain.Invitation{{ID: "inv-1", Email: "a@co.com"}}
	h := NewHandler(&mockSvc{invitations: invs})
	r := httptest.NewRequest(http.MethodGet, "/org/invitations", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.ListInvitations(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body []domain.Invitation
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("len = %d, want 1", len(body))
	}
}

func TestListInvitationsHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{listInvErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/org/invitations", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.ListInvitations(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// DeleteInvitation
// ---------------------------------------------------------------------------

func TestDeleteInvitationHandler_Success(t *testing.T) {
	h := NewHandler(&mockSvc{})
	r := httptest.NewRequest(http.MethodDelete, "/org/invitations/inv-1", nil)
	r = withChiAndAuth(r, "org-1", "actor", "invitationId", "inv-1")
	w := httptest.NewRecorder()
	h.DeleteInvitation(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestDeleteInvitationHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockSvc{deleteInvErr: domain.ErrNotFound})
	r := httptest.NewRequest(http.MethodDelete, "/org/invitations/inv-9", nil)
	r = withChiAndAuth(r, "org-1", "actor", "invitationId", "inv-9")
	w := httptest.NewRecorder()
	h.DeleteInvitation(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetOrgCosts
// ---------------------------------------------------------------------------

func TestGetOrgCostsHandler_Success(t *testing.T) {
	costs := &domain.OrgCosts{Period: "30d", TotalCreditsConsumed: 800, BySeller: []domain.SellerCost{}}
	h := NewHandler(&mockSvc{costs: costs})
	r := httptest.NewRequest(http.MethodGet, "/org/costs", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.GetOrgCosts(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body domain.OrgCosts
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.TotalCreditsConsumed != 800 {
		t.Errorf("total = %d, want 800", body.TotalCreditsConsumed)
	}
}

func TestGetOrgCostsHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{costsErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/org/costs", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.GetOrgCosts(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetOrgCredits
// ---------------------------------------------------------------------------

func TestGetOrgCreditsHandler_Success(t *testing.T) {
	bal := &domain.CreditBalanceResponse{Balance: 4200, OrgID: "org-1"}
	h := NewHandler(&mockSvc{credits: bal})
	r := httptest.NewRequest(http.MethodGet, "/org/credits", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.GetOrgCredits(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body domain.CreditBalanceResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Balance != 4200 {
		t.Errorf("balance = %d, want 4200", body.Balance)
	}
}

func TestGetOrgCreditsHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{creditsErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/org/credits", nil)
	r = withAuthCtx(r, "org-1", "actor")
	w := httptest.NewRecorder()
	h.GetOrgCredits(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetProfile
// ---------------------------------------------------------------------------

func TestGetProfileHandler_Success(t *testing.T) {
	p := &domain.SellerProfile{UserID: "u1", Name: "Ana", OrgName: "Corp", CreditsConsumedMonth: 42}
	h := NewHandler(&mockSvc{profile: p})
	r := httptest.NewRequest(http.MethodGet, "/me/profile", nil)
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.GetProfile(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body domain.SellerProfile
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "Ana" {
		t.Errorf("name = %q, want Ana", body.Name)
	}
}

func TestGetProfileHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockSvc{profileErr: domain.ErrNotFound})
	r := httptest.NewRequest(http.MethodGet, "/me/profile", nil)
	r = withAuthCtx(r, "org-1", "u9")
	w := httptest.NewRecorder()
	h.GetProfile(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// UpdateProfile
// ---------------------------------------------------------------------------

func TestUpdateProfileHandler_Success(t *testing.T) {
	h := NewHandler(&mockSvc{})
	body := `{"name":"Novo Nome"}`
	r := httptest.NewRequest(http.MethodPatch, "/me/profile", bytes.NewBufferString(body))
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.UpdateProfile(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestUpdateProfileHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockSvc{})
	r := httptest.NewRequest(http.MethodPatch, "/me/profile", bytes.NewBufferString("bad-json"))
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.UpdateProfile(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUpdateProfileHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{updateProfErr: errors.New("senha atual inválida")})
	body := `{"new_password":"nova","current_password":""}`
	r := httptest.NewRequest(http.MethodPatch, "/me/profile", bytes.NewBufferString(body))
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.UpdateProfile(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ListSellerSearches
// ---------------------------------------------------------------------------

func TestListSellerSearchesHandler_Success(t *testing.T) {
	searches := []SearchSummary{{SearchID: "s1", Query: "padarias SP", ResultsCount: 50}}
	h := NewHandler(&mockSvc{searches: searches, searchCount: 1})
	r := httptest.NewRequest(http.MethodGet, "/me/searches", nil)
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.ListSellerSearches(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestListSellerSearchesHandler_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{searchesErr: errors.New("db error")})
	r := httptest.NewRequest(http.MethodGet, "/me/searches", nil)
	r = withAuthCtx(r, "org-1", "u1")
	w := httptest.NewRecorder()
	h.ListSellerSearches(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// intParam / parsePeriodDays (utilitários internos)
// ---------------------------------------------------------------------------

func TestIntParam(t *testing.T) {
	cases := []struct{ input string; def, want int }{
		{"", 1, 1}, {"0", 10, 10}, {"-1", 1, 1}, {"abc", 1, 1}, {"42", 1, 42},
	}
	for _, tc := range cases {
		if got := intParam(tc.input, tc.def); got != tc.want {
			t.Errorf("intParam(%q,%d) = %d, want %d", tc.input, tc.def, got, tc.want)
		}
	}
}

func TestParsePeriodDays(t *testing.T) {
	cases := []struct{ period string; want int }{
		{"7d", 7}, {"30d", 30}, {"90d", 90}, {"", 30}, {"invalid", 30},
	}
	for _, tc := range cases {
		if got := parsePeriodDays(tc.period, 30); got != tc.want {
			t.Errorf("parsePeriodDays(%q) = %d, want %d", tc.period, got, tc.want)
		}
	}
}
