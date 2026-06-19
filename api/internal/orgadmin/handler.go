package orgadmin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

// ---------- Org Admin: Users ----------

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	users, err := h.svc.ListUsers(r.Context(), orgID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar usuários")
		return
	}
	httputil.JSON(w, http.StatusOK, users)
}

func (h *Handler) PatchUser(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	actorID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	userID := chi.URLParam(r, "userId")

	var body struct {
		IsActive    *bool `json:"is_active"`
		CreditLimit *int  `json:"credit_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}

	if err := h.svc.PatchUser(r.Context(), userID, orgID, actorID, body.IsActive, body.CreditLimit); err != nil {
		if errors.Is(err, ErrSelfDeactivation) {
			httputil.Error(w, http.StatusUnprocessableEntity, "não é possível desativar o próprio usuário")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao atualizar usuário")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	actorID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	userID := chi.URLParam(r, "userId")

	if err := h.svc.DeleteUser(r.Context(), userID, orgID, actorID); err != nil {
		if errors.Is(err, ErrSelfDeactivation) {
			httputil.Error(w, http.StatusUnprocessableEntity, "não é possível remover o próprio usuário")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao remover usuário")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	userID := chi.URLParam(r, "userId")
	days := parsePeriodDays(r.URL.Query().Get("period"), 30)
	page := intParam(r.URL.Query().Get("page"), 1)
	limit := intParam(r.URL.Query().Get("limit"), 20)

	history, err := h.svc.GetUserHistory(r.Context(), userID, orgID, days, page, limit)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar histórico")
		return
	}
	httputil.JSON(w, http.StatusOK, history)
}

// ---------- Org Admin: Invitations ----------

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	actorID, _ := r.Context().Value(auth.ContextKeyUserID).(string)

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}
	if body.Email == "" {
		httputil.Error(w, http.StatusBadRequest, "email é obrigatório")
		return
	}
	if body.Role == "" {
		body.Role = "seller"
	}

	inv, err := h.svc.CreateInvitation(r.Context(), orgID, body.Email, body.Role, actorID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao criar convite")
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]string{
		"invitation_id": inv.ID,
		"expires_at":    inv.ExpiresAt,
	})
}

func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	invs, err := h.svc.ListInvitations(r.Context(), orgID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar convites")
		return
	}
	httputil.JSON(w, http.StatusOK, invs)
}

func (h *Handler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	invID := chi.URLParam(r, "invitationId")

	if err := h.svc.DeleteInvitation(r.Context(), invID, orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "convite não encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao revogar convite")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Org Admin: Costs & Credits ----------

func (h *Handler) GetOrgCosts(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	days := parsePeriodDays(r.URL.Query().Get("period"), 30)

	costs, err := h.svc.GetOrgCosts(r.Context(), orgID, days)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar custos")
		return
	}
	httputil.JSON(w, http.StatusOK, costs)
}

func (h *Handler) GetOrgCredits(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	bal, err := h.svc.GetOrgCredits(r.Context(), orgID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar créditos")
		return
	}
	httputil.JSON(w, http.StatusOK, bal)
}

// ---------- Seller: Profile & Searches ----------

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	profile, err := h.svc.GetSellerProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar perfil")
		return
	}
	httputil.JSON(w, http.StatusOK, profile)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(auth.ContextKeyUserID).(string)

	var body struct {
		Name            string `json:"name"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, http.StatusBadRequest, "body inválido")
		return
	}

	if err := h.svc.UpdateProfile(r.Context(), userID, body.Name, body.CurrentPassword, body.NewPassword); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListSellerSearches(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(auth.ContextKeyUserID).(string)
	orgID, _ := r.Context().Value(auth.ContextKeyOrgID).(string)
	days := parsePeriodDays(r.URL.Query().Get("period"), 30)
	page := intParam(r.URL.Query().Get("page"), 1)
	limit := intParam(r.URL.Query().Get("limit"), 20)

	searches, total, err := h.svc.ListSellerSearches(r.Context(), userID, orgID, days, page, limit)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao listar pesquisas")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"data":  searches,
		"total": total,
		"page":  page,
	})
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
