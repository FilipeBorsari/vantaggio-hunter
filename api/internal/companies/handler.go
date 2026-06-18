package companies

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	authpkg "github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/httputil"
)

type Handler struct {
	svc       ServiceInterface
	creditSvc credits.ServiceInterface
}

func NewHandler(svc ServiceInterface, creditSvc credits.ServiceInterface) *Handler {
	return &Handler{svc: svc, creditSvc: creditSvc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := Filters{
		UF:    q.Get("uf"),
		City:  q.Get("city"),
		Page:  intParam(q.Get("page"), 1),
		Limit: intParam(q.Get("limit"), 50),
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if cnae := q.Get("cnae"); cnae != "" {
		for _, c := range strings.Split(cnae, ",") {
			if c = strings.TrimSpace(c); c != "" {
				f.CNAEs = append(f.CNAEs, c)
			}
		}
	}
	if v := q.Get("capital_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.CapitalMin = &n
		}
	}
	if v := q.Get("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Status = &n
		}
	}

	result, err := h.svc.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "erro ao consultar empresas")
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetByCNPJ(w http.ResponseWriter, r *http.Request) {
	cnpj := chi.URLParam(r, "cnpj")
	orgID, _ := r.Context().Value(authpkg.ContextKeyOrgID).(string)
	userID, _ := r.Context().Value(authpkg.ContextKeyUserID).(string)

	if h.creditSvc != nil && orgID != "" {
		tx, err := h.creditSvc.BeginTx(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "begin credit tx for company detail", "error", err)
			httputil.Error(w, http.StatusInternalServerError, "erro interno")
			return
		}

		deductErr := h.creditSvc.Deduct(r.Context(), tx, orgID, userID,
			10,
			domain.CreditTxCompanyDetail,
			nil,
			fmt.Sprintf("Consulta CNPJ: %s", cnpj),
		)
		if deductErr != nil {
			_ = tx.Rollback(r.Context())
			if errors.Is(deductErr, domain.ErrInsufficientCredits) {
				httputil.Error(w, http.StatusPaymentRequired, "créditos insuficientes")
				return
			}
			slog.ErrorContext(r.Context(), "deduct credits for company detail", "cnpj", cnpj, "error", deductErr)
			httputil.Error(w, http.StatusInternalServerError, "erro interno")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "commit credit tx for company detail", "error", err)
			httputil.Error(w, http.StatusInternalServerError, "erro interno")
			return
		}
	}

	company, err := h.svc.GetByCNPJ(r.Context(), cnpj)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.Error(w, http.StatusNotFound, "empresa não encontrada")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "erro ao consultar empresa")
		return
	}
	httputil.JSON(w, http.StatusOK, company)
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
