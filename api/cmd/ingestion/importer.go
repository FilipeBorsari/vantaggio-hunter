package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// empresaData holds the fields pulled from staging_empresas + staging_simples.
type empresaData struct {
	RazaoSocial      string
	NaturezaJuridica string
	CapitalSocial    float64
	Porte            int16
	OpcaoSimples     bool
}

// Importer handles all database write operations for the ingestion pipeline.
type Importer struct {
	pool      *pgxpool.Pool
	batchSize int
}

func NewImporter(pool *pgxpool.Pool, batchSize int) *Importer {
	return &Importer{pool: pool, batchSize: batchSize}
}

// CreateStagingTables creates temporary staging tables used during ingestion.
// These hold Empresas and Simples data so Estabelecimentos can be enriched
// without loading 60M+ records into memory.
func (imp *Importer) CreateStagingTables(ctx context.Context) error {
	_, err := imp.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS staging_empresas (
			cnpj_basico       VARCHAR(8)    PRIMARY KEY,
			razao_social      TEXT          NOT NULL DEFAULT '',
			natureza_juridica VARCHAR(4)    NOT NULL DEFAULT '',
			capital_social    NUMERIC(18,2) NOT NULL DEFAULT 0,
			porte             SMALLINT      NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS staging_simples (
			cnpj_basico   VARCHAR(8) PRIMARY KEY,
			opcao_simples BOOLEAN    NOT NULL DEFAULT false
		);
	`)
	return err
}

// DropStagingTables removes the staging tables after ingestion completes.
func (imp *Importer) DropStagingTables(ctx context.Context) error {
	_, err := imp.pool.Exec(ctx, `
		DROP TABLE IF EXISTS staging_empresas;
		DROP TABLE IF EXISTS staging_simples;
	`)
	return err
}

// UpsertCNAEs inserts or updates CNAE rows into tb_cnaes.
func (imp *Importer) UpsertCNAEs(ctx context.Context, rows []CNAERow) error {
	batch := &pgx.Batch{}
	for _, r := range rows {
		if r.Code == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO tb_cnaes (code, description) VALUES ($1, $2)
			ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description
		`, r.Code, r.Description)
	}
	return sendBatch(ctx, imp.pool, batch)
}

// UpsertEmpresas streams Empresa rows into staging_empresas.
func (imp *Importer) UpsertEmpresas(ctx context.Context, rows []EmpresaRow) error {
	batch := &pgx.Batch{}
	for _, r := range rows {
		if r.CNPJBasico == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO staging_empresas (cnpj_basico, razao_social, natureza_juridica, capital_social, porte)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (cnpj_basico) DO UPDATE SET
				razao_social      = EXCLUDED.razao_social,
				natureza_juridica = EXCLUDED.natureza_juridica,
				capital_social    = EXCLUDED.capital_social,
				porte             = EXCLUDED.porte
		`, r.CNPJBasico, r.RazaoSocial, r.NaturezaJuridica, r.CapitalSocial, r.Porte)
	}
	return sendBatch(ctx, imp.pool, batch)
}

// UpsertSimples streams Simples rows into staging_simples.
func (imp *Importer) UpsertSimples(ctx context.Context, rows []SimplesRow) error {
	batch := &pgx.Batch{}
	for _, r := range rows {
		if r.CNPJBasico == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO staging_simples (cnpj_basico, opcao_simples) VALUES ($1, $2)
			ON CONFLICT (cnpj_basico) DO UPDATE SET opcao_simples = EXCLUDED.opcao_simples
		`, r.CNPJBasico, r.OpcaoSimples)
	}
	return sendBatch(ctx, imp.pool, batch)
}

// UpsertEstabelecimentos inserts a batch of Estabelecimento rows into
// tb_companies and tb_company_cnaes, enriching each row from the staging tables.
func (imp *Importer) UpsertEstabelecimentos(ctx context.Context, rows []EstabelecimentoRow, munMap map[int]string) error {
	if len(rows) == 0 {
		return nil
	}

	// Collect unique cnpj_basico values for the batch lookup.
	seen := make(map[string]struct{}, len(rows))
	basicos := make([]string, 0, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.CNPJBasico]; !ok {
			seen[r.CNPJBasico] = struct{}{}
			basicos = append(basicos, r.CNPJBasico)
		}
	}

	// Fetch empresa + simples data for this batch in a single query.
	empresas, err := imp.fetchStagingData(ctx, basicos)
	if err != nil {
		return fmt.Errorf("fetch staging data: %w", err)
	}

	batch := &pgx.Batch{}
	for _, r := range rows {
		e := empresas[r.CNPJBasico] // zero-value if not found (acceptable)
		munNome := munMap[r.MunicipioCode]

		var municipioID *int
		if r.MunicipioCode > 0 {
			municipioID = &r.MunicipioCode
		}

		batch.Queue(`
			INSERT INTO tb_companies (
				cnpj, razao_social, nome_fantasia, situacao_cadastral, data_situacao,
				natureza_juridica, logradouro, numero, complemento, bairro, cep,
				uf, municipio_id, municipio_nome, ddd_telefone1, telefone1, email,
				capital_social, porte, opcao_simples, data_inicio
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
			)
			ON CONFLICT (cnpj, uf) DO UPDATE SET
				razao_social        = EXCLUDED.razao_social,
				nome_fantasia       = EXCLUDED.nome_fantasia,
				situacao_cadastral  = EXCLUDED.situacao_cadastral,
				data_situacao       = EXCLUDED.data_situacao,
				natureza_juridica   = EXCLUDED.natureza_juridica,
				logradouro          = EXCLUDED.logradouro,
				numero              = EXCLUDED.numero,
				complemento         = EXCLUDED.complemento,
				bairro              = EXCLUDED.bairro,
				cep                 = EXCLUDED.cep,
				municipio_id        = EXCLUDED.municipio_id,
				municipio_nome      = EXCLUDED.municipio_nome,
				ddd_telefone1       = EXCLUDED.ddd_telefone1,
				telefone1           = EXCLUDED.telefone1,
				email               = EXCLUDED.email,
				capital_social      = EXCLUDED.capital_social,
				porte               = EXCLUDED.porte,
				opcao_simples       = EXCLUDED.opcao_simples,
				data_inicio         = EXCLUDED.data_inicio
		`,
			r.CNPJ, e.RazaoSocial, nullStr(r.NomeFantasia), r.SituacaoCadastral, r.DataSituacao,
			nullStr(e.NaturezaJuridica), nullStr(r.Logradouro), nullStr(r.Numero), nullStr(r.Complemento),
			nullStr(r.Bairro), nullStr(r.CEP),
			r.UF, municipioID, nullStr(munNome), nullStr(r.DDD1), nullStr(r.Telefone1), nullStr(r.Email),
			e.CapitalSocial, e.Porte, e.OpcaoSimples, r.DataInicio,
		)

		// Primary CNAE
		if r.CNAEPrincipal != "" {
			batch.Queue(`
				INSERT INTO tb_company_cnaes (cnpj, cnae_code, is_primary) VALUES ($1, $2, true)
				ON CONFLICT (cnpj, cnae_code) DO UPDATE SET is_primary = true
			`, r.CNPJ, r.CNAEPrincipal)
		}

		// Secondary CNAEs
		for _, code := range r.CNAEsSecundarios {
			batch.Queue(`
				INSERT INTO tb_company_cnaes (cnpj, cnae_code, is_primary) VALUES ($1, $2, false)
				ON CONFLICT (cnpj, cnae_code) DO NOTHING
			`, r.CNPJ, code)
		}
	}

	return sendBatch(ctx, imp.pool, batch)
}

// UpsertSocios inserts partner rows into tb_partners.
func (imp *Importer) UpsertSocios(ctx context.Context, rows []SocioRow) error {
	batch := &pgx.Batch{}
	for _, r := range rows {
		if r.CNPJBasico == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO tb_partners (cnpj_basico, nome_socio, cpf_cnpj_socio, qualificacao, data_entrada, pais, faixa_etaria)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, r.CNPJBasico, r.NomeSocio, nullStr(r.CPFCNPJSocio), nullInt16(r.Qualificacao),
			r.DataEntrada, nullStr(r.Pais), nullInt16(r.FaixaEtaria))
	}
	return sendBatch(ctx, imp.pool, batch)
}

// CreateHNSWIndex builds the vector similarity index after all embeddings are loaded.
// This is intentionally run after ingestion — building it incrementally during inserts
// is significantly slower for large datasets.
func (imp *Importer) CreateHNSWIndex(ctx context.Context) error {
	_, err := imp.pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_companies_embedding
		ON tb_companies USING hnsw (embedding vector_cosine_ops)
		WITH (m = 16, ef_construction = 64)
	`)
	return err
}

// fetchStagingData queries staging_empresas + staging_simples for a batch of basicos.
func (imp *Importer) fetchStagingData(ctx context.Context, basicos []string) (map[string]empresaData, error) {
	rows, err := imp.pool.Query(ctx, `
		SELECT e.cnpj_basico,
		       e.razao_social,
		       e.natureza_juridica,
		       e.capital_social,
		       e.porte,
		       COALESCE(s.opcao_simples, false)
		FROM staging_empresas e
		LEFT JOIN staging_simples s USING (cnpj_basico)
		WHERE e.cnpj_basico = ANY($1)
	`, basicos)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]empresaData, len(basicos))
	for rows.Next() {
		var basico string
		var e empresaData
		if err := rows.Scan(&basico, &e.RazaoSocial, &e.NaturezaJuridica,
			&e.CapitalSocial, &e.Porte, &e.OpcaoSimples); err != nil {
			return nil, err
		}
		result[basico] = e
	}
	return result, rows.Err()
}

// sendBatch sends a pgx.Batch and drains all results, returning the first error.
func sendBatch(ctx context.Context, pool *pgxpool.Pool, batch *pgx.Batch) error {
	if batch.Len() == 0 {
		return nil
	}
	results := pool.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// nullStr returns nil for empty strings (maps to SQL NULL).
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullInt16 returns nil for zero values.
func nullInt16(v int16) *int16 {
	if v == 0 {
		return nil
	}
	return &v
}

// nullTime is a convenience used in the embedder.
func nullTime(t time.Time) *time.Time {
	return &t
}
