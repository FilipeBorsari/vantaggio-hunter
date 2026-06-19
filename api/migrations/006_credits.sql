-- +goose Up

CREATE TABLE tb_credit_transactions (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id      UUID        REFERENCES tb_users(id),
  type         VARCHAR(20) NOT NULL
               CHECK (type IN (
                 'purchase',
                 'search',
                 'company_detail',
                 'enrichment',
                 'export',
                 'adjustment'
               )),
  amount       INT         NOT NULL,
  description  TEXT,
  reference_id UUID,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_credit_transactions_org ON tb_credit_transactions (org_id, created_at DESC);

CREATE TABLE tb_credit_balances (
  org_id     UUID PRIMARY KEY REFERENCES tb_organizations(id),
  balance    INT  NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tb_credit_balances (org_id)
SELECT id FROM tb_organizations;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_credit_balance()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO tb_credit_balances (org_id, balance, updated_at)
  VALUES (NEW.org_id, NEW.amount, now())
  ON CONFLICT (org_id) DO UPDATE
  SET balance    = tb_credit_balances.balance + NEW.amount,
      updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_update_balance
AFTER INSERT ON tb_credit_transactions
FOR EACH ROW EXECUTE FUNCTION update_credit_balance();

-- +goose Down

DROP TRIGGER IF EXISTS trg_update_balance ON tb_credit_transactions;
DROP FUNCTION IF EXISTS update_credit_balance();
DROP TABLE IF EXISTS tb_credit_balances;
DROP TABLE IF EXISTS tb_credit_transactions;
