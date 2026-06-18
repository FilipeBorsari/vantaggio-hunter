-- +goose Up

-- ========================
-- ETL CONTROL COLUMNS
-- ========================

ALTER TABLE tb_credit_transactions ADD COLUMN IF NOT EXISTS etl_processed_at TIMESTAMPTZ;
ALTER TABLE tb_searches ADD COLUMN IF NOT EXISTS etl_processed_at TIMESTAMPTZ;

-- ========================
-- DIMENSÕES
-- ========================

CREATE TABLE dim_tempo (
  id            INT       PRIMARY KEY,   -- formato: YYYYMMDD
  data          DATE      NOT NULL UNIQUE,
  dia           SMALLINT  NOT NULL,
  mes           SMALLINT  NOT NULL,
  ano           SMALLINT  NOT NULL,
  trimestre     SMALLINT  NOT NULL,
  dia_semana    SMALLINT  NOT NULL,      -- 0=domingo, 6=sábado
  semana_ano    SMALLINT  NOT NULL,
  is_fim_semana BOOLEAN   NOT NULL
);

INSERT INTO dim_tempo (id, data, dia, mes, ano, trimestre, dia_semana, semana_ano, is_fim_semana)
SELECT
  TO_CHAR(d, 'YYYYMMDD')::INT,
  d::DATE,
  EXTRACT(DAY    FROM d)::SMALLINT,
  EXTRACT(MONTH  FROM d)::SMALLINT,
  EXTRACT(YEAR   FROM d)::SMALLINT,
  EXTRACT(QUARTER FROM d)::SMALLINT,
  EXTRACT(DOW    FROM d)::SMALLINT,
  EXTRACT(WEEK   FROM d)::SMALLINT,
  EXTRACT(DOW    FROM d) IN (0, 6)
FROM generate_series('2024-01-01'::date, '2030-12-31'::date, '1 day'::interval) d;

CREATE TABLE dim_organizacao (
  id        UUID    PRIMARY KEY,
  name      TEXT    NOT NULL,
  plan_name TEXT,
  is_active BOOLEAN NOT NULL
);

CREATE TABLE dim_usuario (
  id     UUID PRIMARY KEY,
  email  TEXT NOT NULL,
  role   TEXT NOT NULL,
  org_id UUID NOT NULL
);

CREATE TABLE dim_cnae (
  code        VARCHAR(10) PRIMARY KEY,
  description TEXT        NOT NULL,
  secao       VARCHAR(5),
  divisao     VARCHAR(5)
);

CREATE TABLE dim_geografia (
  id             SERIAL      PRIMARY KEY,
  municipio_nome TEXT        NOT NULL,
  uf             CHAR(2)     NOT NULL,
  regiao         VARCHAR(20),
  UNIQUE (municipio_nome, uf)
);

-- ========================
-- FATOS
-- ========================

CREATE TABLE fato_consumo_creditos (
  id         BIGSERIAL   PRIMARY KEY,
  tempo_id   INT         NOT NULL REFERENCES dim_tempo(id),
  org_id     UUID        NOT NULL REFERENCES dim_organizacao(id),
  usuario_id UUID        REFERENCES dim_usuario(id),
  tipo       VARCHAR(20) NOT NULL,
  creditos   INT         NOT NULL,
  eh_entrada BOOLEAN     NOT NULL
);

CREATE INDEX idx_fato_consumo_org_tempo ON fato_consumo_creditos (org_id, tempo_id);

CREATE TABLE fato_funil_leads (
  id                 BIGSERIAL   PRIMARY KEY,
  tempo_id           INT         NOT NULL REFERENCES dim_tempo(id),
  org_id             UUID        NOT NULL REFERENCES dim_organizacao(id),
  usuario_id         UUID        REFERENCES dim_usuario(id),
  cnae_id            VARCHAR(10) REFERENCES dim_cnae(code),
  geo_id             INT         REFERENCES dim_geografia(id),
  leads_extraidos    INT         NOT NULL DEFAULT 0,
  leads_qualificados INT         NOT NULL DEFAULT 0,
  leads_exportados   INT         NOT NULL DEFAULT 0,
  search_id          UUID        NOT NULL
);

CREATE INDEX idx_fato_funil_org_tempo ON fato_funil_leads (org_id, tempo_id);

-- +goose Down

DROP TABLE IF EXISTS fato_funil_leads;
DROP TABLE IF EXISTS fato_consumo_creditos;
DROP TABLE IF EXISTS dim_geografia;
DROP TABLE IF EXISTS dim_cnae;
DROP TABLE IF EXISTS dim_usuario;
DROP TABLE IF EXISTS dim_organizacao;
DROP TABLE IF EXISTS dim_tempo;

ALTER TABLE tb_searches DROP COLUMN IF EXISTS etl_processed_at;
ALTER TABLE tb_credit_transactions DROP COLUMN IF EXISTS etl_processed_at;
