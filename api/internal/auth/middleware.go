package auth

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyOrgID  contextKey = "org_id"
	ContextKeyRole   contextKey = "role"
)

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			slog.DebugContext(r.Context(), "auth: no bearer token", "path", r.URL.Path)
			http.Error(w, `{"error":"não autorizado"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims := &Claims{}
		_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrTokenInvalid
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil {
			slog.WarnContext(r.Context(), "auth: invalid or expired token", "path", r.URL.Path, "error", err)
			http.Error(w, `{"error":"token inválido ou expirado"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyOrgID, claims.OrgID)
		ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
		slog.DebugContext(ctx, "auth: ok", "user_id", claims.UserID, "org_id", claims.OrgID, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextKeyRole).(string)
			if !allowed[role] {
				http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSuperAdmin allows only super_admin.
func RequireSuperAdmin() func(http.Handler) http.Handler {
	return RequireRole("super_admin")
}

// RequireOrgAdmin allows only org_admin. It does NOT enforce org_id matching —
// handlers must compare chi.URLParam("orgId") against ContextKeyOrgID themselves.
func RequireOrgAdmin() func(http.Handler) http.Handler {
	return RequireRole("org_admin")
}
