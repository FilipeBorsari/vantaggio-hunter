-- +goose Up

CREATE TABLE tb_plans (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(50)  NOT NULL,
  credits     INT          NOT NULL,
  price_cents INT          NOT NULL,
  active      BOOLEAN      NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE tb_organizations (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name         VARCHAR(255) NOT NULL,
  plan_id      UUID         REFERENCES tb_plans(id),
  is_active    BOOLEAN      NOT NULL DEFAULT true,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE tb_users (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID         NOT NULL REFERENCES tb_organizations(id),
  email         VARCHAR(255) NOT NULL UNIQUE,
  password_hash TEXT         NOT NULL,
  role          VARCHAR(20)  NOT NULL CHECK (role IN ('admin','manager','operator')),
  is_active     BOOLEAN      NOT NULL DEFAULT true,
  deleted_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_org_id ON tb_users (org_id);
CREATE INDEX idx_users_email  ON tb_users (email);

CREATE TABLE tb_refresh_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
  token_hash  TEXT        NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON tb_refresh_tokens (user_id);

INSERT INTO tb_plans (name, credits, price_cents) VALUES
  ('Starter',    1000,   9900),
  ('Pro',        10000,  49900),
  ('Enterprise', 100000, 199900);

INSERT INTO tb_organizations (id, name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Vantaggio');

-- senha: admin123 (bcrypt custo 12) — TROCAR após primeiro deploy
INSERT INTO tb_users (org_id, email, password_hash, role) VALUES
  ('00000000-0000-0000-0000-000000000001',
   'admin@vantaggio.com.br',
   '$2a$12$pUhnWiOoNK5O8nvg6kOr2OQXh8ucZlcoWLXKdKNF6NtXGPczgBG5u',
   'admin');

-- +goose Down

DROP TABLE IF EXISTS tb_refresh_tokens;
DROP TABLE IF EXISTS tb_users;
DROP TABLE IF EXISTS tb_organizations;
DROP TABLE IF EXISTS tb_plans;
