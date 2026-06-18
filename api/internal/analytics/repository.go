package analytics

import (
	"context"
	"time"

	"github.com/vantaggio/prospect-api/internal/domain"
)

type Repository interface {
	GetKPIs(ctx context.Context, orgID string, from, to time.Time) (*domain.AnalyticsKPIs, error)
	GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error)
	GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error)
	GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error)
	RunETL(ctx context.Context) error
}
