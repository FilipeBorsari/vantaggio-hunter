package searches

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (r *postgresRepo) Create(ctx context.Context, s *domain.Search) error {
	filtersJSON, err := json.Marshal(s.Filters)
	if err != nil {
		return fmt.Errorf("marshal filters: %w", err)
	}
	var createdAt time.Time
	err = r.db.QueryRow(ctx,
		`INSERT INTO tb_searches (org_id, user_id, mode, filters, query_text)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		s.OrgID, s.UserID, s.Mode, filtersJSON, s.QueryText,
	).Scan(&s.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("create search: %w", err)
	}
	s.CreatedAt = createdAt.Format(time.RFC3339)
	s.Status = domain.SearchStatusQueued
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id, orgID string) (*domain.Search, error) {
	return r.scanSearch(ctx,
		`SELECT id, org_id, user_id, mode, filters, query_text, status, result_count, error_msg, created_at, done_at
		 FROM tb_searches WHERE id=$1 AND org_id=$2`,
		id, orgID,
	)
}

func (r *postgresRepo) GetByIDForWorker(ctx context.Context, id string) (*domain.Search, error) {
	return r.scanSearch(ctx,
		`SELECT id, org_id, user_id, mode, filters, query_text, status, result_count, error_msg, created_at, done_at
		 FROM tb_searches WHERE id=$1`,
		id,
	)
}

func (r *postgresRepo) scanSearch(ctx context.Context, q string, args ...any) (*domain.Search, error) {
	var s domain.Search
	var filtersJSON []byte
	var createdAt time.Time
	var doneAt *time.Time

	err := r.db.QueryRow(ctx, q, args...).Scan(
		&s.ID, &s.OrgID, &s.UserID, &s.Mode, &filtersJSON, &s.QueryText,
		&s.Status, &s.ResultCount, &s.ErrorMsg, &createdAt, &doneAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get search: %w", err)
	}
	s.CreatedAt = createdAt.Format(time.RFC3339)
	if doneAt != nil {
		t := doneAt.Format(time.RFC3339)
		s.DoneAt = &t
	}
	if err := json.Unmarshal(filtersJSON, &s.Filters); err != nil {
		return nil, fmt.Errorf("unmarshal filters: %w", err)
	}
	return &s, nil
}

func (r *postgresRepo) List(ctx context.Context, orgID string, page, limit int) ([]domain.Search, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tb_searches WHERE org_id=$1`, orgID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count searches: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, user_id, mode, filters, query_text, status, result_count, error_msg, created_at, done_at
		 FROM tb_searches WHERE org_id=$1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list searches: %w", err)
	}
	defer rows.Close()

	var result []domain.Search
	for rows.Next() {
		var s domain.Search
		var filtersJSON []byte
		var createdAt time.Time
		var doneAt *time.Time
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.UserID, &s.Mode, &filtersJSON, &s.QueryText,
			&s.Status, &s.ResultCount, &s.ErrorMsg, &createdAt, &doneAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan search: %w", err)
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		if doneAt != nil {
			t := doneAt.Format(time.RFC3339)
			s.DoneAt = &t
		}
		if err := json.Unmarshal(filtersJSON, &s.Filters); err != nil {
			return nil, 0, fmt.Errorf("unmarshal filters: %w", err)
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	if result == nil {
		result = []domain.Search{}
	}
	return result, total, nil
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id string, status domain.SearchStatus, resultCount *int, errMsg *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tb_searches
		 SET status=$1, result_count=$2, error_msg=$3,
		     done_at=CASE WHEN $1 IN ('done','failed') THEN now() ELSE done_at END
		 WHERE id=$4`,
		status, resultCount, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update search status: %w", err)
	}
	return nil
}

func (r *postgresRepo) RecoverStaleSearches(ctx context.Context, staleMinutes int) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE tb_searches
		 SET status = 'queued', error_msg = NULL
		 WHERE status = 'processing'
		   AND created_at < now() - ($1 * INTERVAL '1 minute')`,
		staleMinutes,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale searches: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *postgresRepo) RunStructuredSearch(ctx context.Context, searchID string, f domain.SearchFilters) (int, error) {
	var conds []string
	var args []any
	n := 1

	if len(f.CNAEs) > 0 {
		conds = append(conds, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM tb_company_cnaes cc WHERE cc.cnpj=c.cnpj AND cc.cnae_code=ANY($%d))`, n,
		))
		args = append(args, f.CNAEs)
		n++
	}
	if f.UF != nil && *f.UF != "" {
		conds = append(conds, fmt.Sprintf(`c.uf=$%d`, n))
		args = append(args, strings.ToUpper(*f.UF))
		n++
	}
	if f.City != nil && *f.City != "" {
		conds = append(conds, fmt.Sprintf(`c.municipio_nome ILIKE $%d`, n))
		args = append(args, "%"+*f.City+"%")
		n++
	}
	if f.CapitalMin != nil {
		conds = append(conds, fmt.Sprintf(`c.capital_social>=$%d`, n))
		args = append(args, *f.CapitalMin)
		n++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf(`c.situacao_cadastral=$%d`, n))
		args = append(args, *f.Status)
		n++
	}
	_ = n

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	q := fmt.Sprintf(`
		SELECT c.cnpj
		FROM tb_companies c
		%s
		ORDER BY c.cnpj
		LIMIT 10000`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("structured search query: %w", err)
	}
	defer rows.Close()

	var copyRows [][]any
	pos := 0
	for rows.Next() {
		var cnpj string
		if err := rows.Scan(&cnpj); err != nil {
			return 0, fmt.Errorf("scan cnpj: %w", err)
		}
		copyRows = append(copyRows, []any{searchID, cnpj, nil, pos})
		pos++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("rows error: %w", err)
	}

	if len(copyRows) > 0 {
		if _, err := r.db.CopyFrom(ctx,
			pgx.Identifier{"tb_search_results"},
			[]string{"search_id", "cnpj", "score", "position"},
			pgx.CopyFromRows(copyRows),
		); err != nil {
			return 0, fmt.Errorf("save structured results: %w", err)
		}
	}
	return len(copyRows), nil
}

func (r *postgresRepo) RunSemanticSearch(ctx context.Context, searchID string, f domain.SearchFilters, queryVec []float32) (int, error) {
	var conds []string
	args := []any{vectorLiteral(queryVec)} // $1 = embedding vector
	n := 2

	conds = append(conds, "c.embedding IS NOT NULL")
	conds = append(conds, "c.situacao_cadastral = 2")

	if f.UF != nil && *f.UF != "" {
		conds = append(conds, fmt.Sprintf(`c.uf=$%d`, n))
		args = append(args, strings.ToUpper(*f.UF))
		n++
	}
	_ = n

	where := "WHERE " + strings.Join(conds, " AND ")

	q := fmt.Sprintf(`
		SELECT c.cnpj, c.embedding <=> $1::vector AS score
		FROM tb_companies c
		%s
		ORDER BY score ASC
		LIMIT 10000`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("semantic search query: %w", err)
	}
	defer rows.Close()

	var copyRows [][]any
	pos := 0
	for rows.Next() {
		var cnpj string
		var score float64
		if err := rows.Scan(&cnpj, &score); err != nil {
			return 0, fmt.Errorf("scan semantic result: %w", err)
		}
		copyRows = append(copyRows, []any{searchID, cnpj, score, pos})
		pos++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("rows error: %w", err)
	}

	if len(copyRows) > 0 {
		if _, err := r.db.CopyFrom(ctx,
			pgx.Identifier{"tb_search_results"},
			[]string{"search_id", "cnpj", "score", "position"},
			pgx.CopyFromRows(copyRows),
		); err != nil {
			return 0, fmt.Errorf("save semantic results: %w", err)
		}
	}
	return len(copyRows), nil
}

func (r *postgresRepo) GetResults(ctx context.Context, searchID string, page, limit int) ([]domain.SearchResult, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tb_search_results WHERE search_id=$1`, searchID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count results: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT sr.cnpj, sr.score,
		       c.razao_social, c.municipio_nome, c.uf, c.capital_social, c.situacao_cadastral,
		       aq.score AS ai_score,
		       EXTRACT(DAY FROM now() - aq.created_at)::int AS ai_score_age_days
		FROM tb_search_results sr
		JOIN tb_companies c ON c.cnpj=sr.cnpj
		LEFT JOIN LATERAL (
		    SELECT score, created_at
		    FROM tb_ai_qualifications
		    WHERE cnpj=sr.cnpj AND org_id=(SELECT org_id FROM tb_searches WHERE id=$1)
		    ORDER BY created_at DESC
		    LIMIT 1
		) aq ON true
		WHERE sr.search_id=$1
		ORDER BY sr.position
		LIMIT $2 OFFSET $3`,
		searchID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get results: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	for rows.Next() {
		var res domain.SearchResult
		res.CNAEs = []domain.CNAE{}
		if err := rows.Scan(
			&res.CNPJ, &res.Score,
			&res.RazaoSocial, &res.Municipio, &res.UF, &res.CapitalSocial, &res.SituacaoCadastral,
			&res.AIScore, &res.AIScoreAgeDays,
		); err != nil {
			return nil, 0, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	if results == nil {
		results = []domain.SearchResult{}
	}
	return results, total, nil
}

func (r *postgresRepo) SearchCNAEs(ctx context.Context, q string) ([]domain.CNAE, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, description FROM tb_cnaes
		 WHERE description ILIKE $1 OR code ILIKE $1
		 ORDER BY code LIMIT 20`,
		"%"+q+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search cnaes: %w", err)
	}
	defer rows.Close()

	var cnaes []domain.CNAE
	for rows.Next() {
		var c domain.CNAE
		if err := rows.Scan(&c.Code, &c.Description); err != nil {
			return nil, fmt.Errorf("scan cnae: %w", err)
		}
		cnaes = append(cnaes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if cnaes == nil {
		cnaes = []domain.CNAE{}
	}
	return cnaes, nil
}

// vectorLiteral formats a float32 slice as pgvector text: "[v1,v2,...]"
func vectorLiteral(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteByte(']')
	return sb.String()
}
