package companies

import (
	"context"
	"fmt"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Filters struct {
	CNAEs      []string
	UF         string
	City       string
	CapitalMin *float64
	Status     *int
	Page       int
	Limit      int
}

type ServiceInterface interface {
	List(ctx context.Context, f Filters) (*domain.CompanyListResponse, error)
	GetByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyDetail, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, f Filters) (*domain.CompanyListResponse, error) {
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}

	list, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}

	if err := s.repo.AttachCNAEs(ctx, list); err != nil {
		return nil, fmt.Errorf("attach cnaes: %w", err)
	}

	return &domain.CompanyListResponse{Data: list, Total: total, Page: f.Page, Limit: f.Limit}, nil
}

func (s *Service) GetByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyDetail, error) {
	detail, err := s.repo.GetByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, fmt.Errorf("get by cnpj: %w", err)
	}

	cnaes, err := s.repo.GetCNAEsByCNPJ(ctx, cnpj)
	if err != nil {
		return nil, fmt.Errorf("get cnaes: %w", err)
	}
	detail.CNAEs = cnaes

	basico := ""
	if len(cnpj) >= 8 {
		basico = cnpj[:8]
	}
	partners, err := s.repo.GetPartnersByCNPJBasico(ctx, basico)
	if err != nil {
		return nil, fmt.Errorf("get partners: %w", err)
	}
	detail.Partners = partners

	return detail, nil
}
