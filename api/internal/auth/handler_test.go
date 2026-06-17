package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockAuthSvc is a controllable stub for the ServiceInterface.
type mockAuthSvc struct {
	pair *TokenPair
	err  error
}

func (m *mockAuthSvc) Login(_ context.Context, _, _ string) (*TokenPair, error) {
	return m.pair, m.err
}
func (m *mockAuthSvc) Refresh(_ context.Context, _ string) (*TokenPair, error) {
	return m.pair, m.err
}
func (m *mockAuthSvc) Logout(_ context.Context, _ string) error {
	return m.err
}

// ---------------------------------------------------------------------------
// Login handler
// ---------------------------------------------------------------------------

func TestLoginHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{pair: &TokenPair{AccessToken: "acc-tok", RefreshToken: "ref-tok", ExpiresIn: 86400}}
	h := NewHandler(svc)

	body := `{"email":"user@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp TokenPair
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AccessToken != "acc-tok" {
		t.Errorf("access_token = %q, want acc-tok", resp.AccessToken)
	}
	if resp.RefreshToken != "ref-tok" {
		t.Errorf("refresh_token = %q, want ref-tok", resp.RefreshToken)
	}
}

func TestLoginHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockAuthSvc{})
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	cases := []string{
		`{"email":"","password":"pass"}`,
		`{"email":"u@e.com","password":""}`,
		`{}`,
	}
	for _, body := range cases {
		h := NewHandler(&mockAuthSvc{})
		r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		h.Login(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%s: status = %d, want 400", body, w.Code)
		}
	}
}

func TestLoginHandler_WrongCredentials(t *testing.T) {
	svc := &mockAuthSvc{err: ErrInvalidCredentials}
	h := NewHandler(svc)
	body := `{"email":"u@e.com","password":"wrong"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestLoginHandler_OtherServiceError(t *testing.T) {
	svc := &mockAuthSvc{err: errors.New("db error")}
	h := NewHandler(svc)
	body := `{"email":"u@e.com","password":"pass"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Login(w, r)
	// handler maps any service error to 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Refresh handler
// ---------------------------------------------------------------------------

func TestRefreshHandler_Success(t *testing.T) {
	svc := &mockAuthSvc{pair: &TokenPair{AccessToken: "new-acc", RefreshToken: "new-ref"}}
	h := NewHandler(svc)
	body := `{"refresh_token":"old-token"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRefreshHandler_BadBody(t *testing.T) {
	h := NewHandler(&mockAuthSvc{})
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("bad"))
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	h := NewHandler(&mockAuthSvc{})
	body := `{"refresh_token":""}`
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	svc := &mockAuthSvc{err: ErrTokenInvalid}
	h := NewHandler(svc)
	body := `{"refresh_token":"invalid"}`
	r := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Logout handler
// ---------------------------------------------------------------------------

func TestLogoutHandler_Success(t *testing.T) {
	h := NewHandler(&mockAuthSvc{})
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(r.Context(), ContextKeyUserID, "user-1")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Logout(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestLogoutHandler_ServiceError(t *testing.T) {
	svc := &mockAuthSvc{err: errors.New("db error")}
	h := NewHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(r.Context(), ContextKeyUserID, "user-1")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Logout(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestLogoutHandler_NoUserIDInContext(t *testing.T) {
	// Logout should still attempt to call service with empty userID — no panic.
	h := NewHandler(&mockAuthSvc{})
	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}
