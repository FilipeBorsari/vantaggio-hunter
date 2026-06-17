# STEP-06 — CRM Export

## Objetivo
Implementar a exportação assíncrona de leads para o Chatwoot — fechando o loop do produto: buscar → qualificar → enviar para CRM.

## Pré-requisitos
- STEP-04 concluído (créditos integrados)
- STEP-03 concluído (resultados de busca disponíveis)
- Conta Chatwoot configurada com API Key

## Escopo
- ✅ Tabelas `tb_crm_integrations` e `tb_export_queue`
- ✅ Worker de export com retry e backoff
- ✅ Endpoint de criação de export
- ✅ Integração Chatwoot: criar contato + conversa
- ✅ Deduçao de 1 crédito por lead exportado com sucesso
- ✅ Frontend: botão de export na Lead Bank table + painel de status
- ❌ HubSpot, Pipedrive, RD Station (futuro)
- ❌ Enriquecimento de dados antes do export (STEP-07)

---

## 1. Banco de Dados

### Migration 007 — CRM Export

```sql
-- migrations/007_crm_export.sql

-- Integrações CRM por org (credenciais criptografadas)
CREATE TABLE tb_crm_integrations (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID        NOT NULL REFERENCES tb_organizations(id) UNIQUE,
  crm_type      VARCHAR(30) NOT NULL DEFAULT 'chatwoot',
  base_url      TEXT        NOT NULL,    -- ex: https://app.chatwoot.com
  api_key       TEXT        NOT NULL,    -- criptografado via AES-256-GCM
  inbox_id      INT,                     -- Chatwoot: inbox de destino
  extra_config  JSONB       NOT NULL DEFAULT '{}',
  is_active     BOOLEAN     NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Fila de exportação
CREATE TABLE tb_export_queue (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id         UUID        NOT NULL REFERENCES tb_users(id),
  search_id       UUID        REFERENCES tb_searches(id),
  cnpjs           TEXT[]      NOT NULL,   -- lista de CNPJs a exportar
  crm_type        VARCHAR(30) NOT NULL DEFAULT 'chatwoot',
  status          VARCHAR(12) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','processing','done','partial','failed')),
  total_count     INT         NOT NULL,
  success_count   INT         NOT NULL DEFAULT 0,
  fail_count      INT         NOT NULL DEFAULT 0,
  error_log       JSONB       NOT NULL DEFAULT '[]',  -- [{ cnpj, error, attempt }]
  attempt         SMALLINT    NOT NULL DEFAULT 0,
  next_retry_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  done_at         TIMESTAMPTZ
);

CREATE INDEX idx_export_queue_org ON tb_export_queue (org_id, created_at DESC);
CREATE INDEX idx_export_queue_pending ON tb_export_queue (status, next_retry_at)
  WHERE status IN ('pending', 'failed') AND next_retry_at IS NOT NULL;
```

**Criptografia de api_key:**
- Algoritmo: AES-256-GCM
- Chave derivada de `ENCRYPTION_KEY` (variável de ambiente — 32 bytes)
- Adicionar `ENCRYPTION_KEY` ao `.env.example`

---

## 2. Backend (Go)

### 2.1 Endpoints

**POST /crm/integrations** *(apenas manager ou admin)*
```
Body:    {
           "crm_type": "chatwoot",
           "base_url": "https://app.chatwoot.com",
           "api_key": "...",
           "inbox_id": 3
         }
201:     { "id": "uuid", "crm_type": "chatwoot", "is_active": true }
409:     se org já tem integração cadastrada
```
- Criptografar `api_key` antes de salvar

**GET /crm/integrations**
```
200:     { "id": "uuid", "crm_type": "chatwoot", "base_url": "...", "inbox_id": 3, "is_active": true }
         (api_key nunca retornada)
404:     se org não tem integração
```

**POST /exports**
```
Body:    {
           "cnpjs": ["12345678000195", "98765432000100"],
           "search_id": "uuid"   // opcional, para rastreabilidade
         }
201:     { "export_id": "uuid", "status": "pending", "total": 2 }
400:     se lista vazia ou > 500 CNPJs por vez
402:     se créditos insuficientes (verificar saldo antes de enfileirar)
404:     se org não tem integração CRM configurada
```
- Verificar saldo: `balance >= len(cnpjs) * 1`
- Criar registro em `tb_export_queue`
- Publicar `export_id` na fila Redis `queue:exports`

**GET /exports/:id**
```
200:     {
           "id": "uuid",
           "status": "done",
           "total_count": 50,
           "success_count": 48,
           "fail_count": 2,
           "error_log": [{ "cnpj": "...", "error": "contato duplicado" }],
           "created_at": "...",
           "done_at": "..."
         }
```

**GET /exports**
```
Query:   ?page=1&limit=20
200:     { "data": [...], "total": N }
```

### 2.2 Worker de Export

Localização: `api/internal/exports/worker.go`

**Fluxo:**
```
BLPOP queue:exports
        │
        ▼
  Buscar tb_export_queue por ID
  Marcar status='processing', attempt++
        │
        ▼
  Para cada CNPJ no array:
    1. Buscar dados da empresa em tb_companies + tb_company_cnaes
    2. Chamar Chatwoot API (ver seção 2.3)
    3. Se sucesso: success_count++, debitar 1 crédito
    4. Se erro: registrar em error_log, fail_count++
        │
        ▼
  Se fail_count > 0 e attempt < 3:
    → status='failed', next_retry_at = now() + backoff(attempt)
    → Recolocar na fila Redis com delay
  Se fail_count = 0:
    → status='done'
  Se success_count > 0 e fail_count > 0:
    → status='partial'
        │
        ▼
  Atualizar tb_export_queue com resultado final
  Atualizar fato_funil_leads.leads_exportados (para dashboard)
```

**Backoff exponencial:**
- attempt 1: retry após 2min
- attempt 2: retry após 10min
- attempt 3: falha definitiva

### 2.3 Integração Chatwoot

Localização: `api/internal/exports/chatwoot.go`

**Criar ou localizar contato:**
```http
POST {base_url}/api/v1/accounts/{account_id}/contacts/search
  { "q": "{email ou telefone}" }

Se não encontrar:
POST {base_url}/api/v1/accounts/{account_id}/contacts
  {
    "name": "{razao_social}",
    "phone_number": "{ddd+telefone}",
    "email": "{email}",
    "additional_attributes": {
      "cnpj": "...",
      "municipio": "...",
      "uf": "...",
      "cnae": "...",
      "capital_social": "..."
    }
  }
```

**Criar conversa:**
```http
POST {base_url}/api/v1/accounts/{account_id}/conversations
  {
    "inbox_id": {inbox_id},
    "contact_id": {contact_id},
    "additional_attributes": {
      "origem": "Vantaggio PrHunter",
      "search_id": "..."
    }
  }
```

---

## 3. Frontend (Next.js)

### 3.1 Botão de Export na Lead Bank (`/app/search/[id]/page.tsx`)

- Checkbox por linha na tabela de resultados
- "Selecionar tudo" na página atual
- Barra de ações ao selecionar linhas:
  - "Exportar para CRM (N selecionados)" → abre modal de confirmação
  - Modal: resumo do custo (N créditos), saldo disponível, botão confirmar
  - Ao confirmar: `POST /exports` → toast "Export iniciado"

### 3.2 Página de Exports (`/app/exports/page.tsx`)

- Tabela com: data, total enviados, sucesso, falhas, status (badge)
- Expandir linha para ver `error_log` detalhado
- Polling automático para exports com status `pending` ou `processing`

### 3.3 Página de Configuração CRM (`/app/settings/crm/page.tsx`)

- Formulário para configurar integração Chatwoot
- Campos: URL base, API Key, Inbox ID
- Botão "Testar conexão" (`GET /crm/integrations/test`) → verificar se API Key é válida
- Status visual: integração ativa/inativa

---

## 4. Integrações / Infra

- `ENCRYPTION_KEY=<32 bytes base64>` no `.env`
- Redis queue: `queue:exports` com `BLPOP` no worker
- Fila de retry: usar `ZADD queue:exports:delayed <timestamp> <export_id>` + job de scheduler que move para a fila principal quando o tempo chega

---

## 5. Testes e Validação

- [ ] Configurar integração Chatwoot via `POST /crm/integrations`
- [ ] Exportar 5 CNPJs selecionados → verificar que aparecem como contatos no Chatwoot
- [ ] Verificar que 5 créditos foram debitados após export bem-sucedido
- [ ] Simular falha de rede → verificar que status=failed e retry é agendado
- [ ] Após 3 tentativas com falha → status=partial com error_log preenchido
- [ ] Exportar CNPJ que já existe no Chatwoot → não criar duplicata (usar search antes de criar)
- [ ] `GET /exports` lista o export com status correto
- [ ] Org sem integração CRM → `POST /exports` retorna 404
- [ ] Saldo insuficiente → `POST /exports` retorna 402 sem enfileirar nada
- [ ] Dashboard: `leads_exportados` no funil reflete os exports bem-sucedidos

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| API Key Chatwoot armazenada em texto puro | AES-256-GCM obrigatório; ENCRYPTION_KEY rotacionável |
| Duplicatas de contatos no CRM | Sempre fazer search por telefone/email antes de criar |
| Worker processa export mas créditos não debitados por crash | Débito por CNPJ individual dentro de transação; export parcial tem créditos apenas para os bem-sucedidos |
| Chatwoot fora do ar → retry infinito | Limite de 3 tentativas + status=failed definitivo após esgotamento |
| Export de 500 CNPJs bloqueia worker por muito tempo | Worker processa em chunks de 50; yielda entre chunks |
