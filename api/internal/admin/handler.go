package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/internal/auth"
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
		Name       string  `json:"name"`
		PlanID     *string `json:"plan_id"`
		AdminEmail string  `json:"admin_email"`
		AdminName  string  `json:"admin_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Name == "" || body.AdminEmail == "" {
		httputil.Error(w, http.StatusBadRequest, "name e admin_email são obrigatórios")
		return
	}
	if body.AdminName == "" {
		body.AdminName = body.AdminEmail
	}

	result, err := h.svc.CreateOrgWithAdmin(r.Context(), body.Name, body.PlanID, body.AdminEmail, body.AdminName)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			httputil.Error(w, http.StatusConflict, "email já existe")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao criar organização")
		return
	}
	httputil.JSON(w, http.StatusCreated, result)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	var body struct {
		Name     string `json:"name"`
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
	if body.Name == "" {
		body.Name = body.Email
	}
	user, err := h.svc.CreateUser(r.Context(), orgID, body.Name, body.Email, body.Password, body.Role)
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
	q := r.URL.Query().Get("q")
	list, err := h.svc.ListOrgs(r.Context(), page, limit, q)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar organizações")
		return
	}
	httputil.JSON(w, http.StatusOK, list)
}

func (h *Handler) GetOrgDetail(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
	detail, err := h.svc.GetOrgDetail(r.Context(), orgID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar organização")
		return
	}
	httputil.JSON(w, http.StatusOK, detail)
}

func (h *Handler) PatchOrg(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
	var body struct {
		IsActive *bool   `json:"is_active"`
		PlanID   *string `json:"plan_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if err := h.svc.PatchOrg(r.Context(), orgID, body.IsActive, body.PlanID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao atualizar organização")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	days := parsePeriodDays(r.URL.Query().Get("period"), 30)
	dash, err := h.svc.GetAdminDashboard(r.Context(), days)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar dashboard")
		return
	}
	httputil.JSON(w, http.StatusOK, dash)
}

func (h *Handler) AddOrgCredits(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
	actorID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	var body struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Amount <= 0 {
		httputil.Error(w, http.StatusBadRequest, "amount deve ser positivo")
		return
	}
	newBalance, err := h.svc.AddCreditsToOrg(r.Context(), orgID, body.Amount, body.Description, actorID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao adicionar créditos")
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]int{"new_balance": newBalance})
}

func (h *Handler) Impersonate(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
	actorID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	token, err := h.svc.Impersonate(r.Context(), orgID, actorID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao criar token de impersonação")
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]string{"access_token": token})
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

func parsePeriodDays(period string, def int) int {
	switch period {
	case "7d":
		return 7
	case "30d":
		return 30
	case "90d":
		return 90
	default:
		return def
	}
}
