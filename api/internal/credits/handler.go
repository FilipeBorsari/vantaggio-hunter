package credits

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)

	resp, err := h.svc.GetBalance(r.Context(), orgID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get credit balance", "org_id", orgID, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	resp, err := h.svc.ListTransactions(r.Context(), orgID, page, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "list credit transactions", "org_id", orgID, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

type addCreditsRequest struct {
	OrgID       string `json:"org_id"`
	Amount      int    `json:"amount"`
	Description string `json:"description"`
}

func (h *Handler) AdminAddCredits(w http.ResponseWriter, r *http.Request) {
	var req addCreditsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if req.OrgID == "" {
		httputil.Error(w, http.StatusBadRequest, "org_id é obrigatório")
		return
	}
	if req.Amount <= 0 {
		httputil.Error(w, http.StatusBadRequest, "amount deve ser positivo")
		return
	}

	if err := h.svc.AddCredits(r.Context(), req.OrgID, req.Amount, req.Description); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "organização não encontrada")
			return
		}
		slog.ErrorContext(r.Context(), "admin add credits", "org_id", req.OrgID, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseIntParam(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}
