package exports

import (
	"context"
	"errors"
	"fmt"

	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/crypto"
)

type ServiceInterface interface {
	CreateIntegration(ctx context.Context, orgID, crmType, baseURL, apiKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error)
	GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error)
	CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error)
	GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error)
	ListExports(ctx context.Context, orgID string, page, limit int) (*domain.ExportListResponse, error)
}

// ErrNoCRMIntegration is returned when an export is attempted with no configured CRM.
var ErrNoCRMIntegration = errors.New("integração CRM não configurada")

type Service struct {
	repo      Repository
	creditSvc credits.ServiceInterface
	encKey    []byte
}

func NewService(repo Repository, creditSvc credits.ServiceInterface, encKey []byte) *Service {
	return &Service{repo: repo, creditSvc: creditSvc, encKey: encKey}
}

func (s *Service) CreateIntegration(ctx context.Context, orgID, crmType, baseURL, apiKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error) {
	encKey, err := crypto.Encrypt([]byte(apiKey), s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt api key: %w", err)
	}
	intg, err := s.repo.SaveIntegration(ctx, orgID, crmType, baseURL, encKey, inboxID, accountID)
	if err != nil {
		return nil, fmt.Errorf("save integration: %w", err)
	}
	return intg, nil
}

func (s *Service) GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error) {
	intg, err := s.repo.GetIntegration(ctx, orgID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get integration: %w", err)
	}
	return intg, nil
}

func (s *Service) CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error) {
	if len(cnpjs) == 0 {
		return nil, fmt.Errorf("lista de CNPJs vazia")
	}
	if len(cnpjs) > 500 {
		return nil, fmt.Errorf("máximo de 500 CNPJs por exportação")
	}

	// Ensure a CRM integration exists for this org.
	if _, err := s.repo.GetIntegration(ctx, orgID); errors.Is(err, domain.ErrNotFound) {
		return nil, ErrNoCRMIntegration
	} else if err != nil {
		return nil, fmt.Errorf("check integration: %w", err)
	}

	// Verify sufficient balance.
	balanceResp, err := s.creditSvc.GetBalance(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	if balanceResp.Balance < len(cnpjs) {
		return nil, domain.ErrInsufficientCredits
	}

	job, err := s.repo.CreateExport(ctx, orgID, userID, searchID, cnpjs)
	if err != nil {
		return nil, fmt.Errorf("create export: %w", err)
	}
	return job, nil
}

func (s *Service) GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error) {
	job, err := s.repo.GetExport(ctx, id, orgID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get export: %w", err)
	}
	return job, nil
}

func (s *Service) ListExports(ctx context.Context, orgID string, page, limit int) (*domain.ExportListResponse, error) {
	jobs, total, err := s.repo.ListExports(ctx, orgID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list exports: %w", err)
	}
	if jobs == nil {
		jobs = []domain.ExportJob{}
	}
	return &domain.ExportListResponse{Data: jobs, Total: total}, nil
}
