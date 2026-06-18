package exports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) SaveIntegration(ctx context.Context, orgID, crmType, baseURL, encAPIKey string, inboxID *int, accountID int) (*domain.CRMIntegration, error) {
	const q = `
		INSERT INTO tb_crm_integrations (org_id, crm_type, base_url, api_key, inbox_id, extra_config)
		VALUES ($1, $2, $3, $4, $5, jsonb_build_object('account_id', $6::int))
		ON CONFLICT (org_id) DO UPDATE
		  SET crm_type=$2, base_url=$3, api_key=$4, inbox_id=$5,
		      extra_config=jsonb_build_object('account_id', $6::int),
		      updated_at=now()
		RETURNING id, org_id, crm_type, base_url, inbox_id,
		          (extra_config->>'account_id')::int, is_active, created_at::text`

	row := r.db.QueryRow(ctx, q, orgID, crmType, baseURL, encAPIKey, inboxID, accountID)
	return scanIntegration(row)
}

func (r *postgresRepo) GetIntegration(ctx context.Context, orgID string) (*domain.CRMIntegration, error) {
	const q = `
		SELECT id, org_id, crm_type, base_url, inbox_id,
		       COALESCE((extra_config->>'account_id')::int, 1),
		       is_active, created_at::text
		FROM tb_crm_integrations
		WHERE org_id = $1`

	row := r.db.QueryRow(ctx, q, orgID)
	intg, err := scanIntegration(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return intg, err
}

func (r *postgresRepo) GetIntegrationRaw(ctx context.Context, orgID string) (baseURL, encAPIKey string, inboxID *int, accountID int, err error) {
	const q = `
		SELECT base_url, api_key, inbox_id,
		       COALESCE((extra_config->>'account_id')::int, 1)
		FROM tb_crm_integrations
		WHERE org_id = $1 AND is_active = true`

	err = r.db.QueryRow(ctx, q, orgID).Scan(&baseURL, &encAPIKey, &inboxID, &accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, 0, domain.ErrNotFound
	}
	if err != nil {
		return "", "", nil, 0, fmt.Errorf("get integration raw: %w", err)
	}
	return baseURL, encAPIKey, inboxID, accountID, nil
}

func (r *postgresRepo) CreateExport(ctx context.Context, orgID, userID string, searchID *string, cnpjs []string) (*domain.ExportJob, error) {
	const q = `
		INSERT INTO tb_export_queue (org_id, user_id, search_id, cnpjs, total_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, org_id, user_id, search_id, cnpjs, crm_type, status,
		          total_count, success_count, fail_count, error_log,
		          attempt, next_retry_at, created_at::text, done_at::text`

	row := r.db.QueryRow(ctx, q, orgID, userID, searchID, cnpjs, len(cnpjs))
	return scanExport(row)
}

func (r *postgresRepo) GetExport(ctx context.Context, id, orgID string) (*domain.ExportJob, error) {
	const q = `
		SELECT id, org_id, user_id, search_id, cnpjs, crm_type, status,
		       total_count, success_count, fail_count, error_log,
		       attempt, next_retry_at, created_at::text, done_at::text
		FROM tb_export_queue
		WHERE id = $1 AND org_id = $2`

	row := r.db.QueryRow(ctx, q, id, orgID)
	job, err := scanExport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return job, err
}

func (r *postgresRepo) ListExports(ctx context.Context, orgID string, page, limit int) ([]domain.ExportJob, int, error) {
	const countQ = `SELECT COUNT(*) FROM tb_export_queue WHERE org_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count exports: %w", err)
	}

	const q = `
		SELECT id, org_id, user_id, search_id, cnpjs, crm_type, status,
		       total_count, success_count, fail_count, error_log,
		       attempt, next_retry_at, created_at::text, done_at::text
		FROM tb_export_queue
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, orgID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list exports: %w", err)
	}
	defer rows.Close()

	var jobs []domain.ExportJob
	for rows.Next() {
		job, err := scanExport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan export: %w", err)
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return jobs, total, nil
}

func (r *postgresRepo) MarkProcessing(ctx context.Context, id string) (int, error) {
	const q = `
		UPDATE tb_export_queue
		SET status='processing', attempt=attempt+1
		WHERE id = $1
		RETURNING attempt`

	var attempt int
	err := r.db.QueryRow(ctx, q, id).Scan(&attempt)
	if err != nil {
		return 0, fmt.Errorf("mark processing: %w", err)
	}
	return attempt, nil
}

func (r *postgresRepo) UpdateExportResult(ctx context.Context, id string, status domain.ExportStatus, successCount, failCount int, errorLog []domain.ExportErrorEntry, nextRetryAt *time.Time) error {
	errJSON, err := json.Marshal(errorLog)
	if err != nil {
		return fmt.Errorf("marshal error log: %w", err)
	}

	doneAt := (*time.Time)(nil)
	if status == domain.ExportStatusDone || status == domain.ExportStatusPartial || status == domain.ExportStatusFailed {
		now := time.Now()
		if nextRetryAt == nil {
			doneAt = &now
		}
	}

	const q = `
		UPDATE tb_export_queue
		SET status=$2, success_count=$3, fail_count=$4,
		    error_log=$5::jsonb, next_retry_at=$6, done_at=$7
		WHERE id = $1`

	if _, err := r.db.Exec(ctx, q, id, string(status), successCount, failCount, string(errJSON), nextRetryAt, doneAt); err != nil {
		return fmt.Errorf("update export result: %w", err)
	}
	return nil
}

func (r *postgresRepo) IncrFunnelExported(ctx context.Context, searchID, orgID string, count int) error {
	const q = `
		UPDATE fato_funil_leads
		SET leads_exportados = leads_exportados + $3
		WHERE search_id = $1 AND org_id = $2`

	if _, err := r.db.Exec(ctx, q, searchID, orgID, count); err != nil {
		return fmt.Errorf("incr funnel exported: %w", err)
	}
	return nil
}

// scanIntegration scans one row into a CRMIntegration.
func scanIntegration(row interface {
	Scan(...any) error
}) (*domain.CRMIntegration, error) {
	var intg domain.CRMIntegration
	if err := row.Scan(
		&intg.ID, &intg.OrgID, &intg.CRMType, &intg.BaseURL,
		&intg.InboxID, &intg.AccountID, &intg.IsActive, &intg.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &intg, nil
}

// scanExport scans one row into an ExportJob.
func scanExport(row interface {
	Scan(...any) error
}) (*domain.ExportJob, error) {
	var job domain.ExportJob
	var errLogRaw []byte
	var doneAt *string
	if err := row.Scan(
		&job.ID, &job.OrgID, &job.UserID, &job.SearchID,
		&job.CNPJs, &job.CRMType, &job.Status,
		&job.TotalCount, &job.SuccessCount, &job.FailCount, &errLogRaw,
		&job.Attempt, &job.NextRetryAt, &job.CreatedAt, &doneAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(errLogRaw, &job.ErrorLog); err != nil {
		job.ErrorLog = []domain.ExportErrorEntry{}
	}
	if doneAt != nil {
		job.DoneAt = doneAt
	}
	return &job, nil
}
