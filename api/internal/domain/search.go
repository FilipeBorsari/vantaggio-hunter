package domain

type SearchMode string
type SearchStatus string

const (
	SearchModeStructured SearchMode   = "structured"
	SearchModeSemantic   SearchMode   = "semantic"
	SearchStatusQueued   SearchStatus = "queued"
	SearchStatusProc     SearchStatus = "processing"
	SearchStatusDone     SearchStatus = "done"
	SearchStatusFailed   SearchStatus = "failed"
)

type SearchFilters struct {
	CNAEs      []string `json:"cnaes,omitempty"`
	UF         *string  `json:"uf,omitempty"`
	City       *string  `json:"city,omitempty"`
	CapitalMin *float64 `json:"capital_min,omitempty"`
	CapitalMax *float64 `json:"capital_max,omitempty"`
	Status     *int     `json:"status,omitempty"`
	Porte      *int     `json:"porte,omitempty"`
}

type Search struct {
	ID          string        `json:"id"`
	OrgID       string        `json:"org_id"`
	UserID      string        `json:"user_id"`
	Mode        SearchMode    `json:"mode"`
	Filters     SearchFilters `json:"filters"`
	QueryText   *string       `json:"query_text,omitempty"`
	Status      SearchStatus  `json:"status"`
	ResultCount *int          `json:"result_count,omitempty"`
	ErrorMsg    *string       `json:"error_msg,omitempty"`
	CreatedAt   string        `json:"created_at"`
	DoneAt      *string       `json:"done_at,omitempty"`
}

type SearchResult struct {
	CNPJ              string   `json:"cnpj"`
	RazaoSocial       string   `json:"razao_social"`
	Municipio         *string  `json:"municipio,omitempty"`
	UF                string   `json:"uf"`
	CapitalSocial     *float64 `json:"capital_social,omitempty"`
	SituacaoCadastral int      `json:"situacao"`
	Score             *float64 `json:"score,omitempty"`
	CNAEs             []CNAE   `json:"cnaes"`
}

type SearchResponse struct {
	ID          string         `json:"id"`
	Status      SearchStatus   `json:"status"`
	ResultCount *int           `json:"result_count,omitempty"`
	Results     []SearchResult `json:"results,omitempty"`
	Page        int            `json:"page,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	Total       *int           `json:"total,omitempty"`
}

type SearchListResponse struct {
	Data  []Search `json:"data"`
	Total int      `json:"total"`
}
