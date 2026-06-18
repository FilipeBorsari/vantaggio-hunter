package ia

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Qualify(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	userID, _ := r.Context().Value(authpkg.ContextKeyUserID).(string)
	cnpj := chi.URLParam(r, "cnpj")

	if len(cnpj) != 14 {
		httputil.Error(w, http.StatusBadRequest, "CNPJ deve ter 14 dígitos")
		return
	}

	result, err := h.svc.Qualify(r.Context(), orgID, userID, cnpj)
	if err != nil {
		switch {
		case errors.Is(err, ErrInsufficientCredits):
			httputil.Error(w, http.StatusPaymentRequired, "créditos insuficientes")
		case errors.Is(err, ErrCompanyNotFound):
			httputil.Error(w, http.StatusNotFound, "empresa não encontrada")
		default:
			slog.ErrorContext(r.Context(), "qualify", "cnpj", cnpj, "error", err)
			httputil.Error(w, http.StatusInternalServerError, "erro interno")
		}
		return
	}

	status := http.StatusCreated
	if result.FromCache {
		status = http.StatusOK
	}
	httputil.JSON(w, status, result)
}

func (h *Handler) ListQualifications(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)

	var cnpj *string
	if q := r.URL.Query().Get("cnpj"); q != "" {
		cnpj = &q
	}

	results, err := h.svc.ListQualifications(r.Context(), orgID, cnpj)
	if err != nil {
		slog.ErrorContext(r.Context(), "list qualifications", "error", err)
		httputil.Error(w, http.StatusInternalServerError, "erro interno")
		return
	}
	httputil.JSON(w, http.StatusOK, results)
}
