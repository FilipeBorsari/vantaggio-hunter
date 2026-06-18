package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepo{db: db}
}

func tempoID(t time.Time) int {
	y, m, d := t.Date()
	return y*10000 + int(m)*100 + d
}

func (r *postgresRepo) GetKPIs(ctx context.Context, orgID string, from, to time.Time) (*domain.AnalyticsKPIs, error) {
	fromID := tempoID(from)
	toID := tempoID(to)

	kpis := &domain.AnalyticsKPIs{}

	const qCredits = `
		SELECT
			COALESCE(SUM(CASE WHEN NOT eh_entrada THEN creditos ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eh_entrada     THEN creditos ELSE 0 END), 0)
		FROM fato_consumo_creditos
		WHERE org_id = $1 AND tempo_id BETWEEN $2 AND $3`

	if err := r.db.QueryRow(ctx, qCredits, orgID, fromID, toID).
		Scan(&kpis.CreditsConsumed, &kpis.CreditsPurchased); err != nil {
		return nil, fmt.Errorf("query credits kpis: %w", err)
	}

	const qLeads = `
		SELECT
			COALESCE(SUM(leads_extraidos), 0),
			COALESCE(SUM(leads_qualificados), 0),
			COALESCE(SUM(leads_exportados), 0),
			COUNT(*)
		FROM fato_funil_leads
		WHERE org_id = $1 AND tempo_id BETWEEN $2 AND $3`

	if err := r.db.QueryRow(ctx, qLeads, orgID, fromID, toID).
		Scan(&kpis.LeadsExtracted, &kpis.LeadsQualified, &kpis.LeadsExported, &kpis.SearchesCount); err != nil {
		return nil, fmt.Errorf("query leads kpis: %w", err)
	}

	return kpis, nil
}

func (r *postgresRepo) GetDailyConsumption(ctx context.Context, orgID string, from, to time.Time) ([]domain.DailyPoint, error) {
	fromID := tempoID(from)
	toID := tempoID(to)

	const q = `
		SELECT
			dt.data::TEXT,
			COALESCE(c.credits, 0),
			COALESCE(l.leads, 0)
		FROM dim_tempo dt
		LEFT JOIN (
			SELECT tempo_id, SUM(creditos) AS credits
			FROM fato_consumo_creditos
			WHERE org_id = $1 AND NOT eh_entrada
			GROUP BY tempo_id
		) c ON c.tempo_id = dt.id
		LEFT JOIN (
			SELECT tempo_id, SUM(leads_extraidos) AS leads
			FROM fato_funil_leads
			WHERE org_id = $1
			GROUP BY tempo_id
		) l ON l.tempo_id = dt.id
		WHERE dt.id BETWEEN $2 AND $3
		ORDER BY dt.id`

	rows, err := r.db.Query(ctx, q, orgID, fromID, toID)
	if err != nil {
		return nil, fmt.Errorf("query daily consumption: %w", err)
	}
	defer rows.Close()

	var points []domain.DailyPoint
	for rows.Next() {
		var p domain.DailyPoint
		if err := rows.Scan(&p.Date, &p.Credits, &p.Leads); err != nil {
			return nil, fmt.Errorf("scan daily point: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating daily consumption rows: %w", err)
	}
	return points, nil
}

func (r *postgresRepo) GetTopCNAEs(ctx context.Context, orgID string, from, to time.Time, limit int) ([]domain.TopCNAE, error) {
	fromID := tempoID(from)
	toID := tempoID(to)

	const q = `
		SELECT
			ffl.cnae_id,
			COALESCE(dc.description, ffl.cnae_id),
			SUM(ffl.leads_extraidos) AS leads
		FROM fato_funil_leads ffl
		LEFT JOIN dim_cnae dc ON dc.code = ffl.cnae_id
		WHERE ffl.org_id = $1
		  AND ffl.tempo_id BETWEEN $2 AND $3
		  AND ffl.cnae_id IS NOT NULL
		GROUP BY ffl.cnae_id, dc.description
		ORDER BY leads DESC
		LIMIT $4`

	rows, err := r.db.Query(ctx, q, orgID, fromID, toID, limit)
	if err != nil {
		return nil, fmt.Errorf("query top cnaes: %w", err)
	}
	defer rows.Close()

	var results []domain.TopCNAE
	for rows.Next() {
		var t domain.TopCNAE
		if err := rows.Scan(&t.CNAECode, &t.Description, &t.Leads); err != nil {
			return nil, fmt.Errorf("scan top cnae: %w", err)
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating top cnaes rows: %w", err)
	}
	return results, nil
}

func (r *postgresRepo) GetFunnel(ctx context.Context, orgID string, from, to time.Time) (*domain.FunnelResponse, error) {
	fromID := tempoID(from)
	toID := tempoID(to)

	const q = `
		SELECT
			COALESCE(SUM(leads_extraidos), 0),
			COALESCE(SUM(leads_qualificados), 0),
			COALESCE(SUM(leads_exportados), 0)
		FROM fato_funil_leads
		WHERE org_id = $1 AND tempo_id BETWEEN $2 AND $3`

	var extracted, qualified, exported int
	if err := r.db.QueryRow(ctx, q, orgID, fromID, toID).
		Scan(&extracted, &qualified, &exported); err != nil {
		return nil, fmt.Errorf("query funnel: %w", err)
	}

	return &domain.FunnelResponse{
		Stages: []domain.FunnelStage{
			{Name: "Extraídos", Count: extracted},
			{Name: "Qualificados", Count: qualified},
			{Name: "Exportados", Count: exported},
		},
	}, nil
}

func (r *postgresRepo) RunETL(ctx context.Context) error {
	if err := r.syncDimOrganizacao(ctx); err != nil {
		return fmt.Errorf("sync dim_organizacao: %w", err)
	}
	if err := r.syncDimUsuario(ctx); err != nil {
		return fmt.Errorf("sync dim_usuario: %w", err)
	}
	if err := r.syncDimCNAE(ctx); err != nil {
		return fmt.Errorf("sync dim_cnae: %w", err)
	}
	if err := r.processCredits(ctx); err != nil {
		return fmt.Errorf("process credits: %w", err)
	}
	if err := r.processSearches(ctx); err != nil {
		return fmt.Errorf("process searches: %w", err)
	}
	return nil
}

func (r *postgresRepo) syncDimOrganizacao(ctx context.Context) error {
	const q = `
		INSERT INTO dim_organizacao (id, name, plan_name, is_active)
		SELECT o.id, o.name, p.name, o.is_active
		FROM tb_organizations o
		LEFT JOIN tb_plans p ON p.id = o.plan_id
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name, plan_name = EXCLUDED.plan_name, is_active = EXCLUDED.is_active`
	if _, err := r.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *postgresRepo) syncDimUsuario(ctx context.Context) error {
	const q = `
		INSERT INTO dim_usuario (id, email, role, org_id)
		SELECT id, email, role, org_id FROM tb_users
		ON CONFLICT (id) DO UPDATE
		SET email = EXCLUDED.email, role = EXCLUDED.role, org_id = EXCLUDED.org_id`
	if _, err := r.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *postgresRepo) syncDimCNAE(ctx context.Context) error {
	const q = `
		INSERT INTO dim_cnae (code, description)
		SELECT code, description FROM tb_cnaes
		ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description`
	if _, err := r.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *postgresRepo) processCredits(ctx context.Context) error {
	// Process in batches of 1000 until no more unprocessed rows.
	// The eligible CTE ensures only rows whose org is already in dim_organizacao
	// are inserted AND marked; rows for recently created orgs are retried next run.
	for {
		const q = `
			WITH batch AS (
				SELECT id, org_id, user_id, type, amount, created_at
				FROM tb_credit_transactions
				WHERE etl_processed_at IS NULL
				LIMIT 1000
				FOR UPDATE SKIP LOCKED
			),
			eligible AS (
				SELECT b.* FROM batch b
				WHERE b.org_id IN (SELECT id FROM dim_organizacao)
			),
			ins AS (
				INSERT INTO fato_consumo_creditos (tempo_id, org_id, usuario_id, tipo, creditos, eh_entrada)
				SELECT
					TO_CHAR(e.created_at::date, 'YYYYMMDD')::INT,
					e.org_id,
					CASE WHEN e.user_id IN (SELECT id FROM dim_usuario) THEN e.user_id ELSE NULL END,
					e.type,
					ABS(e.amount),
					e.amount > 0
				FROM eligible e
				RETURNING 1
			)
			UPDATE tb_credit_transactions t
			SET etl_processed_at = now()
			FROM eligible e
			WHERE t.id = e.id`

		tag, err := r.db.Exec(ctx, q)
		if err != nil {
			return fmt.Errorf("process credits batch: %w", err)
		}
		if tag.RowsAffected() == 0 {
			break
		}
	}
	return nil
}

func (r *postgresRepo) processSearches(ctx context.Context) error {
	// Same eligible-CTE pattern as processCredits: only rows with a synced org
	// are inserted and marked; the rest are retried next ETL run.
	// Only cnaes[0] is used to classify each search; multi-CNAE searches attribute
	// all leads to the first CNAE — a known simplification until STEP-07 refines it.
	for {
		const q = `
			WITH batch AS (
				SELECT id, org_id, user_id, filters, result_count, done_at
				FROM tb_searches
				WHERE etl_processed_at IS NULL AND status = 'done' AND done_at IS NOT NULL
				LIMIT 1000
				FOR UPDATE SKIP LOCKED
			),
			eligible AS (
				SELECT b.* FROM batch b
				WHERE b.org_id IN (SELECT id FROM dim_organizacao)
			),
			ins AS (
				INSERT INTO fato_funil_leads
					(tempo_id, org_id, usuario_id, cnae_id, geo_id, leads_extraidos, search_id)
				SELECT
					TO_CHAR(e.done_at::date, 'YYYYMMDD')::INT,
					e.org_id,
					CASE WHEN e.user_id IN (SELECT id FROM dim_usuario) THEN e.user_id ELSE NULL END,
					CASE
						WHEN e.filters#>>'{cnaes,0}' IS NOT NULL
						 AND e.filters#>>'{cnaes,0}' IN (SELECT code FROM dim_cnae)
						THEN e.filters#>>'{cnaes,0}'
						ELSE NULL
					END,
					NULL,
					COALESCE(e.result_count, 0),
					e.id
				FROM eligible e
				RETURNING 1
			)
			UPDATE tb_searches t
			SET etl_processed_at = now()
			FROM eligible e
			WHERE t.id = e.id`

		tag, err := r.db.Exec(ctx, q)
		if err != nil {
			return fmt.Errorf("process searches batch: %w", err)
		}
		if tag.RowsAffected() == 0 {
			break
		}
	}
	return nil
}
