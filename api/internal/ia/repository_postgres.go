package ia

import (
	"context"
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

func (r *postgresRepo) Save(ctx context.Context, q *domain.AIQualification) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tb_ai_qualifications
		  (cnpj, org_id, user_id, score, justification, prompt_used, model_used,
		   tokens_input, tokens_output, raw_response)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb)
		RETURNING id, created_at`,
		q.CNPJ, q.OrgID, q.UserID, q.Score, q.Justification, q.PromptUsed,
		q.ModelUsed, q.TokensInput, q.TokensOutput,
		fmt.Sprintf(`{"justification":%q}`, q.Justification),
	).Scan(&q.ID, &q.CreatedAt)
	if err != nil {
		return fmt.Errorf("save qualification: %w", err)
	}
	return nil
}

func (r *postgresRepo) FindRecent(ctx context.Context, cnpj, orgID string, maxAge time.Duration) (*domain.AIQualification, error) {
	since := time.Now().Add(-maxAge)
	row := r.db.QueryRow(ctx, `
		SELECT id, cnpj, org_id, user_id, score, justification, model_used,
		       tokens_input, tokens_output, created_at
		FROM tb_ai_qualifications
		WHERE cnpj=$1 AND org_id=$2 AND created_at >= $3
		ORDER BY created_at DESC
		LIMIT 1`,
		cnpj, orgID, since,
	)
	q := &domain.AIQualification{}
	err := row.Scan(
		&q.ID, &q.CNPJ, &q.OrgID, &q.UserID,
		&q.Score, &q.Justification, &q.ModelUsed,
		&q.TokensInput, &q.TokensOutput, &q.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find recent qualification: %w", err)
	}
	return q, nil
}

func (r *postgresRepo) List(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error) {
	query := `
		SELECT id, cnpj, org_id, user_id, score, justification, model_used,
		       tokens_input, tokens_output, created_at
		FROM tb_ai_qualifications
		WHERE org_id=$1`
	args := []any{orgID}
	if cnpj != nil {
		query += " AND cnpj=$2"
		args = append(args, *cnpj)
	}
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list qualifications: %w", err)
	}
	defer rows.Close()

	var out []domain.AIQualification
	for rows.Next() {
		var q domain.AIQualification
		if err := rows.Scan(
			&q.ID, &q.CNPJ, &q.OrgID, &q.UserID,
			&q.Score, &q.Justification, &q.ModelUsed,
			&q.TokensInput, &q.TokensOutput, &q.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan qualification: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if out == nil {
		out = []domain.AIQualification{}
	}
	return out, nil
}

func (r *postgresRepo) GetCompanyPromptData(ctx context.Context, cnpj string) (*CompanyPromptData, error) {
	data := &CompanyPromptData{}
	var dataInicio *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT c.cnpj, c.razao_social, c.municipio_nome, c.uf,
		       c.capital_social, c.situacao_cadastral, c.data_inicio,
		       c.porte, c.opcao_simples,
		       (SELECT cc.cnae_code FROM tb_company_cnaes cc
		        WHERE cc.cnpj=c.cnpj AND cc.is_primary=true LIMIT 1)
		FROM tb_companies c
		WHERE c.cnpj=$1`,
		cnpj,
	).Scan(
		&data.CNPJ, &data.RazaoSocial, &data.Municipio, &data.UF,
		&data.CapitalSocial, &data.SituacaoCadastral, &dataInicio,
		&data.Porte, &data.OpcaoSimples, &data.PrimaryCNAE,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get company prompt data: %w", err)
	}
	if dataInicio != nil {
		s := dataInicio.Format("2006-01-02")
		data.DataInicio = &s
	}
	return data, nil
}
