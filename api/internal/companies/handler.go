package companies

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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
