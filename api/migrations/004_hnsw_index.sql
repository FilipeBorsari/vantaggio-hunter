-- +goose Up

-- Desabilita workers paralelos para evitar problema de shared memory no Docker.
-- Em produção, ajuste max_parallel_maintenance_workers conforme recursos disponíveis.
SET max_parallel_maintenance_workers = 0;

CREATE INDEX IF NOT EXISTS idx_companies_embedding
    ON tb_companies
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

RESET max_parallel_maintenance_workers;

-- +goose Down

DROP INDEX IF EXISTS idx_companies_embedding;
