package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

// orgIDFromCtx extracts the org ID set by the auth middleware.
// Returns an empty string and false if the middleware did not run.
func orgIDFromCtx(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	return id, ok && id != ""
}

func (h *Handler) GetKPIs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromCtx(r)
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	label, from, to, err := ParsePeriod(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	kpis, err := h.svc.GetKPIs(r.Context(), orgID, label, from, to)
	if err != nil {
		slog.Error("get kpis", "error", err, "org_id", orgID)
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar KPIs")
		return
	}
	httputil.JSON(w, http.StatusOK, kpis)
}

func (h *Handler) GetDailyConsumption(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromCtx(r)
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	_, from, to, err := ParsePeriod(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	points, err := h.svc.GetDailyConsumption(r.Context(), orgID, from, to)
	if err != nil {
		slog.Error("get daily consumption", "error", err, "org_id", orgID)
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar consumo diário")
		return
	}
	httputil.JSON(w, http.StatusOK, points)
}

func (h *Handler) GetTopCNAEs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromCtx(r)
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	_, from, to, err := ParsePeriod(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.svc.GetTopCNAEs(r.Context(), orgID, from, to, limit)
	if err != nil {
		slog.Error("get top cnaes", "error", err, "org_id", orgID)
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar CNAEs")
		return
	}
	httputil.JSON(w, http.StatusOK, results)
}

func (h *Handler) GetFunnel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromCtx(r)
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	_, from, to, err := ParsePeriod(r)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	funnel, err := h.svc.GetFunnel(r.Context(), orgID, from, to)
	if err != nil {
		slog.Error("get funnel", "error", err, "org_id", orgID)
		httputil.Error(w, http.StatusInternalServerError, "erro ao buscar funil")
		return
	}
	httputil.JSON(w, http.StatusOK, funnel)
}

// TriggerETL is an internal endpoint for manual ETL runs, restricted to admin role.
func (h *Handler) TriggerETL(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	if err := h.svc.RunETL(ctx); err != nil {
		slog.Error("manual etl run failed", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "ETL falhou")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ParsePeriod converts query params into a UTC from/to range.
// Supported period values: 7d, 30d (default), 90d, custom.
// For custom: requires ?from=YYYY-MM-DD&to=YYYY-MM-DD with from <= to.
func ParsePeriod(r *http.Request) (label string, from, to time.Time, err error) {
	period := r.URL.Query().Get("period")
	now := time.Now().UTC()
	switch period {
	case "7d":
		return "7d", now.AddDate(0, 0, -7), now, nil
	case "90d":
		return "90d", now.AddDate(0, 0, -90), now, nil
	case "custom":
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return "", time.Time{}, time.Time{}, fmt.Errorf("data inicial inválida: %w", err)
		}
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return "", time.Time{}, time.Time{}, fmt.Errorf("data final inválida: %w", err)
		}
		if to.Before(from) {
			return "", time.Time{}, time.Time{}, fmt.Errorf("data inicial deve ser anterior à data final")
		}
		return "custom", from, to, nil
	default: // "30d" e valores desconhecidos
		return "30d", now.AddDate(0, 0, -30), now, nil
	}
}
