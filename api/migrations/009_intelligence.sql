-- +goose Up

CREATE TABLE tb_ai_qualifications (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  cnpj            VARCHAR(14) NOT NULL,
  org_id          UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id         UUID        NOT NULL REFERENCES tb_users(id),
  score           SMALLINT    NOT NULL CHECK (score BETWEEN 0 AND 100),
  justification   TEXT        NOT NULL,
  prompt_used     TEXT        NOT NULL,
  model_used      VARCHAR(50) NOT NULL,
  tokens_input    INT         NOT NULL,
  tokens_output   INT         NOT NULL,
  raw_response    JSONB       NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_qual_cnpj_org ON tb_ai_qualifications (cnpj, org_id, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_ai_qual_cnpj_org;
DROP TABLE IF EXISTS tb_ai_qualifications;
