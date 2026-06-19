package invitations

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	rec, err := h.svc.ValidateToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "convite não encontrado ou já usado")
			return
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			httputil.Error(w, http.StatusGone, "convite expirado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao validar convite")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{
		"email":    rec.Email,
		"org_name": rec.OrgName,
		"role":     rec.Role,
	})
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Name == "" || body.Password == "" {
		httputil.Error(w, http.StatusBadRequest, "name e password são obrigatórios")
		return
	}

	accessToken, err := h.svc.Accept(r.Context(), token, body.Name, body.Password)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "convite não encontrado ou já usado")
			return
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			httputil.Error(w, http.StatusGone, "convite expirado")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			httputil.Error(w, http.StatusConflict, "email já cadastrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao aceitar convite")
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]string{"access_token": accessToken})
}
