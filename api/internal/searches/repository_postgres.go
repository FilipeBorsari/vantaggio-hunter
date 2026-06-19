package searches

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/pkg/brazil"
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
		     done_at=CASE WHEN $5 IN ('done','failed') THEN now() ELSE done_at END
		 WHERE id=$4`,
		status, resultCount, errMsg, id, string(status),
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

func (r *postgresRepo) ListQueuedSearchIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM tb_searches WHERE status = 'queued'`)
	if err != nil {
		return nil, fmt.Errorf("list queued searches: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan queued search id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *postgresRepo) RunStructuredSearch(ctx context.Context, searchID, orgID string, f domain.SearchFilters) (int, error) {
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
	conds = append(conds, fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM tb_search_results sr2
		INNER JOIN tb_searches s2 ON sr2.search_id = s2.id
		WHERE sr2.cnpj = c.cnpj AND s2.org_id = $%d
	)`, n))
	args = append(args, orgID)
	n++
	_ = n

	where := "WHERE " + strings.Join(conds, " AND ")

	limitVal := 10000
	if f.MaxResults != nil && *f.MaxResults > 0 && *f.MaxResults < limitVal {
		limitVal = *f.MaxResults
	}

	q := fmt.Sprintf(`
		SELECT DISTINCT c.cnpj
		FROM tb_companies c
		%s
		ORDER BY c.cnpj
		LIMIT %d`, where, limitVal)

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

func (r *postgresRepo) RunSemanticSearch(ctx context.Context, searchID, orgID string, f domain.SearchFilters, queryVec []float32, queryText string) (int, error) {
	count, err := r.runVectorSearch(ctx, searchID, orgID, f, queryVec)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return count, nil
	}
	// Fallback: nenhuma empresa tem embedding ainda — usa full-text search em razao_social.
	return r.runTextFallback(ctx, searchID, orgID, f, queryText)
}

func (r *postgresRepo) runVectorSearch(ctx context.Context, searchID, orgID string, f domain.SearchFilters, queryVec []float32) (int, error) {
	var conds []string
	args := []any{vectorLiteral(queryVec)} // $1 = embedding vector
	n := 2

	conds = append(conds, "c.embedding IS NOT NULL")

	if f.UF != nil && *f.UF != "" {
		conds = append(conds, fmt.Sprintf(`c.uf=$%d`, n))
		args = append(args, strings.ToUpper(*f.UF))
		n++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf(`c.situacao_cadastral=$%d`, n))
		args = append(args, *f.Status)
		n++
	}
	conds = append(conds, fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM tb_search_results sr2
		INNER JOIN tb_searches s2 ON sr2.search_id = s2.id
		WHERE sr2.cnpj = c.cnpj AND s2.org_id = $%d
	)`, n))
	args = append(args, orgID)
	n++
	_ = n

	where := "WHERE " + strings.Join(conds, " AND ")

	limitVal := 10000
	if f.MaxResults != nil && *f.MaxResults > 0 && *f.MaxResults < limitVal {
		limitVal = *f.MaxResults
	}

	q := fmt.Sprintf(`
		SELECT c.cnpj, c.embedding <=> $1::vector AS score
		FROM tb_companies c
		%s
		ORDER BY score ASC
		LIMIT %d`, where, limitVal)

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

// ptSearchStopWords are words too generic to be useful when matching against company names.
var ptSearchStopWords = map[string]bool{
	"para": true, "como": true, "mais": true, "menos": true, "muito": true, "pouco": true,
	"todos": true, "todas": true, "este": true, "esta": true, "esse": true, "essa": true,
	"isso": true, "outro": true, "outra": true,
	"empresa": true, "empresas": true, "negocio": true, "negocios": true, "ramo": true,
	"setor": true, "area": true, "segmento": true, "tipo": true,
	"servico": true, "servicos": true, "produto": true, "produtos": true,
	"cliente": true, "clientes": true, "mercado": true, "atividade": true, "atividades": true,
}

// extractKeywords returns meaningful words from a natural-language query for ILIKE matching.
// Words shorter than 5 characters or in the stop-word list are excluded.
func extractKeywords(text string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len([]rune(w)) >= 5 && !ptSearchStopWords[w] && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}

func (r *postgresRepo) runTextFallback(ctx context.Context, searchID, orgID string, f domain.SearchFilters, queryText string) (int, error) {
	var conds []string
	var args []any
	n := 1

	// Use OR-based keyword matching so natural language descriptions like
	// "empresas do ramo de energia solar" match companies whose razao_social or
	// nome_fantasia contains any of the meaningful terms ("energia", "solar", …).
	keywords := extractKeywords(queryText)
	if len(keywords) == 0 {
		return 0, nil
	}
	var orParts []string
	for _, kw := range keywords {
		orParts = append(orParts, fmt.Sprintf(
			`(c.razao_social ILIKE '%%' || $%d || '%%' OR c.nome_fantasia ILIKE '%%' || $%d || '%%')`, n, n,
		))
		args = append(args, kw)
		n++
	}
	conds = append(conds, "("+strings.Join(orParts, " OR ")+")")

	if f.UF != nil && *f.UF != "" {
		conds = append(conds, fmt.Sprintf(`c.uf=$%d`, n))
		args = append(args, strings.ToUpper(*f.UF))
		n++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf(`c.situacao_cadastral=$%d`, n))
		args = append(args, *f.Status)
		n++
	}
	conds = append(conds, fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM tb_search_results sr2
		INNER JOIN tb_searches s2 ON sr2.search_id = s2.id
		WHERE sr2.cnpj = c.cnpj AND s2.org_id = $%d
	)`, n))
	args = append(args, orgID)
	n++
	_ = n

	where := "WHERE " + strings.Join(conds, " AND ")

	limitVal := 10000
	if f.MaxResults != nil && *f.MaxResults > 0 && *f.MaxResults < limitVal {
		limitVal = *f.MaxResults
	}

	q := fmt.Sprintf(`
		SELECT DISTINCT c.cnpj
		FROM tb_companies c
		%s
		LIMIT %d`, where, limitVal)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("text fallback query: %w", err)
	}
	defer rows.Close()

	var copyRows [][]any
	pos := 0
	for rows.Next() {
		var cnpj string
		if err := rows.Scan(&cnpj); err != nil {
			return 0, fmt.Errorf("scan text fallback result: %w", err)
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
			return 0, fmt.Errorf("save text fallback results: %w", err)
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
	codePattern := brazil.NormalizeCNAE(q)
	if codePattern == "" {
		codePattern = q
	}
	rows, err := r.db.Query(ctx,
		`SELECT code, description FROM tb_cnaes
		 WHERE description ILIKE $1 OR code ILIKE $2
		 ORDER BY code LIMIT 20`,
		"%"+q+"%", "%"+codePattern+"%",
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

func (r *postgresRepo) GetCompanyEmbedInputs(ctx context.Context, searchID string, limit int) ([]domain.CompanyEmbedInput, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.cnpj, c.uf,
		       c.razao_social
		       || ' | ' || coalesce(tc.description, '')
		       || ' | ' || coalesce(c.municipio_nome, '') || ' ' || c.uf
		       || ' | situacao:' || c.situacao_cadastral::text
		       || ' capital:' || coalesce(c.capital_social::text, '0')
		FROM tb_search_results sr
		JOIN tb_companies c ON c.cnpj = sr.cnpj
		LEFT JOIN tb_company_cnaes cc ON cc.cnpj = c.cnpj AND cc.is_primary = true
		LEFT JOIN tb_cnaes tc ON tc.code = cc.cnae_code
		WHERE sr.search_id = $1 AND c.embedding IS NULL
		LIMIT $2`,
		searchID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get embed inputs: %w", err)
	}
	defer rows.Close()

	var inputs []domain.CompanyEmbedInput
	for rows.Next() {
		var inp domain.CompanyEmbedInput
		if err := rows.Scan(&inp.CNPJ, &inp.UF, &inp.Text); err != nil {
			return nil, fmt.Errorf("scan embed input: %w", err)
		}
		inputs = append(inputs, inp)
	}
	return inputs, rows.Err()
}

func (r *postgresRepo) SaveEmbeddings(ctx context.Context, embeddings []domain.CompanyEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, e := range embeddings {
		batch.Queue(`
			UPDATE tb_companies
			SET embedding = $1::vector, embedding_updated_at = now()
			WHERE cnpj = $2 AND uf = $3`,
			vectorLiteral(e.Vector), e.CNPJ, e.UF,
		)
	}
	results := r.db.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("save embedding: %w", err)
		}
	}
	return nil
}

func (r *postgresRepo) EstimateCount(ctx context.Context, mode domain.SearchMode, f domain.SearchFilters, queryText string) (int, error) {
	switch mode {
	case domain.SearchModeStructured:
		return r.countStructured(ctx, f)
	case domain.SearchModeSemantic:
		return r.countSemantic(ctx, f, queryText)
	default:
		return 0, fmt.Errorf("unknown mode: %s", mode)
	}
}

func (r *postgresRepo) countStructured(ctx context.Context, f domain.SearchFilters) (int, error) {
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

	var count int
	if err := r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(DISTINCT c.cnpj) FROM tb_companies c %s`, where),
		args...,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count structured: %w", err)
	}
	return count, nil
}

func (r *postgresRepo) countSemantic(ctx context.Context, f domain.SearchFilters, queryText string) (int, error) {
	keywords := extractKeywords(queryText)
	if len(keywords) == 0 {
		return 0, nil
	}

	var conds []string
	var args []any
	n := 1

	var orParts []string
	for _, kw := range keywords {
		orParts = append(orParts, fmt.Sprintf(
			`(c.razao_social ILIKE '%%' || $%d || '%%' OR c.nome_fantasia ILIKE '%%' || $%d || '%%')`, n, n,
		))
		args = append(args, kw)
		n++
	}
	conds = append(conds, "("+strings.Join(orParts, " OR ")+")")

	if f.UF != nil && *f.UF != "" {
		conds = append(conds, fmt.Sprintf(`c.uf=$%d`, n))
		args = append(args, strings.ToUpper(*f.UF))
		n++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf(`c.situacao_cadastral=$%d`, n))
		args = append(args, *f.Status)
		n++
	}
	_ = n

	var count int
	if err := r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(DISTINCT c.cnpj) FROM tb_companies c WHERE %s`, strings.Join(conds, " AND ")),
		args...,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count semantic: %w", err)
	}
	return count, nil
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
