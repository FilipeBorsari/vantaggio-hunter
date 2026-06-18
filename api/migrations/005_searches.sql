-- +goose Up

CREATE TABLE tb_searches (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id      UUID        NOT NULL REFERENCES tb_users(id),
  mode         VARCHAR(12) NOT NULL CHECK (mode IN ('structured','semantic')),
  filters      JSONB       NOT NULL DEFAULT '{}',
  query_text   TEXT,
  status       VARCHAR(10) NOT NULL DEFAULT 'queued'
                           CHECK (status IN ('queued','processing','done','failed')),
  result_count INT,
  error_msg    TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  done_at      TIMESTAMPTZ
);

CREATE INDEX idx_searches_org_id ON tb_searches (org_id, created_at DESC);

CREATE TABLE tb_search_results (
  search_id UUID        NOT NULL REFERENCES tb_searches(id) ON DELETE CASCADE,
  cnpj      VARCHAR(14) NOT NULL,
  score     FLOAT,
  position  INT         NOT NULL,
  PRIMARY KEY (search_id, cnpj)
);

CREATE INDEX idx_search_results_search_id ON tb_search_results (search_id);

-- +goose Down

DROP TABLE IF EXISTS tb_search_results;
DROP TABLE IF EXISTS tb_searches;
