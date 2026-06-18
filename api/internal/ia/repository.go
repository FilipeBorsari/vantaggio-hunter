package ia

import (
	"context"
	"time"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	// Save persists a new qualification record.
	Save(ctx context.Context, q *domain.AIQualification) error
	// FindRecent returns the most recent qualification for cnpj+org within the last maxAge duration.
	// Returns nil, nil when none exists.
	FindRecent(ctx context.Context, cnpj, orgID string, maxAge time.Duration) (*domain.AIQualification, error)
	// List returns qualifications for orgID, optionally filtered by CNPJ.
	List(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error)
	// GetCompanyPromptData retrieves minimal company fields needed to build the qualification prompt.
	GetCompanyPromptData(ctx context.Context, cnpj string) (*CompanyPromptData, error)
}

// CompanyPromptData holds company fields relevant to the AI qualification prompt.
type CompanyPromptData struct {
	CNPJ              string
	RazaoSocial       string
	Municipio         *string
	UF                string
	CapitalSocial     *float64
	SituacaoCadastral int
	DataInicio        *string
	Porte             *int
	OpcaoSimples      *bool
	PrimaryCNAE       *string
}
