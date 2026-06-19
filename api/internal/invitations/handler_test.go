package invitations

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

// ---------------------------------------------------------------------------
// mockInvSvc: stub para ServiceInterface
// ---------------------------------------------------------------------------

type mockInvSvc struct {
	record    *InvitationRecord
	validateErr error
	accessToken string
	acceptErr   error
}

func (m *mockInvSvc) ValidateToken(_ context.Context, _ string) (*InvitationRecord, error) {
	return m.record, m.validateErr
}
func (m *mockInvSvc) Accept(_ context.Context, _, _, _ string) (string, error) {
	return m.accessToken, m.acceptErr
}

// ---------------------------------------------------------------------------
// helper: injeta token como parâmetro de rota chi
// ---------------------------------------------------------------------------

func withToken(r *http.Request, token string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("token", token)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// ValidateToken handler
// ---------------------------------------------------------------------------

func TestValidateTokenHandler_Success(t *testing.T) {
	rec := &InvitationRecord{Email: "v@co.com", OrgName: "Corp", Role: "seller"}
	h := NewHandler(&mockInvSvc{record: rec})

	r := httptest.NewRequest(http.MethodGet, "/invitations/tok123", nil)
	r = withToken(r, "tok123")
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["email"] != "v@co.com" {
		t.Errorf("email = %q, want v@co.com", body["email"])
	}
	if body["org_name"] != "Corp" {
		t.Errorf("org_name = %q, want Corp", body["org_name"])
	}
	if body["role"] != "seller" {
		t.Errorf("role = %q, want seller", body["role"])
	}
}

func TestValidateTokenHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockInvSvc{validateErr: domain.ErrNotFound})

	r := httptest.NewRequest(http.MethodGet, "/invitations/bad", nil)
	r = withToken(r, "bad")
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestValidateTokenHandler_Expired(t *testing.T) {
	h := NewHandler(&mockInvSvc{validateErr: domain.ErrTokenExpired})

	r := httptest.NewRequest(http.MethodGet, "/invitations/exp", nil)
	r = withToken(r, "exp")
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusGone {
		t.Errorf("status = %d, want 410", w.Code)
	}
}

func TestValidateTokenHandler_InternalError(t *testing.T) {
	h := NewHandler(&mockInvSvc{validateErr: errors.New("db error")})

	r := httptest.NewRequest(http.MethodGet, "/invitations/tok", nil)
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Accept handler
// ---------------------------------------------------------------------------

func TestAcceptHandler_Success(t *testing.T) {
	h := NewHandler(&mockInvSvc{accessToken: "jwt-tok"})

	body := `{"name":"Vendedor","password":"senhaForte!"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/tok123/accept", bytes.NewBufferString(body))
	r = withToken(r, "tok123")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["access_token"] != "jwt-tok" {
		t.Errorf("access_token = %q, want jwt-tok", resp["access_token"])
	}
}

func TestAcceptHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockInvSvc{})

	r := httptest.NewRequest(http.MethodPost, "/invitations/tok/accept", bytes.NewBufferString("not-json"))
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAcceptHandler_MissingName(t *testing.T) {
	h := NewHandler(&mockInvSvc{})

	body := `{"name":"","password":"senha123"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/tok/accept", bytes.NewBufferString(body))
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAcceptHandler_MissingPassword(t *testing.T) {
	h := NewHandler(&mockInvSvc{})

	body := `{"name":"Fulano","password":""}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/tok/accept", bytes.NewBufferString(body))
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAcceptHandler_NotFound(t *testing.T) {
	h := NewHandler(&mockInvSvc{acceptErr: domain.ErrNotFound})

	body := `{"name":"Fulano","password":"senha123"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/bad/accept", bytes.NewBufferString(body))
	r = withToken(r, "bad")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAcceptHandler_Expired(t *testing.T) {
	h := NewHandler(&mockInvSvc{acceptErr: domain.ErrTokenExpired})

	body := `{"name":"Fulano","password":"senha123"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/exp/accept", bytes.NewBufferString(body))
	r = withToken(r, "exp")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusGone {
		t.Errorf("status = %d, want 410", w.Code)
	}
}

func TestAcceptHandler_EmailConflict(t *testing.T) {
	h := NewHandler(&mockInvSvc{acceptErr: domain.ErrConflict})

	body := `{"name":"Fulano","password":"senha123"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/tok/accept", bytes.NewBufferString(body))
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestAcceptHandler_InternalError(t *testing.T) {
	h := NewHandler(&mockInvSvc{acceptErr: errors.New("db error")})

	body := `{"name":"Fulano","password":"senha123"}`
	r := httptest.NewRequest(http.MethodPost, "/invitations/tok/accept", bytes.NewBufferString(body))
	r = withToken(r, "tok")
	w := httptest.NewRecorder()
	h.Accept(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
