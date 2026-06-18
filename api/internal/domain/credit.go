package domain

type CreditTxType string

const (
	CreditTxPurchase      CreditTxType = "purchase"
	CreditTxSearch        CreditTxType = "search"
	CreditTxCompanyDetail CreditTxType = "company_detail"
	CreditTxEnrichment    CreditTxType = "enrichment"
	CreditTxExport        CreditTxType = "export"
	CreditTxAdjustment    CreditTxType = "adjustment"
)

type CreditTransaction struct {
	ID          string       `json:"id"`
	OrgID       string       `json:"org_id"`
	UserID      *string      `json:"user_id,omitempty"`
	Type        CreditTxType `json:"type"`
	Amount      int          `json:"amount"`
	Description *string      `json:"description,omitempty"`
	ReferenceID *string      `json:"reference_id,omitempty"`
	CreatedAt   string       `json:"created_at"`
}

type CreditBalanceResponse struct {
	Balance int    `json:"balance"`
	OrgID   string `json:"org_id"`
}

type CreditTransactionsResponse struct {
	Data    []CreditTransaction `json:"data"`
	Total   int                 `json:"total"`
	Balance int                 `json:"balance"`
}
