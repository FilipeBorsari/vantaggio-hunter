package companies

import (
	"context"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	Count(ctx context.Context, f Filters) (int, error)
	List(ctx context.Context, f Filters) ([]domain.Company, error)
	AttachCNAEs(ctx context.Context, companies []domain.Company) error
	GetByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyDetail, error)
	GetCNAEsByCNPJ(ctx context.Context, cnpj string) ([]domain.CNAE, error)
	GetPartnersByCNPJBasico(ctx context.Context, cnpjBasico string) ([]domain.Partner, error)
}
