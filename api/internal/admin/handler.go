package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar planos")
		return
	}
	httputil.JSON(w, http.StatusOK, plans)
}

func (h *Handler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string  `json:"name"`
		PlanID *string `json:"plan_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Name == "" {
		httputil.Error(w, http.StatusBadRequest, "name é obrigatório")
		return
	}
	org, err := h.svc.CreateOrg(r.Context(), body.Name, body.PlanID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao criar organização")
		return
	}
	httputil.JSON(w, http.StatusCreated, org)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Email == "" || body.Password == "" || body.Role == "" {
		httputil.Error(w, http.StatusBadRequest, "email, password e role são obrigatórios")
		return
	}
	user, err := h.svc.CreateUser(r.Context(), orgID, body.Email, body.Password, body.Role)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			httputil.Error(w, http.StatusConflict, "email já existe")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	httputil.JSON(w, http.StatusCreated, user)
}

func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	page := intParam(r.URL.Query().Get("page"), 1)
	limit := intParam(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	list, err := h.svc.ListOrgs(r.Context(), page, limit)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar organizações")
		return
	}
	httputil.JSON(w, http.StatusOK, list)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := h.svc.SetUserActive(r.Context(), userID, body.IsActive); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao atualizar usuário")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}
