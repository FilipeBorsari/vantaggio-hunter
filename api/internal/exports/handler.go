package exports

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/brazil"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

// Queuer enqueues an export ID for async processing.
type Queuer interface {
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
}

type Handler struct {
	svc   ServiceInterface
	queue Queuer
}

func NewHandler(svc ServiceInterface, queue Queuer) *Handler {
	return &Handler{svc: svc, queue: queue}
}

// POST /crm/integrations
type createIntegrationRequest struct {
	CRMType   string `json:"crm_type"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	InboxID   *int   `json:"inbox_id"`
	AccountID int    `json:"account_id"`
}

func (h *Handler) CreateIntegration(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	role, _ := r.Context().Value(authpkg.ContextKeyRole).(string)
	if role != "admin" && role != "manager" {
		httputil.Error(w, http.StatusForbidden, "acesso negado")
		return
	}

	var req createIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if req.BaseURL == "" || req.APIKey == "" {
		httputil.Error(w, http.StatusBadRequest, "base_url e api_key são obrigatórios")
		return
	}
	if req.CRMType == "" {
		req.CRMType = "chatwoot"
	}
	if req.AccountID == 0 {
		req.AccountID = 1
	}

	intg, err := h.svc.CreateIntegration(r.Context(), orgID, req.CRMType, req.BaseURL, req.APIKey, req.InboxID, req.AccountID)
	if err != nil {
		slog.ErrorContext(r.Context(), "create integration", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusCreated, intg)
}

// GET /crm/integrations
func (h *Handler) GetIntegration(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)

	intg, err := h.svc.GetIntegration(r.Context(), orgID)
	if errors.Is(err, domain.ErrNotFound) {
		httputil.Error(w, http.StatusNotFound, "integração CRM não encontrada")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get integration", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, intg)
}

// POST /exports
type createExportRequest struct {
	CNPJs    []string `json:"cnpjs"`
	SearchID *string  `json:"search_id"`
}

func (h *Handler) CreateExport(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	userID, _ := r.Context().Value(authpkg.ContextKeyUserID).(string)

	var req createExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	if len(req.CNPJs) == 0 {
		httputil.Error(w, http.StatusBadRequest, "cnpjs não pode ser vazio")
		return
	}
	if len(req.CNPJs) > 500 {
		httputil.Error(w, http.StatusBadRequest, "máximo de 500 CNPJs por exportação")
		return
	}

	for i, cnpj := range req.CNPJs {
		req.CNPJs[i] = brazil.NormalizeCNPJ(cnpj)
	}

	job, err := h.svc.CreateExport(r.Context(), orgID, userID, req.SearchID, req.CNPJs)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoCRMIntegration):
			httputil.Error(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientCredits):
			httputil.Error(w, http.StatusPaymentRequired, "créditos insuficientes")
		default:
			slog.ErrorContext(r.Context(), "create export", "error", err)
			httputil.Error(w, http.StatusInternalServerError, "erro interno")
		}
		return
	}

	if err := h.queue.RPush(r.Context(), "queue:exports", job.ID).Err(); err != nil {
		slog.ErrorContext(r.Context(), "enqueue export", "export_id", job.ID, "error", err)
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"export_id": job.ID,
		"status":    job.Status,
		"total":     job.TotalCount,
	})
}

// GET /exports/:id
func (h *Handler) GetExport(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	id := chi.URLParam(r, "id")

	job, err := h.svc.GetExport(r.Context(), id, orgID)
	if errors.Is(err, domain.ErrNotFound) {
		httputil.Error(w, http.StatusNotFound, "export não encontrado")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "get export", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, job)
}

// GET /exports
func (h *Handler) ListExports(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	resp, err := h.svc.ListExports(r.Context(), orgID, page, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "list exports", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func parseIntParam(r *http.Request, key string, def int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v < 1 {
		return def
	}
	return v
}
