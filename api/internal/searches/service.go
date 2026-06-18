package searches

import (
	"context"
	"errors"
	"fmt"

	"github.com/vantaggio/prospect-api/internal/domain"
)

var ErrInvalidSearchInput = errors.New("forneça filtros (busca estruturada) ou texto (busca semântica)")

type ServiceInterface interface {
	Create(ctx context.Context, orgID, userID string, mode domain.SearchMode, filters domain.SearchFilters, query *string) (*domain.Search, error)
	Get(ctx context.Context, id, orgID string, page, limit int) (*domain.SearchResponse, error)
	List(ctx context.Context, orgID string, page, limit int) (*domain.SearchListResponse, error)
	SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, orgID, userID string, mode domain.SearchMode, filters domain.SearchFilters, query *string) (*domain.Search, error) {
	if mode == domain.SearchModeSemantic && (query == nil || *query == "") {
		return nil, ErrInvalidSearchInput
	}
	if mode == domain.SearchModeStructured {
		empty := len(filters.CNAEs) == 0 &&
			(filters.UF == nil || *filters.UF == "") &&
			(filters.City == nil || *filters.City == "") &&
			filters.CapitalMin == nil &&
			filters.Status == nil
		if empty {
			return nil, ErrInvalidSearchInput
		}
	}

	search := &domain.Search{
		OrgID:     orgID,
		UserID:    userID,
		Mode:      mode,
		Filters:   filters,
		QueryText: query,
	}
	if err := s.repo.Create(ctx, search); err != nil {
		return nil, fmt.Errorf("create search: %w", err)
	}
	return search, nil
}

func (s *Service) Get(ctx context.Context, id, orgID string, page, limit int) (*domain.SearchResponse, error) {
	search, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get search: %w", err)
	}

	resp := &domain.SearchResponse{
		ID:          search.ID,
		Mode:        search.Mode,
		Status:      search.Status,
		ResultCount: search.ResultCount,
	}

	if search.Status == domain.SearchStatusDone {
		results, total, err := s.repo.GetResults(ctx, id, page, limit)
		if err != nil {
			return nil, fmt.Errorf("get results: %w", err)
		}
		resp.Results = results
		resp.Page = page
		resp.Limit = limit
		resp.Total = &total
	}
	return resp, nil
}

func (s *Service) List(ctx context.Context, orgID string, page, limit int) (*domain.SearchListResponse, error) {
	searches, total, err := s.repo.List(ctx, orgID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list searches: %w", err)
	}
	return &domain.SearchListResponse{Data: searches, Total: total}, nil
}

func (s *Service) SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error) {
	if q == "" {
		return []domain.CNAE{}, nil
	}
	cnaes, err := s.repo.SearchCNAEs(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("search cnaes: %w", err)
	}
	return cnaes, nil
}
