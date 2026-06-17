-- +goose Up

-- Índices estruturados em tb_companies
CREATE INDEX IF NOT EXISTS idx_companies_uf        ON tb_companies (uf);
CREATE INDEX IF NOT EXISTS idx_companies_municipio ON tb_companies (municipio_id);
CREATE INDEX IF NOT EXISTS idx_companies_situacao  ON tb_companies (situacao_cadastral);
CREATE INDEX IF NOT EXISTS idx_companies_capital   ON tb_companies (capital_social);
CREATE INDEX IF NOT EXISTS idx_companies_porte     ON tb_companies (porte);
CREATE INDEX IF NOT EXISTS idx_companies_simples   ON tb_companies (opcao_simples);

-- Índices para tb_company_cnaes
CREATE INDEX IF NOT EXISTS idx_company_cnaes_code ON tb_company_cnaes (cnae_code);
CREATE INDEX IF NOT EXISTS idx_company_cnaes_cnpj ON tb_company_cnaes (cnpj);

-- Índice em tb_partners para lookup por empresa
CREATE INDEX IF NOT EXISTS idx_partners_cnpj_basico ON tb_partners (cnpj_basico);

-- Nota: o índice HNSW em tb_companies.embedding deve ser criado DEPOIS da ingestão completa
-- e da geração de embeddings. O CLI de ingestão faz isso automaticamente ao final.
-- Para criar manualmente:
--
-- CREATE INDEX idx_companies_embedding ON tb_companies
--     USING hnsw (embedding vector_cosine_ops)
--     WITH (m = 16, ef_construction = 64);

-- +goose Down

DROP INDEX IF EXISTS idx_partners_cnpj_basico;
DROP INDEX IF EXISTS idx_company_cnaes_cnpj;
DROP INDEX IF EXISTS idx_company_cnaes_code;
DROP INDEX IF EXISTS idx_companies_simples;
DROP INDEX IF EXISTS idx_companies_porte;
DROP INDEX IF EXISTS idx_companies_capital;
DROP INDEX IF EXISTS idx_companies_situacao;
DROP INDEX IF EXISTS idx_companies_municipio;
DROP INDEX IF EXISTS idx_companies_uf;
