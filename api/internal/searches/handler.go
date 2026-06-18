package searches

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/httputil"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	svc   ServiceInterface
	queue *redis.Client
}

func NewHandler(svc ServiceInterface, queue *redis.Client) *Handler {
	return &Handler{svc: svc, queue: queue}
}

type createSearchRequest struct {
	Mode    domain.SearchMode    `json:"mode"`
	Filters domain.SearchFilters `json:"filters"`
	Query   *string              `json:"query"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	userID, _ := r.Context().Value(authpkg.ContextKeyUserID).(string)

	var req createSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	if req.Mode != domain.SearchModeStructured && req.Mode != domain.SearchModeSemantic {
		httputil.Error(w, http.StatusBadRequest, `mode deve ser "structured" ou "semantic"`)
		return
	}

	search, err := h.svc.Create(r.Context(), orgID, userID, req.Mode, req.Filters, req.Query)
	if err != nil {
		if errors.Is(err, ErrInvalidSearchInput) {
			httputil.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.ErrorContext(r.Context(), "create search", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}

	if err := h.queue.RPush(r.Context(), "queue:searches", search.ID).Err(); err != nil {
		slog.ErrorContext(r.Context(), "enqueue search", "search_id", search.ID, "error", err)
	}

	httputil.JSON(w, http.StatusCreated, map[string]string{
		"search_id": search.ID,
		"status":    string(search.Status),
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	id := chi.URLParam(r, "id")

	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", 100)
	if limit > 100 {
		limit = 100
	}

	resp, err := h.svc.Get(r.Context(), id, orgID, page, limit)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "busca não encontrada")
			return
		}
		slog.ErrorContext(r.Context(), "get search", "id", id, "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	resp, err := h.svc.List(r.Context(), orgID, page, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "list searches", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SearchCNAEs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	cnaes, err := h.svc.SearchCNAEs(r.Context(), q)
	if err != nil {
		slog.ErrorContext(r.Context(), "search cnaes", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, cnaes)
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
