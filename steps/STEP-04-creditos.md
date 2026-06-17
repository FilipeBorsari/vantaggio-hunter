# STEP-04 — Créditos (MVP Completo)

## Objetivo
Implementar o sistema de créditos com ledger append-only e deduções atômicas — após este step o produto está monetizável e pronto para os primeiros clientes.

## Pré-requisitos
- STEP-03 concluído (buscas funcionando sem créditos)
- STEP-02 concluído (orgs e planos criados)

## Escopo
- ✅ Tabela `tb_credit_transactions` (ledger imutável)
- ✅ Middleware `requireCredits(n)` com deduções atômicas
- ✅ Integração com `/searches` (1 crédito por lead retornado)
- ✅ Integração com `GET /companies/:cnpj` (10 créditos — consulta individual)
- ✅ Endpoints de saldo e histórico
- ✅ Frontend: saldo no topbar + bloqueio gracioso
- ❌ Gateway de pagamento (Stripe/Abacate) — admin distribui créditos manualmente por ora
- ❌ Enriquecimento e export (virão nos STEP-06 e STEP-07)

---

## 1. Banco de Dados

### Migration 005 — Créditos

```sql
-- migrations/005_credits.sql

CREATE TABLE tb_credit_transactions (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id         UUID        REFERENCES tb_users(id),  -- NULL para transações do sistema
  type            VARCHAR(20) NOT NULL
                  CHECK (type IN (
                    'purchase',       -- admin adiciona créditos
                    'search',         -- consumo por busca
                    'company_detail', -- consulta individual (10 créditos)
                    'enrichment',     -- enriquecimento (STEP-07)
                    'export',         -- exportação CRM (STEP-06)
                    'adjustment'      -- correção manual pelo admin
                  )),
  amount          INT         NOT NULL,  -- positivo=entrada, negativo=saída
  description     TEXT,
  reference_id    UUID,                  -- search_id, export_id, etc. para rastreabilidade
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Índices para cálculo de saldo eficiente
CREATE INDEX idx_credit_transactions_org ON tb_credit_transactions (org_id, created_at DESC);

-- View materializada para saldo por org (atualizada via trigger)
CREATE TABLE tb_credit_balances (
  org_id   UUID PRIMARY KEY REFERENCES tb_organizations(id),
  balance  INT  NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Inserir saldo inicial zerado para cada org existente
INSERT INTO tb_credit_balances (org_id) SELECT id FROM tb_organizations;

-- Trigger para manter tb_credit_balances sincronizado
CREATE OR REPLACE FUNCTION update_credit_balance()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO tb_credit_balances (org_id, balance, updated_at)
  VALUES (NEW.org_id, NEW.amount, now())
  ON CONFLICT (org_id) DO UPDATE
  SET balance = tb_credit_balances.balance + NEW.amount,
      updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_balance
AFTER INSERT ON tb_credit_transactions
FOR EACH ROW EXECUTE FUNCTION update_credit_balance();
```

> **Por que `tb_credit_balances`?** Calcular `SUM(amount)` em milhões de transações a cada request seria lento. A tabela de saldo é uma view materializada mantida por trigger — leitura O(1) e escrita transacional segura.

---

## 2. Backend (Go)

### 2.1 Pacote de créditos

Localização: `api/internal/credits/`

```go
// service.go — interface pública do pacote

// GetBalance retorna o saldo atual da org
func (s *Service) GetBalance(ctx context.Context, orgID uuid.UUID) (int, error)

// Deduct debita N créditos atomicamente.
// Retorna ErrInsufficientCredits se saldo < amount.
// Usa SELECT FOR UPDATE em tb_credit_balances para evitar race condition.
func (s *Service) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID uuid.UUID, amount int, txType string, refID *uuid.UUID, desc string) error

// AddCredits adiciona créditos (uso admin)
func (s *Service) AddCredits(ctx context.Context, orgID uuid.UUID, amount int, desc string) error

// ListTransactions retorna histórico paginado
func (s *Service) ListTransactions(ctx context.Context, orgID uuid.UUID, page, limit int) ([]Transaction, int, error)
```

**Fluxo de deduçao atômica com SELECT FOR UPDATE:**
```go
// Dentro de uma transação Postgres:
// 1. SELECT balance FROM tb_credit_balances WHERE org_id = $1 FOR UPDATE
// 2. Se balance < amount → rollback → retornar ErrInsufficientCredits
// 3. INSERT INTO tb_credit_transactions (amount = -N, ...)
//    (o trigger atualiza tb_credit_balances automaticamente)
// 4. COMMIT
```

### 2.2 Endpoints de Créditos

**GET /credits/balance**
```
Header:  Authorization: Bearer {token}
200:     { "balance": 450, "org_id": "uuid" }
```

**GET /credits/transactions**
```
Query:   ?page=1&limit=20
200:     {
           "data": [{
             "id": "uuid",
             "type": "search",
             "amount": -100,
             "description": "Busca: 100 leads retornados",
             "created_at": "..."
           }],
           "total": N,
           "balance": 450
         }
```

**POST /admin/credits/add** *(apenas admin)*
```
Body:    { "org_id": "uuid", "amount": 1000, "description": "Compra plano Pro" }
204:     sem body
```

### 2.3 Integração com Buscas

Modificar o worker de busca (`STEP-03`) para:

1. **Antes de processar:** verificar saldo mínimo (ex: pelo menos 1 crédito)
2. **Após obter resultados:** deduzir `result_count * 1` crédito
3. Se `Deduct` falhar por saldo insuficiente: definir `status=failed`, `error_msg="Créditos insuficientes"`

```go
// No worker, após calcular result_count:
err := creditSvc.Deduct(ctx, tx, orgID, userID,
    resultCount * 1,
    "search",
    &search.ID,
    fmt.Sprintf("Busca: %d leads retornados", resultCount),
)
```

### 2.4 Integração com GET /companies/:cnpj

```go
// No handler de detalhe de empresa:
err := creditSvc.Deduct(ctx, tx, orgID, userID,
    10,
    "company_detail",
    nil,
    fmt.Sprintf("Consulta CNPJ: %s", cnpj),
)
// Se ErrInsufficientCredits → retornar 402 Payment Required
```

---

## 3. Frontend (Next.js)

### 3.1 Saldo no Topbar

- Componente `<CreditBalance />` na Topbar
- Buscar `GET /credits/balance` no mount + refetch a cada 30s
- Exibir: "💳 450 créditos"
- Se saldo < 100: exibir em laranja com ícone de alerta
- Se saldo = 0: exibir em vermelho

### 3.2 Bloqueio Gracioso

Quando uma ação falha com erro de créditos insuficientes (HTTP 402 ou mensagem no `status=failed`):
- Toast de erro: "Créditos insuficientes. Entre em contato com o administrador."
- Não redirecionar; manter o usuário na página atual

### 3.3 Página de Créditos (`/app/credits/page.tsx`)

- Saldo atual destacado em destaque
- Tabela de transações com tipo, valor (+/-), descrição e data
- Paginação

---

## 4. Testes e Validação

- [ ] Admin adiciona 500 créditos para org via `POST /admin/credits/add`
- [ ] `GET /credits/balance` retorna 500
- [ ] Busca retorna 100 leads → saldo cai para 400
- [ ] `GET /credits/transactions` mostra a transação de -100 com `reference_id` = search_id
- [ ] Saldo chega a 0 → próxima busca falha com status=failed e mensagem de créditos insuficientes
- [ ] **Teste de concorrência:** 2 requests simultâneos tentando usar os últimos 50 créditos (org com saldo=50, cada request pede 50) → apenas 1 sucede; saldo final = 0, nunca negativo
- [ ] `GET /companies/:cnpj` deduz 10 créditos e retorna os dados
- [ ] Topbar exibe saldo atualizado após busca
- [ ] Usuário operator não consegue acessar `POST /admin/credits/add`

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Race condition → saldo negativo | `SELECT FOR UPDATE` garante exclusividade; testado com concorrência |
| Trigger falha silenciosamente | Monitorar `tb_credit_balances` vs `SUM(amount)` periodicamente via query de reconciliação |
| Admin esquece de adicionar créditos | Notificação por email quando saldo < 10% do plano (implementar no futuro) |
| Deduçao ocorre mas busca falha | A transação de crédito é parte da mesma transação Postgres; rollback automático em caso de erro |
