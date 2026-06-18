package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type ServiceInterface interface {
	GetKPIs(ctx context.Context, orgID, period string, from, to time.Time) (*domain.AnalyticsKPIs, error)
	GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error)
	GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error)
	GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error)
	RunETL(ctx context.Context) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetKPIs(ctx context.Context, orgID, period string, from, to time.Time) (*domain.AnalyticsKPIs, error) {
	kpis, err := s.repo.GetKPIs(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get kpis: %w", err)
	}
	kpis.Period = period
	if kpis.LeadsExtracted > 0 {
		kpis.ConversionRate = float64(kpis.LeadsExported) / float64(kpis.LeadsExtracted)
	}
	return kpis, nil
}

func (s *Service) GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error) {
	points, err := s.repo.GetDailyConsumption(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get daily consumption: %w", err)
	}
	if points == nil {
		points = []domain.DailyPoint{}
	}
	return points, nil
}

func (s *Service) GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}
	results, err := s.repo.GetTopCNAEs(ctx, orgID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("get top cnaes: %w", err)
	}
	if results == nil {
		results = []domain.TopCNAE{}
	}
	return results, nil
}

func (s *Service) GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error) {
	funnel, err := s.repo.GetFunnel(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get funnel: %w", err)
	}
	return funnel, nil
}

func (s *Service) RunETL(ctx context.Context) error {
	return s.repo.RunETL(ctx)
}
