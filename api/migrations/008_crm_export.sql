-- +goose Up

CREATE TABLE tb_crm_integrations (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID        NOT NULL REFERENCES tb_organizations(id) UNIQUE,
  crm_type      VARCHAR(30) NOT NULL DEFAULT 'chatwoot',
  base_url      TEXT        NOT NULL,
  api_key       TEXT        NOT NULL,    -- AES-256-GCM encrypted
  inbox_id      INT,
  extra_config  JSONB       NOT NULL DEFAULT '{}',
  is_active     BOOLEAN     NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tb_export_queue (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id         UUID        NOT NULL REFERENCES tb_users(id),
  search_id       UUID        REFERENCES tb_searches(id),
  cnpjs           TEXT[]      NOT NULL,
  crm_type        VARCHAR(30) NOT NULL DEFAULT 'chatwoot',
  status          VARCHAR(12) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','processing','done','partial','failed')),
  total_count     INT         NOT NULL,
  success_count   INT         NOT NULL DEFAULT 0,
  fail_count      INT         NOT NULL DEFAULT 0,
  error_log       JSONB       NOT NULL DEFAULT '[]',
  attempt         SMALLINT    NOT NULL DEFAULT 0,
  next_retry_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  done_at         TIMESTAMPTZ
);

CREATE INDEX idx_export_queue_org ON tb_export_queue (org_id, created_at DESC);
CREATE INDEX idx_export_queue_pending ON tb_export_queue (status, next_retry_at)
  WHERE status IN ('pending', 'failed') AND next_retry_at IS NOT NULL;

-- +goose Down

DROP TABLE IF EXISTS tb_export_queue;
DROP TABLE IF EXISTS tb_crm_integrations;
