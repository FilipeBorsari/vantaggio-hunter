# STEP-03 — Busca (Search Engine + Frontend)

## Objetivo
Implementar o motor de busca assíncrono com Redis queue e a interface completa de pesquisa — o core do produto, com dois modos: filtros estruturados e busca semântica por texto livre.

## Pré-requisitos
- STEP-02 concluído (auth, orgs, Lead Bank básico)
- STEP-01 concluído (embeddings gerados em `tb_companies`)
- `REDIS_URL` configurado no `.env`
- `OPENAI_API_KEY` (para gerar embedding da query semântica)

## Escopo
- ✅ Tabelas `tb_searches` e `tb_search_results`
- ✅ Redis queue para processamento assíncrono
- ✅ Worker Go que executa a busca e salva resultados
- ✅ Busca estruturada (filtros SQL) e semântica (pgvector)
- ✅ Frontend: `SearchPage` com dois modos de busca
- ✅ Histórico de buscas por org
- ❌ Dedução de créditos (STEP-04) — neste step a busca é livre
- ❌ Exportação de leads (STEP-06)

---

## 1. Banco de Dados

### Migration 004 — Buscas

```sql
-- migrations/004_searches.sql

CREATE TABLE tb_searches (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id     UUID        NOT NULL REFERENCES tb_users(id),
  mode        VARCHAR(12) NOT NULL CHECK (mode IN ('structured','semantic')),
  filters     JSONB       NOT NULL DEFAULT '{}',   -- filtros usados
  query_text  TEXT,                                -- texto da busca semântica
  status      VARCHAR(10) NOT NULL DEFAULT 'queued'
                          CHECK (status IN ('queued','processing','done','failed')),
  result_count INT,
  error_msg   TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  done_at     TIMESTAMPTZ
);

CREATE INDEX idx_searches_org_id ON tb_searches (org_id, created_at DESC);

CREATE TABLE tb_search_results (
  search_id   UUID        NOT NULL REFERENCES tb_searches(id) ON DELETE CASCADE,
  cnpj        VARCHAR(14) NOT NULL,
  score       FLOAT,      -- distância vetorial (NULL para busca estruturada)
  position    INT         NOT NULL,
  PRIMARY KEY (search_id, cnpj)
);

CREATE INDEX idx_search_results_search_id ON tb_search_results (search_id);
```

**Estrutura do JSONB `filters`:**
```json
{
  "cnaes":       ["4520-0/01", "4711-3/01"],
  "uf":          "SP",
  "city":        "São Paulo",
  "capital_min": 100000,
  "capital_max": null,
  "status":      2,
  "porte":       null
}
```

---

## 2. Backend (Go)

### 2.1 Endpoints

**POST /searches**
```
Header:  Authorization: Bearer {token}
Body:    {
           "mode": "structured" | "semantic",
           "filters": {                       // para mode=structured
             "cnaes": ["4520-0/01"],
             "uf": "SP",
             "city": "São Paulo",
             "capital_min": 100000,
             "status": 2
           },
           "query": "mecânicas de alto faturamento em SP"  // para mode=semantic
         }
201:     { "search_id": "uuid", "status": "queued" }
400:     se nem filters nem query fornecidos
```
- Criar registro em `tb_searches` com status `queued`
- Publicar `search_id` na fila Redis `queue:searches`
- Retornar imediatamente (não bloquear)

**GET /searches/:id**
```
Header:  Authorization: Bearer {token}
200 (queued/processing): { "id": "uuid", "status": "queued", "result_count": null }
200 (done):              {
                           "id": "uuid",
                           "status": "done",
                           "result_count": 842,
                           "results": [{
                             "cnpj": "...",
                             "razao_social": "...",
                             "municipio": "...",
                             "uf": "...",
                             "capital_social": 150000,
                             "situacao": 2,
                             "score": 0.87,   // null se structured
                             "cnaes": [...]
                           }],
                           "page": 1,
                           "limit": 100,
                           "total": 842
                         }
404:     se search_id não pertence à org do token
```
- Validar que `tb_searches.org_id` == `org_id` do JWT
- Suportar paginação: `?page=1&limit=100`

**GET /searches**
```
Header:  Authorization: Bearer {token}
Query:   ?page=1&limit=20
200:     {
           "data": [{
             "id": "uuid",
             "mode": "structured",
             "status": "done",
             "result_count": 842,
             "filters": {...},
             "query_text": null,
             "created_at": "..."
           }],
           "total": N
         }
```

### 2.2 Worker de Busca

Localização: `api/internal/searches/worker.go`

Roda como goroutine junto ao servidor (ou como processo separado via `api/cmd/worker/main.go`).

**Fluxo do worker:**
```
BLPOP queue:searches (blocking pop Redis)
        │
        ▼
  Buscar search em tb_searches
        │
        ├─ mode=structured
        │       ▼
        │  SQL: JOIN tb_company_cnaes + filtros WHERE
        │  INSERT INTO tb_search_results (search_id, cnpj, position)
        │
        └─ mode=semantic
                ▼
           Gerar embedding do query_text via OpenAI API
                ▼
           SELECT cnpj, embedding <=> $vec AS score
           FROM tb_companies
           WHERE (filtros opcionais se fornecidos)
           ORDER BY score ASC LIMIT 10000
                ▼
           INSERT INTO tb_search_results (search_id, cnpj, score, position)
        │
        ▼
  UPDATE tb_searches SET status='done', result_count=N, done_at=now()
  (em caso de erro: status='failed', error_msg=...)
```

**Query SQL para busca estruturada:**
```sql
SELECT DISTINCT c.cnpj
FROM tb_companies c
JOIN tb_company_cnaes cc ON cc.cnpj = c.cnpj
WHERE
  ($cnaes IS NULL OR cc.cnae_code = ANY($cnaes))
  AND ($uf IS NULL OR c.uf = $uf)
  AND ($city IS NULL OR c.municipio_nome ILIKE $city)
  AND ($capital_min IS NULL OR c.capital_social >= $capital_min)
  AND ($status IS NULL OR c.situacao_cadastral = $status)
ORDER BY c.cnpj
LIMIT 10000;
```

**Query SQL para busca semântica (híbrida):**
```sql
SELECT c.cnpj, c.embedding <=> $vec AS score
FROM tb_companies c
WHERE
  ($uf IS NULL OR c.uf = $uf)
  AND c.situacao_cadastral = 2          -- apenas ativas por padrão
  AND c.embedding IS NOT NULL
ORDER BY score ASC
LIMIT 10000;
```

---

## 3. Frontend (Next.js)

### Rotas

```
/search          ← nova busca
/search/:id      ← resultados de uma busca específica
/search/history  ← histórico de buscas
```

### 3.1 SearchPage (`/app/search/page.tsx`)

**Modo Estruturado (padrão):**
- Multi-select de CNAEs com autocomplete (buscar em `tb_cnaes` via endpoint `GET /cnaes?q=`)
- Select de UF (lista fixa dos 27 estados)
- Input de cidade (texto livre)
- Input de capital mínimo (numérico com máscara)
- Select de situação cadastral (Ativa / Inapta / Baixada)
- Botão "Buscar" → `POST /searches` → redirect para `/search/:id`

**Modo Semântico (toggle):**
- Campo de texto: "Descreva o perfil de empresa que você procura..."
- Placeholder: "Ex: mecânicas de alto faturamento em São Paulo com presença digital"
- Filtros opcionais de UF e status mesmo em modo semântico
- Botão "Buscar com IA"

### 3.2 SearchResultsPage (`/app/search/[id]/page.tsx`)

**Estado de loading (status = queued/processing):**
- Skeleton loader
- Polling automático a cada 2s via `GET /searches/:id`
- Mensagem: "Processando sua busca..."

**Estado de resultados (status = done):**
- Header: "{N} empresas encontradas"
- Tabela virtualizada (`@tanstack/react-virtual`) com colunas:
  - CNPJ (formatado: XX.XXX.XXX/XXXX-XX)
  - Razão Social
  - Município/UF
  - Capital Social (formatado: R$ X.XXX.XXX)
  - Situação (badge colorido: verde=ativa, vermelho=baixada)
  - Score de relevância (apenas modo semântico, de 0 a 100)
- Seleção de linhas (checkbox) para ações em lote (preparar para STEP-06)
- Paginação com `limit=100` por página
- Botão "Nova Busca"

### 3.3 Endpoint auxiliar de CNAEs

**GET /cnaes**
```
Query:   ?q=mecânica
200:     [{ "code": "4520-0/01", "description": "Manutenção e reparação..." }, ...]
```
- Para o autocomplete do campo CNAE no formulário de busca

### 3.4 SearchHistoryPage (`/app/search/history/page.tsx`)
- Tabela com buscas anteriores da org: data, modo, filtros resumidos, resultado, status
- Clicar em uma linha abre os resultados (`/search/:id`)

---

## 4. Integrações / Infra

- **Redis queue:** chave `queue:searches`, worker usa `BLPOP` com timeout de 5s para graceful shutdown
- **Concorrência:** iniciar N workers em goroutines (N configurável por env `SEARCH_WORKERS=4`)
- **Limite de resultados:** cap em 10.000 leads por busca para proteger a DB
- **Polling no frontend:** parar após 60s sem resposta (timeout de busca)

---

## 5. Testes e Validação

- [ ] `POST /searches` com mode=structured retorna `{ search_id, status: "queued" }` imediatamente
- [ ] Worker processa a busca e atualiza status para `done` em < 5s (para buscas pequenas)
- [ ] `GET /searches/:id` retorna resultados corretos após processamento
- [ ] Busca estruturada por CNAE `4520-0/01` + UF `SP` retorna apenas empresas do setor em SP
- [ ] Busca semântica por "padarias artesanais em Curitiba" retorna empresas semanticamente relevantes
- [ ] Usuário de org A não consegue ver resultados da busca da org B (retorna 404)
- [ ] Tabela de resultados com 5.000 linhas não trava o browser (virtualização funcionando)
- [ ] Polling para quando status = `done`
- [ ] Histórico de buscas mostra buscas anteriores da org em ordem decrescente

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Worker cai no meio da busca | Status fica como `processing`; implementar job de recovery que retorna `queued` buscas travadas há > 5min |
| Busca semântica lenta (sem índice HNSW) | Verificar se índice foi criado no STEP-01; criar manualmente se necessário |
| Muitas buscas simultâneas sobrecarregam DB | Worker pool com limite configurável (`SEARCH_WORKERS`); Redis age como buffer |
| Embedding da query falha (API offline) | Retornar erro gracioso com `status=failed` e mensagem para o usuário tentar novamente |
| Usuário clica "buscar" várias vezes | Debounce no frontend + desabilitar botão enquanto status=queued/processing |
