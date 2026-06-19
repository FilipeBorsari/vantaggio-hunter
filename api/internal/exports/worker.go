package exports

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vantaggio/prospect-api/internal/companies"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/crypto"
)

const (
	exportQueueKey        = "queue:exports"
	exportDelayedKey      = "queue:exports:delayed"
	exportBlpopTimeout    = 5 * time.Second
	exportChunkSize       = 50
	maxAttempts           = 3
)

var backoffDurations = []time.Duration{
	2 * time.Minute,
	10 * time.Minute,
}

// CompanyFetcher fetches detailed company info for export.
type CompanyFetcher interface {
	GetByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyDetail, error)
}

type Worker struct {
	repo       Repository
	queue      *redis.Client
	creditSvc  credits.ServiceInterface
	companies  CompanyFetcher
	encKey     []byte
	httpClient *http.Client
}

func NewWorker(repo Repository, queue *redis.Client, creditSvc credits.ServiceInterface, companiesRepo companies.Repository, encKey []byte) *Worker {
	return &Worker{
		repo:       repo,
		queue:      queue,
		creditSvc:  creditSvc,
		companies:  companiesRepo,
		encKey:     encKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Run processes exports from Redis until ctx is cancelled.
// It also runs a separate goroutine to promote delayed retries.
func (w *Worker) Run(ctx context.Context) {
	go w.runDelayedPromoter(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := w.queue.BLPop(ctx, exportBlpopTimeout, exportQueueKey).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if len(res) < 2 {
			continue
		}
		exportID := res[1]
		if err := w.process(ctx, exportID); err != nil {
			slog.ErrorContext(ctx, "export worker: process failed", "export_id", exportID, "error", err)
		}
	}
}

// runDelayedPromoter checks queue:exports:delayed every 30s and moves ready items to queue:exports.
func (w *Worker) runDelayedPromoter(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.promoteDelayed(ctx)
		}
	}
}

func (w *Worker) promoteDelayed(ctx context.Context) {
	now := float64(time.Now().Unix())
	ids, err := w.queue.ZRangeByScore(ctx, exportDelayedKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil || len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if err := w.queue.RPush(ctx, exportQueueKey, id).Err(); err != nil {
			slog.Error("promote delayed export", "id", id, "error", err)
			continue
		}
		if err := w.queue.ZRem(ctx, exportDelayedKey, id).Err(); err != nil {
			slog.Error("remove promoted export from delayed set", "id", id, "error", err)
		}
	}
}

func (w *Worker) process(ctx context.Context, exportID string) error {
	// Use a background context so we don't cancel mid-export on request timeout.
	procCtx := context.Background()

	// We need org_id to fetch the integration, but GetExport requires org_id.
	// Use a version without org scoping for the worker.
	job, err := w.getExportByID(procCtx, exportID)
	if err != nil {
		return fmt.Errorf("get export: %w", err)
	}

	attempt, err := w.repo.MarkProcessing(procCtx, exportID)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	baseURL, encAPIKey, inboxID, accountID, err := w.repo.GetIntegrationRaw(procCtx, job.OrgID)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}

	rawKey, err := crypto.Decrypt(encAPIKey, w.encKey)
	if err != nil {
		return fmt.Errorf("decrypt api key: %w", err)
	}

	chatwoot := NewChatwootClient(baseURL, string(rawKey), accountID, inboxID, w.httpClient)

	var (
		successCount int
		failCount    int
		errorLog     []domain.ExportErrorEntry
	)

	searchID := ""
	if job.SearchID != nil {
		searchID = *job.SearchID
	}

	// Process in chunks of exportChunkSize to avoid blocking the worker too long.
	for i := 0; i < len(job.CNPJs); i += exportChunkSize {
		if procCtx.Err() != nil {
			break
		}
		end := int(math.Min(float64(i+exportChunkSize), float64(len(job.CNPJs))))
		chunk := job.CNPJs[i:end]

		for _, cnpj := range chunk {
			company, err := w.companies.GetByCNPJ(procCtx, cnpj)
			if err != nil {
				failCount++
				errorLog = append(errorLog, domain.ExportErrorEntry{
					CNPJ:    cnpj,
					Error:   fmt.Sprintf("buscar empresa: %v", err),
					Attempt: attempt,
				})
				continue
			}

			if err := chatwoot.ExportCompany(procCtx, company, searchID); err != nil {
				failCount++
				errorLog = append(errorLog, domain.ExportErrorEntry{
					CNPJ:    cnpj,
					Error:   err.Error(),
					Attempt: attempt,
				})
				continue
			}

			// Debit 1 credit per successfully exported lead.
			tx, err := w.creditSvc.BeginTx(procCtx)
			if err != nil {
				slog.Error("begin credit tx", "cnpj", cnpj, "export_id", exportID, "error", err)
				failCount++
				errorLog = append(errorLog, domain.ExportErrorEntry{
					CNPJ:    cnpj,
					Error:   fmt.Sprintf("begin credit tx: %v", err),
					Attempt: attempt,
				})
				continue
			}
			ref := exportID
			deductErr := w.creditSvc.Deduct(procCtx, tx, job.OrgID, job.UserID, 1,
				domain.CreditTxExport, &ref, "Export lead "+cnpj)
			if deductErr != nil {
				if rbErr := tx.Rollback(procCtx); rbErr != nil {
					slog.Error("rollback credit tx", "cnpj", cnpj, "export_id", exportID, "error", rbErr)
				}
				slog.Error("deduct credit after export", "cnpj", cnpj, "export_id", exportID, "error", deductErr)
				failCount++
				errorLog = append(errorLog, domain.ExportErrorEntry{
					CNPJ:    cnpj,
					Error:   fmt.Sprintf("deduct credit: %v", deductErr),
					Attempt: attempt,
				})
				continue
			}
			if err := tx.Commit(procCtx); err != nil {
				slog.Error("commit credit tx", "cnpj", cnpj, "export_id", exportID, "error", err)
				failCount++
				errorLog = append(errorLog, domain.ExportErrorEntry{
					CNPJ:    cnpj,
					Error:   fmt.Sprintf("commit credit tx: %v", err),
					Attempt: attempt,
				})
				continue
			}
			successCount++
		}
	}

	// Determine final status and schedule retry if needed.
	var (
		finalStatus domain.ExportStatus
		nextRetryAt *time.Time
	)

	switch {
	case failCount > 0 && attempt < maxAttempts:
		finalStatus = domain.ExportStatusFailed
		delay := backoffDurations[min(attempt-1, len(backoffDurations)-1)]
		retryAt := time.Now().Add(delay)
		nextRetryAt = &retryAt
		score := float64(retryAt.Unix())
		if err := w.queue.ZAdd(procCtx, exportDelayedKey, redis.Z{
			Score:  score,
			Member: exportID,
		}).Err(); err != nil {
			slog.Error("schedule delayed retry", "export_id", exportID, "error", err)
		}
	case failCount == 0:
		finalStatus = domain.ExportStatusDone
	default:
		// Some successes and some failures after exhausting retries.
		if successCount > 0 {
			finalStatus = domain.ExportStatusPartial
		} else {
			finalStatus = domain.ExportStatusFailed
		}
	}

	if err := w.repo.UpdateExportResult(procCtx, exportID, finalStatus, successCount, failCount, errorLog, nextRetryAt); err != nil {
		slog.Error("update export result", "export_id", exportID, "error", err)
	}

	if successCount > 0 && searchID != "" {
		if err := w.repo.IncrFunnelExported(procCtx, searchID, job.OrgID, successCount); err != nil {
			slog.Error("incr funnel exported", "export_id", exportID, "error", err)
		}
	}

	slog.Info("export processed",
		"export_id", exportID,
		"status", finalStatus,
		"success", successCount,
		"fail", failCount,
	)
	return nil
}

// getExportByID fetches an export job without org scoping (worker-internal use).
func (w *Worker) getExportByID(ctx context.Context, id string) (*domain.ExportJob, error) {
	// The Repository.GetExport requires org_id; for the worker we use a raw query via the repo.
	// We work around this by using ListExports is not suitable either.
	// Instead, define a worker-only method that bypasses the org filter.
	// We cast to the concrete type — this is an intentional internal dependency kept isolated to main.
	pr, ok := w.repo.(*postgresRepo)
	if !ok {
		return nil, fmt.Errorf("repo type assertion failed")
	}
	const q = `
		SELECT id, org_id, user_id, search_id, cnpjs, crm_type, status,
		       total_count, success_count, fail_count, error_log,
		       attempt, next_retry_at, created_at::text, done_at::text
		FROM tb_export_queue WHERE id = $1`

	row := pr.db.QueryRow(ctx, q, id)
	return scanExport(row)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
