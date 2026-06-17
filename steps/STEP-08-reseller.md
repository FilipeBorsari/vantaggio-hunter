# STEP-08 — Reseller (Multi-Tenancy Completo)

## Objetivo
Implementar o modelo de revendedores — permitindo que parceiros criem e gerenciem seus próprios clientes, distribuam créditos e acompanhem métricas agregadas — viabilizando a escala via canal indireto.

## Pré-requisitos
- STEP-04 concluído (créditos implementados)
- STEP-02 concluído (orgs e usuários base)
- STEP-05 concluído (dashboard — reseller terá versão agregada)

## Escopo
- ✅ `tb_resellers` + vínculo com `tb_organizations`
- ✅ Role `reseller` no RBAC
- ✅ Reseller cria e gerencia orgs filhas
- ✅ Reseller distribui créditos entre suas orgs
- ✅ Dashboard agregado do reseller (métricas de todos os clientes)
- ✅ Admin gerencia resellers (criar, definir crédito master)
- ❌ Billing automático de resellers (pagamento entre reseller e Vantaggio — fora do escopo V1)
- ❌ White-label (domínio próprio, logo customizada — V2+)

---

## 1. Banco de Dados

### Migration 009 — Resellers

```sql
-- migrations/009_resellers.sql

CREATE TABLE tb_resellers (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID        NOT NULL UNIQUE REFERENCES tb_organizations(id),
  contact_email TEXT        NOT NULL,
  contact_name  TEXT        NOT NULL,
  is_active     BOOLEAN     NOT NULL DEFAULT true,
  credit_limit  INT         NOT NULL DEFAULT 0,   -- créditos que o admin alocou para este reseller
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Vincular org filha a um reseller
ALTER TABLE tb_organizations
  ADD COLUMN reseller_id UUID REFERENCES tb_resellers(id);

CREATE INDEX idx_organizations_reseller ON tb_organizations (reseller_id)
  WHERE reseller_id IS NOT NULL;

-- Usuários com role reseller
-- Adicionar 'reseller' ao CHECK de tb_users
ALTER TABLE tb_users
  DROP CONSTRAINT tb_users_role_check,
  ADD CONSTRAINT tb_users_role_check
    CHECK (role IN ('admin', 'reseller', 'manager', 'operator'));
```

**Hierarquia de créditos:**
```
Admin (Vantaggio)
  └─ Aloca créditos para tb_resellers.credit_limit
       └─ Reseller distribui créditos das suas orgs filhas
            └─ cada org filha tem seu próprio saldo em tb_credit_balances
```

O `credit_limit` do reseller não é um saldo diretamente — é o teto de quanto o reseller pode distribuir no total. O admin controla esse teto.

---

## 2. Backend (Go)

### 2.1 Endpoints Admin (apenas role: admin)

**POST /admin/resellers**
```
Body:    { "org_name": "Parceiro XYZ", "contact_email": "email@parceiro.com", "contact_name": "João Silva", "credit_limit": 50000 }
201:     {
           "reseller_id": "uuid",
           "org_id": "uuid",     // org master do reseller criada automaticamente
           "user_id": "uuid"     // usuário reseller criado com senha temporária
         }
```
- Criar `tb_organizations` para o reseller
- Criar `tb_resellers` vinculado
- Criar `tb_users` com role=reseller e senha temporária (enviar por email — ou retornar na resposta por ora)

**GET /admin/resellers**
```
200:     [{
           "reseller_id": "uuid",
           "org_name": "...",
           "contact_email": "...",
           "credit_limit": 50000,
           "credits_distributed": 30000,
           "org_count": 5,
           "is_active": true
         }]
```

**PATCH /admin/resellers/:id/credit-limit**
```
Body:    { "credit_limit": 100000 }
204:     sem body
```

### 2.2 Endpoints Reseller (role: reseller)

**POST /reseller/organizations**
```
Body:    {
           "name": "Cliente ABC",
           "plan_id": "uuid",
           "admin_email": "admin@clienteabc.com",
           "admin_password": "senha123"
         }
201:     { "org_id": "uuid", "user_id": "uuid" }
409:     se email já existe
422:     se reseller não tem crédito_limit suficiente para distribuir ao criar
```
- Criar org com `reseller_id` = ID do reseller autenticado
- Criar usuário admin da org filha

**POST /reseller/organizations/:orgId/credits**
```
Body:    { "amount": 1000, "description": "Plano mensal" }
201:     { "new_balance": 1000, "reseller_credits_used": 31000 }
422:     se amount > (credit_limit - credits_distributed)
```
- Registrar transação `type=purchase` em `tb_credit_transactions` da org filha
- Incrementar `credits_distributed` do reseller (para controle de teto)

**GET /reseller/organizations**
```
200:     [{
           "org_id": "uuid",
           "name": "Cliente ABC",
           "plan_name": "Pro",
           "balance": 450,
           "leads_this_month": 1200,
           "is_active": true
         }]
```

**GET /reseller/dashboard**
```
Query:   ?period=30d
200:     {
           "total_orgs": 5,
           "total_leads_extracted": 8400,
           "total_credits_distributed": 10000,
           "total_credits_consumed": 8400,
           "orgs": [{
             "org_id": "uuid",
             "name": "...",
             "leads_this_month": 1200,
             "credits_remaining": 450
           }]
         }
```

**PATCH /reseller/organizations/:orgId/users/:userId**
```
Body:    { "is_active": bool }
204:     sem body
```
*Reseller pode ativar/desativar usuários das suas orgs filhas.*

### 2.3 Isolamento de Dados

Middleware de autorização deve garantir:
- Reseller só acessa orgs onde `tb_organizations.reseller_id = reseller.id`
- Reseller não acessa dados de orgs de outros resellers
- Manager/Operator não veem outros clientes do mesmo reseller

---

## 3. Frontend (Next.js)

### Rotas

```
/reseller                     ← dashboard reseller (role=reseller)
/reseller/organizations       ← lista de clientes
/reseller/organizations/new   ← criar novo cliente
/reseller/organizations/:id   ← detalhe de um cliente
```

### 3.1 Reseller Dashboard (`/app/reseller/page.tsx`)

- Cards de KPI: total de clientes, leads gerados no mês, créditos distribuídos vs. usados
- Tabela de clientes com: nome, saldo, leads este mês, status
- Barra de progresso: "créditos distribuídos / credit_limit total"

### 3.2 Criar Cliente (`/app/reseller/organizations/new/page.tsx`)

- Formulário: nome da empresa, email do admin, senha temporária, plano, créditos iniciais
- Campo de créditos iniciais: mostrar saldo disponível do reseller para distribuir
- Submit: `POST /reseller/organizations` + `POST /reseller/organizations/:id/credits`

### 3.3 Detalhe de Cliente (`/app/reseller/organizations/[id]/page.tsx`)

- KPIs do cliente no período
- Histórico de créditos distribuídos
- Lista de usuários da org (com toggle ativar/desativar)
- Botão "Adicionar Créditos" → modal com valor

### 3.4 Sidebar condicional por role

- role=admin: mostra link "Resellers" (lista de todos os resellers)
- role=reseller: mostra links "Meus Clientes" e "Dashboard Reseller"
- role=manager/operator: não vê nada de reseller

---

## 4. Testes e Validação

- [ ] Admin cria reseller com credit_limit=10000 → org e usuário reseller criados
- [ ] Reseller loga e vê dashboard reseller; não vê menu de admin
- [ ] Reseller cria org filha "Cliente ABC" → org aparece em `GET /reseller/organizations`
- [ ] Reseller distribui 1000 créditos para "Cliente ABC" → balance da org = 1000
- [ ] Distribuir 10000 créditos (acima do credit_limit de 10000) → retorna 422
- [ ] Usuário de "Cliente ABC" faz busca → créditos debitados do saldo da própria org, não do reseller
- [ ] Reseller não consegue ver dados de orgs de outro reseller (retorna 404)
- [ ] Admin vê todos os resellers em `GET /admin/resellers` com totais corretos
- [ ] Dashboard reseller mostra soma de leads de todos os clientes no período

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Reseller excede credit_limit por race condition | `SELECT FOR UPDATE` em credit_limit ao distribuir, igual ao padrão de créditos |
| Reseller acessa dados de org de outro reseller | Middleware valida `reseller_id` em TODAS as rotas de reseller |
| Admin cria reseller com email já existente de usuário comum | Checar unicidade de email em tb_users antes de criar |
| Org filha fica sem admin se usuário for desativado | Validar que a org tem pelo menos um usuário ativo com role=manager antes de desativar |

---

## Verificação End-to-End (todos os steps)

Fluxo completo do produto funcionando:

1. **Admin** cria reseller "Parceiro XYZ" com credit_limit=50.000
2. **Reseller** cria cliente "Empresa Beta" e distribui 5.000 créditos
3. **Manager de Beta** configura integração Chatwoot
4. **Operator de Beta** usa CNAE Assistant → descobre CNAEs relevantes
5. **Operator** faz busca semântica "distribuidoras de alimentos em MG" → 800 leads
6. **Operator** qualifica 5 leads com IA → scores calculados
7. **Operator** exporta 50 leads para Chatwoot → aparecem como contatos
8. **Manager** acessa dashboard → vê KPIs com funil correto
9. **Reseller** acessa dashboard reseller → vê consumo da "Empresa Beta"
10. **Admin** acessa visão geral → todos os dados agregados corretos

Créditos ao final: 5000 - 800 (busca) - 50 (qualificação IA) - 50 (export) = **4.100 restantes**
