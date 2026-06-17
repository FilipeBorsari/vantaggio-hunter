package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// makeTestToken signs a JWT with the given claims and expiry using the provided secret.
func makeTestToken(t *testing.T, userID, orgID, role, secret string, ttl time.Duration) string {
	t.Helper()
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return str
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ---------------------------------------------------------------------------
// Authenticate middleware
// ---------------------------------------------------------------------------

func TestAuthenticate_NoHeader(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	Authenticate(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticate_WrongScheme(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	Authenticate(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticate_MalformedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer this-is-not-a-jwt")
	w := httptest.NewRecorder()
	Authenticate(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	t.Setenv("JWT_SECRET", secret)
	token := makeTestToken(t, "u1", "o1", "admin", secret, -time.Hour)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	Authenticate(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticate_WrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "correct-secret")
	token := makeTestToken(t, "u1", "o1", "admin", "wrong-secret", time.Hour)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	Authenticate(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthenticate_ValidToken_SetsContext(t *testing.T) {
	secret := "test-secret"
	t.Setenv("JWT_SECRET", secret)
	token := makeTestToken(t, "user-42", "org-7", "manager", secret, time.Hour)

	var gotUserID, gotOrgID, gotRole string
	capture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = r.Context().Value(ContextKeyUserID).(string)
		gotOrgID, _ = r.Context().Value(ContextKeyOrgID).(string)
		gotRole, _ = r.Context().Value(ContextKeyRole).(string)
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	Authenticate(capture).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if gotUserID != "user-42" {
		t.Errorf("user_id = %q, want user-42", gotUserID)
	}
	if gotOrgID != "org-7" {
		t.Errorf("org_id = %q, want org-7", gotOrgID)
	}
	if gotRole != "manager" {
		t.Errorf("role = %q, want manager", gotRole)
	}
}

// ---------------------------------------------------------------------------
// RequireRole middleware
// ---------------------------------------------------------------------------

func ctxWithRole(r *http.Request, role string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ContextKeyRole, role))
}

func TestRequireRole_AllowedSingleRole(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = ctxWithRole(r, "admin")
	w := httptest.NewRecorder()
	RequireRole("admin")(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireRole_AllowedMultipleRoles(t *testing.T) {
	cases := []string{"admin", "manager"}
	for _, role := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = ctxWithRole(r, role)
		w := httptest.NewRecorder()
		RequireRole("admin", "manager")(okHandler()).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("role %q: status = %d, want 200", role, w.Code)
		}
	}
}

func TestRequireRole_Denied(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = ctxWithRole(r, "operator")
	w := httptest.NewRecorder()
	RequireRole("admin")(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireRole_NoRoleInContext(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// no role set in context
	w := httptest.NewRecorder()
	RequireRole("admin")(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireRole_EmptyAllowedList(t *testing.T) {
	// no roles allowed → every request is denied
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = ctxWithRole(r, "admin")
	w := httptest.NewRecorder()
	RequireRole()(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}
