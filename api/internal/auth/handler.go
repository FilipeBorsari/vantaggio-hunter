package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Email == "" || body.Password == "" {
		httputil.Error(w, http.StatusBadRequest, "email e password são obrigatórios")
		return
	}
	pair, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		httputil.Error(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}
	httputil.JSON(w, http.StatusOK, pair)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.RefreshToken == "" {
		httputil.Error(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}
	pair, err := h.svc.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		httputil.Error(w, http.StatusUnauthorized, "token inválido ou expirado")
		return
	}
	httputil.JSON(w, http.StatusOK, pair)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ContextKeyUserID).(string)
	if err := h.svc.Logout(r.Context(), userID); err != nil {
		slog.ErrorContext(r.Context(), "logout failed", "user_id", userID, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro ao encerrar sessão")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
