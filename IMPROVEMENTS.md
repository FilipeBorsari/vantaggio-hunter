# IMPROVEMENTS.md — Ajustes de qualidade de código

Checklist vivo de violações encontradas na auditoria de 2026-06-17. Marque `[x]` conforme corrigido e anote o commit.

---

## HIGH — Bloqueadores de testabilidade e corretude

### H1 — Services acoplados ao pool de banco diretamente

**Problema:** Services recebem `*pgxpool.Pool` em vez de uma interface de repository. Impossível testar com mock.

**Arquivos afetados:**
- [api/internal/auth/service.go:43](api/internal/auth/service.go)
- [api/internal/companies/service.go:77](api/internal/companies/service.go)
- [api/internal/admin/service.go:44](api/internal/admin/service.go)

**O que fazer:**
1. Criar `internal/domain/` com as structs compartilhadas (`Company`, `User`, `Org`, `Plan`)
2. Criar `repository.go` em cada package com a interface (`type Repository interface { ... }`)
3. Criar `repository_postgres.go` com a implementação concreta recebendo `*pgxpool.Pool`
4. Alterar `Service` para receber `Repository` (interface) no construtor
5. Atualizar `main.go` para instanciar `NewPostgresRepository(pool)` e injetar

- [x] `internal/auth`: extrair repository interface + impl postgres
- [x] `internal/companies`: extrair repository interface + impl postgres
- [x] `internal/admin`: extrair repository interface + impl postgres
- [x] `internal/domain/`: criar package com entidades compartilhadas

---

### H2 — Zero interfaces definidas (código 100% não testável)

**Problema:** Nenhuma interface existe para services nem repositories. Handlers recebem structs concretas. Impede mocks e testes unitários.

**Arquivos afetados:** Todo `api/internal/`

**O que fazer:**
1. Para cada `Service`, definir `type ServiceInterface interface` com os métodos públicos
2. Handlers devem receber a interface, não a struct concreta
3. Seguir exemplo em `CLAUDE.md` (seção Clean Architecture)

- [x] Interface para `auth.Service`
- [x] Interface para `companies.Service`
- [x] Interface para `admin.Service`

---

### H3 — Erros silenciados sistematicamente

**Problema:** Oito ou mais locais ignoram erros com `//nolint:errcheck`, `_, _ =`, ou `_ =`. Falhas passam despercebidas em produção.

**Arquivos e linhas:**

| Arquivo | Linha | Problema |
|---|---|---|
| [api/cmd/server/main.go:40](api/cmd/server/main.go) | 40 | `w.Write([]byte(...)) //nolint:errcheck` |
| [api/internal/auth/handler.go:58](api/internal/auth/handler.go) | 58 | `json.NewEncoder(w).Encode(v) //nolint:errcheck` |
| [api/internal/auth/handler.go:50-51](api/internal/auth/handler.go) | 50-51 | Ignora erro do Logout silenciosamente |
| [api/internal/auth/service.go:86](api/internal/auth/service.go) | 86 | `_, _ = s.db.Exec(ctx, ...)` |
| [api/internal/auth/service.go:59](api/internal/auth/service.go) | 59 | `errors.Is(err, pgx.ErrNoRows) \|\| err != nil` — condição engole todo erro como "not found" |
| [api/internal/companies/handler.go:70](api/internal/companies/handler.go) | 70 | `json.NewEncoder(w).Encode(v) //nolint:errcheck` |
| [api/internal/companies/service.go:124](api/internal/companies/service.go) | 124 | `_ = s.db.QueryRow(...)` — falha de count silenciada |
| [api/internal/companies/service.go:200,219](api/internal/companies/service.go) | 200, 219 | `if err == nil { defer... }` — path de erro não tratado |
| [api/internal/companies/service.go:256](api/internal/companies/service.go) | 256 | Retorna `nil` após falha de query |
| [api/internal/admin/handler.go:102](api/internal/admin/handler.go) | 102 | `json.NewEncoder(w).Encode(v) //nolint:errcheck` |
| [api/internal/admin/service.go:141](api/internal/admin/service.go) | 141 | `_ = s.db.QueryRow(...).Scan(&total)` — falha de count silenciada |

**O que fazer:** Para cada linha, substituir `//nolint:errcheck` e `_ =` por tratamento real. Ver padrão em `CLAUDE.md` (seção Error handling).

- [x] `cmd/server/main.go:40` — health endpoint usa httputil.JSON; sem write silenciado
- [x] `auth/handler.go:58` — substituído por httputil.JSON com logging interno
- [x] `auth/handler.go:50-51` — Logout agora retorna 500 em caso de erro
- [x] `auth/service.go:86` — DeleteRefreshToken via repo; erro logado com slog.Warn
- [x] `auth/service.go:59` — `ErrNoRows` tratado separadamente; DB errors propagados
- [x] `companies/handler.go:70` — substituído por httputil.JSON
- [x] `companies/service.go:124` — Count() na repo retorna erro; service propaga
- [x] `companies/service.go:200,219` — GetCNAEsByCNPJ/GetPartnersByCNPJBasico propagam erro
- [x] `companies/service.go:256` — attachCNAEs loga warning; CNAEs opcionais (não falha listing)
- [x] `admin/handler.go:102` — substituído por httputil.JSON
- [x] `admin/service.go:141` — CountOrgs() na repo retorna erro; service propaga

---

### H4 — Zero arquivos de teste

**Problema:** Nenhum `*_test.go` no projeto inteiro. Qualquer refactor é cego.

**O que fazer:**
1. Após resolver H1 e H2 (interfaces + repositories), criar testes unitários para services com mocks
2. Criar testes de integração para repositories com banco real (usar `testcontainers-go` ou banco Docker dedicado)
3. Criar testes de handler com `httptest.NewRecorder`

- [ ] `auth/service_test.go` (Login, Refresh, Logout)
- [ ] `companies/service_test.go` (List, GetByCNPJ, busca com filtros)
- [ ] `admin/service_test.go` (CreateOrg, CreateUser, ListPlans)
- [ ] `auth/repository_postgres_test.go`
- [ ] `companies/repository_postgres_test.go`
- [ ] Configurar `make test` no Makefile

---

## MEDIUM — Qualidade e manutenibilidade

### M1 — Helper `writeJSON` duplicado em 3 packages

**Problema:** Mesmo código de serialização JSON copiado em `auth`, `companies` e `admin`. Mudança de formato de resposta exige editar 3 arquivos.

**Arquivos:**
- [api/internal/auth/handler.go:55-59](api/internal/auth/handler.go)
- [api/internal/companies/handler.go:67-71](api/internal/companies/handler.go)
- [api/internal/admin/handler.go:99-103](api/internal/admin/handler.go)

**O que fazer:**
1. Criar `api/pkg/httputil/response.go` com `func JSON(w http.ResponseWriter, status int, v any)` e `func Error(w http.ResponseWriter, status int, msg string)`
2. Substituir as cópias locais pelo helper centralizado

- [x] Criar `pkg/httputil/response.go`
- [x] Atualizar `auth/handler.go`
- [x] Atualizar `companies/handler.go`
- [x] Atualizar `admin/handler.go`

---

### M2 — Sem camada de domínio — structs espalhadas

**Problema:** `Company`, `User`, `Org`, `Plan` e outros modelos definidos dentro dos packages de feature. Imports cruzados vão surgir conforme o projeto cresce.

**Arquivos:**
- [api/internal/companies/service.go:13-58](api/internal/companies/service.go) — structs de Company
- [api/internal/admin/service.go:16-42](api/internal/admin/service.go) — structs de Org e Plan

**O que fazer:**
1. Criar `api/internal/domain/` com sub-arquivos por entidade (`company.go`, `user.go`, `org.go`)
2. Mover structs para lá e atualizar imports

- [x] Criar `internal/domain/company.go`
- [x] Criar `internal/domain/user.go` (dentro de org.go)
- [x] Criar `internal/domain/org.go`
- [x] Criar `internal/domain/errors.go` com tipos de erro de negócio

---

### M3 — Sem validação de input nos handlers

**Problema:** Handlers confiam que os dados do request são válidos. Erros de validação chegam como erros de banco (confuso) ou causam panics.

**O que fazer:**
1. Para cada handler que decodifica JSON, adicionar validação explícita dos campos obrigatórios
2. Retornar `400 Bad Request` com mensagem descritiva antes de chamar o service

- [x] `auth/handler.go` — valida `email` e `password` no Login; valida `refresh_token` no Refresh
- [x] `admin/handler.go` — valida `name` em CreateOrg; valida `email`, `password`, `role` em CreateUser

---

### M4 — Sem rate limiting

**Problema:** Endpoints de autenticação e de busca (que consomem créditos) sem proteção contra abuso ou força bruta.

**O que fazer:**
1. Adicionar middleware de rate limiting por IP usando `golang.org/x/time/rate` ou `go-chi/httprate`
2. Aplicar limite mais restrito em `/auth/login` (ex: 10 req/min por IP)
3. Aplicar limite em `/companies` atrelado ao plano da organização

- [ ] Middleware de rate limiting global
- [ ] Rate limit específico em `/auth/login`
- [ ] Rate limit por organização em `/companies`

---

### M5 — CORS hardcoded em `main.go` com fallback `"*"`

**Problema:** `corsMiddleware` em [api/cmd/server/main.go:74-90](api/cmd/server/main.go) usa `origin = "*"` quando o header `Origin` está ausente — allow-all silencioso.

**O que fazer:**
1. Mover para `pkg/middleware/cors.go`
2. Ler origens permitidas de variável de ambiente `CORS_ALLOWED_ORIGINS`
3. Remover fallback para `"*"`

- [x] Criar `pkg/middleware/cors.go` com configuração via env `CORS_ALLOWED_ORIGINS`
- [x] Atualizar `main.go` para usar o novo middleware

---

### M6 — Sem logging estruturado

**Problema:** Uso de `log.Printf` e `log.Fatal` do pacote padrão. Sem campos estruturados, difícil filtrar em produção.

**O que fazer:**
1. Substituir por `log/slog` (nativo no Go 1.21+, zero dependência extra)
2. Inicializar em `main.go` com `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))`
3. Nos services/handlers, usar `slog.ErrorContext(ctx, "msg", "key", value)`

- [x] Inicializar `slog` em `main.go` com JSON handler
- [x] Substituir `log.Printf` em `main.go` por `slog.Info`
- [x] Usar `slog.WarnContext` no Refresh (rotação de token) e `slog.ErrorContext` no Logout
- [x] httputil.JSON loga encode errors via `slog.Error`

---

## LOW / Tooling

### T1 — Sem CI/CD

- [x] Criar `.github/workflows/ci.yml` com: lint Go, lint frontend, testes, build
- [x] Rodar em PR e push para `main`

---

### T2 — Sem pre-commit hooks ativos

- [ ] Configurar `pre-commit` ou adicionar hook em `.git/hooks/pre-commit` para rodar `make lint`
- [ ] Documentar no README como ativar

---

### T3 — Índice HNSW fora das migrations

**Arquivo:** [api/migrations/002_indices.sql](api/migrations/002_indices.sql) — comentário indica criação manual pós-ingestão.

- [x] Criar migration 004 dedicada (`004_hnsw_index.sql`) com `SET maintenance_work_mem = '2GB'` — rodar após ingestão completa e geração de embeddings

---

### T4 — `docker-compose.prod.yml` vazio

- [ ] Definir serviços de produção: API Go (multi-stage build), Next.js, Postgres, Redis
- [ ] Adicionar healthchecks e restart policies

---

### T5 — Sem documentação de API

- [ ] Considerar OpenAPI 3.1 (arquivo `api/openapi.yaml`) para documentar endpoints
- [ ] Ou ao menos um `API.md` com curl examples por endpoint

---

## Progresso geral

| Categoria | Total | Concluído |
|---|---|---|
| HIGH | 4 itens principais | 3 (H4 pendente — testes) |
| MEDIUM | 6 itens principais | 5 (M4 pendente — rate limiting) |
| LOW/Tooling | 5 itens | 2 (T1, T3) |

_Atualizado em: 2026-06-17 — refactoring completo de H1/H2/H3/M1/M2/M3/M5/M6; T1 e T3 implementados_
