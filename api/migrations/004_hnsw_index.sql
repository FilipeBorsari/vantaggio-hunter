-- +goose Up

-- Aumenta a memória de manutenção para construção eficiente do índice HNSW.
-- Execute esta migration APÓS a ingestão completa e a geração de embeddings;
-- rodá-la com a tabela vazia não causa erro, mas o índice não terá utilidade.
SET maintenance_work_mem = '2GB';

CREATE INDEX IF NOT EXISTS idx_companies_embedding
    ON tb_companies
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

RESET maintenance_work_mem;

-- +goose Down

DROP INDEX IF EXISTS idx_companies_embedding;
