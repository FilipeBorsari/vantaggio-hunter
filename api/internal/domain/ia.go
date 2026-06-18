package domain

import "time"

type AIQualification struct {
	ID            string    `json:"id"`
	CNPJ          string    `json:"cnpj"`
	OrgID         string    `json:"org_id"`
	UserID        string    `json:"user_id"`
	Score         int16     `json:"score"`
	Justification string    `json:"justification"`
	PromptUsed    string    `json:"-"`
	ModelUsed     string    `json:"model"`
	TokensInput   int       `json:"tokens_input"`
	TokensOutput  int       `json:"tokens_output"`
	CreatedAt     time.Time `json:"created_at"`
}
