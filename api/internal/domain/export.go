package domain

import "time"

type CRMIntegration struct {
	ID        string `json:"id"`
	OrgID     string `json:"org_id"`
	CRMType   string `json:"crm_type"`
	BaseURL   string `json:"base_url"`
	InboxID   *int   `json:"inbox_id,omitempty"`
	AccountID int    `json:"account_id"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusDone       ExportStatus = "done"
	ExportStatusPartial    ExportStatus = "partial"
	ExportStatusFailed     ExportStatus = "failed"
)

type ExportErrorEntry struct {
	CNPJ    string `json:"cnpj"`
	Error   string `json:"error"`
	Attempt int    `json:"attempt"`
}

type ExportJob struct {
	ID           string             `json:"id"`
	OrgID        string             `json:"org_id"`
	UserID       string             `json:"user_id"`
	SearchID     *string            `json:"search_id,omitempty"`
	CNPJs        []string           `json:"cnpjs"`
	CRMType      string             `json:"crm_type"`
	Status       ExportStatus       `json:"status"`
	TotalCount   int                `json:"total_count"`
	SuccessCount int                `json:"success_count"`
	FailCount    int                `json:"fail_count"`
	ErrorLog     []ExportErrorEntry `json:"error_log"`
	Attempt      int                `json:"attempt"`
	NextRetryAt  *time.Time         `json:"next_retry_at,omitempty"`
	CreatedAt    string             `json:"created_at"`
	DoneAt       *string            `json:"done_at,omitempty"`
}

type ExportListResponse struct {
	Data  []ExportJob `json:"data"`
	Total int         `json:"total"`
}
