package companies

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func buildWhere(f Filters) (clause string, args []any) {
	var conds []string
	n := 1

	if f.UF != "" {
		conds = append(conds, fmt.Sprintf("c.uf=$%d", n))
		args = append(args, strings.ToUpper(f.UF))
		n++
	}
	if f.City != "" {
		conds = append(conds, fmt.Sprintf("c.municipio_nome ILIKE $%d", n))
		args = append(args, "%"+f.City+"%")
		n++
	}
	if f.CapitalMin != nil {
		conds = append(conds, fmt.Sprintf("c.capital_social>=$%d", n))
		args = append(args, *f.CapitalMin)
		n++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf("c.situacao_cadastral=$%d", n))
		args = append(args, *f.Status)
		n++
	}
	if len(f.CNAEs) > 0 {
		conds = append(conds, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM tb_company_cnaes cc WHERE cc.cnpj=c.cnpj AND cc.cnae_code=ANY($%d))`, n,
		))
		args = append(args, f.CNAEs)
		n++
	}
	_ = n // last value unused after loop

	if len(conds) > 0 {
		clause = "WHERE " + strings.Join(conds, " AND ")
	}
	return clause, args
}

func (r *postgresRepo) Count(ctx context.Context, f Filters) (int, error) {
	where, args := buildWhere(f)
	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM tb_companies c %s`, where),
		args...,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count companies: %w", err)
	}
	return total, nil
}

func (r *postgresRepo) List(ctx context.Context, f Filters) ([]domain.Company, error) {
	where, args := buildWhere(f)
	n := len(args) + 1
	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)
	q := fmt.Sprintf(`
		SELECT c.cnpj, c.razao_social, c.nome_fantasia, c.municipio_nome, c.uf,
		       c.capital_social, c.situacao_cadastral
		FROM tb_companies c
		%s
		ORDER BY c.razao_social
		LIMIT $%d OFFSET $%d`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	result := []domain.Company{}
	for rows.Next() {
		var c domain.Company
		c.CNAEs = []domain.CNAE{}
		if err := rows.Scan(
			&c.CNPJ, &c.RazaoSocial, &c.NomeFantasia, &c.Municipio, &c.UF,
			&c.CapitalSocial, &c.SituacaoCadastral,
		); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return result, nil
}

func (r *postgresRepo) AttachCNAEs(ctx context.Context, companies []domain.Company) error {
	if len(companies) == 0 {
		return nil
	}
	cnpjs := make([]string, len(companies))
	index := make(map[string]int, len(companies))
	for i, c := range companies {
		cnpjs[i] = c.CNPJ
		index[c.CNPJ] = i
	}

	rows, err := r.db.Query(ctx,
		`SELECT cc.cnpj, cc.cnae_code, cc.is_primary, COALESCE(cn.description,'')
		 FROM tb_company_cnaes cc
		 LEFT JOIN tb_cnaes cn ON cn.code=cc.cnae_code
		 WHERE cc.cnpj=ANY($1)`,
		cnpjs,
	)
	if err != nil {
		// CNAEs are optional — log but do not fail the listing
		slog.WarnContext(ctx, "attach cnaes: query failed", "error", err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var cnpj, code, desc string
		var primary bool
		if err := rows.Scan(&cnpj, &code, &primary, &desc); err != nil {
			continue
		}
		if i, ok := index[cnpj]; ok {
			companies[i].CNAEs = append(companies[i].CNAEs, domain.CNAE{Code: code, Description: desc, IsPrimary: primary})
		}
	}
	return nil
}

func (r *postgresRepo) GetByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyDetail, error) {
	var d domain.CompanyDetail
	var dataInicio *time.Time
	d.CNAEs = []domain.CNAE{}
	d.Partners = []domain.Partner{}

	err := r.db.QueryRow(ctx,
		`SELECT cnpj, razao_social, nome_fantasia, logradouro, numero, complemento, bairro, cep,
		        municipio_nome, uf, capital_social, situacao_cadastral, porte, opcao_simples,
		        data_inicio, ddd_telefone1, telefone1, email
		 FROM tb_companies WHERE cnpj=$1 LIMIT 1`,
		cnpj,
	).Scan(
		&d.CNPJ, &d.RazaoSocial, &d.NomeFantasia, &d.Logradouro, &d.Numero, &d.Complemento,
		&d.Bairro, &d.CEP, &d.Municipio, &d.UF, &d.CapitalSocial, &d.SituacaoCadastral,
		&d.Porte, &d.OpcaoSimples, &dataInicio, &d.DDDTelefone1, &d.Telefone1, &d.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get company by cnpj: %w", err)
	}
	if dataInicio != nil {
		s := dataInicio.Format("2006-01-02")
		d.DataInicio = &s
	}
	return &d, nil
}

func (r *postgresRepo) GetCNAEsByCNPJ(ctx context.Context, cnpj string) ([]domain.CNAE, error) {
	rows, err := r.db.Query(ctx,
		`SELECT cc.cnae_code, cc.is_primary, COALESCE(cn.description,'')
		 FROM tb_company_cnaes cc
		 LEFT JOIN tb_cnaes cn ON cn.code=cc.cnae_code
		 WHERE cc.cnpj=$1`,
		cnpj,
	)
	if err != nil {
		return nil, fmt.Errorf("get cnaes: %w", err)
	}
	defer rows.Close()
	cnaes := []domain.CNAE{}
	for rows.Next() {
		var c domain.CNAE
		if err := rows.Scan(&c.Code, &c.IsPrimary, &c.Description); err != nil {
			return nil, fmt.Errorf("scan cnae: %w", err)
		}
		cnaes = append(cnaes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return cnaes, nil
}

func (r *postgresRepo) GetPartnersByCNPJBasico(ctx context.Context, cnpjBasico string) ([]domain.Partner, error) {
	rows, err := r.db.Query(ctx,
		`SELECT nome_socio, cpf_cnpj_socio, qualificacao, data_entrada
		 FROM tb_partners WHERE cnpj_basico=$1`,
		cnpjBasico,
	)
	if err != nil {
		return nil, fmt.Errorf("get partners: %w", err)
	}
	defer rows.Close()
	partners := []domain.Partner{}
	for rows.Next() {
		var p domain.Partner
		var dataEntrada *time.Time
		if err := rows.Scan(&p.Nome, &p.CPFCNPJSocio, &p.Qualificacao, &dataEntrada); err != nil {
			return nil, fmt.Errorf("scan partner: %w", err)
		}
		if dataEntrada != nil {
			s := dataEntrada.Format("2006-01-02")
			p.DataEntrada = &s
		}
		partners = append(partners, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return partners, nil
}
