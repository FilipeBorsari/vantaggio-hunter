-- +goose Up

-- Migrar roles antes de aplicar a nova constraint
ALTER TABLE tb_users DROP CONSTRAINT IF EXISTS tb_users_role_check;

UPDATE tb_users SET role = 'super_admin' WHERE role = 'admin';
UPDATE tb_users SET role = 'org_admin'   WHERE role = 'manager';
UPDATE tb_users SET role = 'seller'      WHERE role = 'operator';

ALTER TABLE tb_users
  ADD CONSTRAINT tb_users_role_check CHECK (role IN ('super_admin', 'org_admin', 'seller'));

-- Display name for users (previously only email was stored)
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';

-- Seller credit limit (NULL = no limit, consumes freely from org balance)
ALTER TABLE tb_users ADD COLUMN IF NOT EXISTS credit_limit INT;

-- Invitation table (org_admin invites sellers by email)
CREATE TABLE tb_invitations (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID        NOT NULL REFERENCES tb_organizations(id),
  email        TEXT        NOT NULL,
  role         TEXT        NOT NULL DEFAULT 'seller',
  token        TEXT        NOT NULL UNIQUE,
  invited_by   UUID        NOT NULL REFERENCES tb_users(id),
  accepted_at  TIMESTAMPTZ,
  expires_at   TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT tb_invitations_role_check CHECK (role IN ('org_admin', 'seller'))
);

CREATE INDEX idx_invitations_token ON tb_invitations (token);
CREATE INDEX idx_invitations_org   ON tb_invitations (org_id);

-- Audit log table
CREATE TABLE tb_audit_logs (
  id         BIGSERIAL   PRIMARY KEY,
  org_id     UUID        REFERENCES tb_organizations(id),
  actor_id   UUID        NOT NULL REFERENCES tb_users(id),
  action     TEXT        NOT NULL,
  target_id  TEXT,
  metadata   JSONB       NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_org   ON tb_audit_logs (org_id, created_at DESC);
CREATE INDEX idx_audit_logs_actor ON tb_audit_logs (actor_id, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_audit_logs_actor;
DROP INDEX IF EXISTS idx_audit_logs_org;
DROP TABLE IF EXISTS tb_audit_logs;

DROP INDEX IF EXISTS idx_invitations_org;
DROP INDEX IF EXISTS idx_invitations_token;
DROP TABLE IF EXISTS tb_invitations;

ALTER TABLE tb_users DROP COLUMN IF EXISTS credit_limit;

UPDATE tb_users SET role = 'admin'    WHERE role = 'super_admin';
UPDATE tb_users SET role = 'manager'  WHERE role = 'org_admin';
UPDATE tb_users SET role = 'operator' WHERE role = 'seller';

ALTER TABLE tb_users DROP CONSTRAINT IF EXISTS tb_users_role_check;
ALTER TABLE tb_users
  ADD CONSTRAINT tb_users_role_check CHECK (role IN ('admin','manager','operator'));
