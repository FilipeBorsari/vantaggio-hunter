# CLAUDE.md — Vantaggio Hunter

Guia de desenvolvimento para Claude. Leia este arquivo antes de qualquer sessão de código neste repositório.

## Produto

B2B SaaS de prospecção de leads qualificados via dados da Receita Federal. Multi-tenant com créditos por busca, planos e organizações. Stack: Go API + Next.js (App Router) + PostgreSQL 16 (pgvector) + Redis.

## Estrutura do monorepo

```
api/          Go API (chi, pgx/v5, goose)
web/          Next.js 16 (App Router, TypeScript, Tailwind 4)
infra/        Docker Compose (dev e prod)
steps/        Documentação de implementação por etapa
migrations/   SQL migrations (goose, numeradas)
```

---

## Clean Architecture — regras obrigatórias

### Camadas (de fora para dentro)

```
Handler (HTTP)  →  Service (interface)  →  Repository (interface)  →  DB impl
```

- **Handler** só faz decode do request, chama o service, encode da response. Sem lógica de negócio.
- **Service** recebe a interface do repository — nunca `*pgxpool.Pool` diretamente.
- **Repository** é uma interface definida no package de domínio; a implementação fica em `_postgres.go`.
- **Domain/entities** ficam em `internal/domain/` — structs compartilhadas entre packages.

### Exemplo de estrutura para um novo feature

```
internal/
  domain/
    company.go          ← structs Company, SearchFilter, etc.
  companies/
    repository.go       ← type Repository interface { ... }
    repository_postgres.go ← type postgresRepo struct { db *pgxpool.Pool }
    service.go          ← type Service struct { repo Repository }
    handler.go          ← type Handler struct { svc *Service }
```

### Inversão de dependência

- Defina a interface **no package consumidor** (service define `Repository`, não o contrário).
- Injete via construtor: `NewService(repo Repository) *Service`.
- `main.go` é o único lugar que instancia implementações concretas.

---

## Error handling — regras obrigatórias

- **Nunca** use `//nolint:errcheck`, `_, _ =`, ou `_ =` para silenciar erros. Se o erro não importa, documente explicitamente por quê.
- Sempre use `fmt.Errorf("contexto: %w", err)` ao propagar — nunca descarte o `%w`.
- Erros de negócio devem ter tipos próprios (ex: `ErrNotFound`, `ErrInvalidCNPJ`) definidos em `internal/domain/errors.go`.
- Handlers mapeiam erros de domínio para status HTTP — sem vazar detalhes de infraestrutura para o cliente.

```go
// ERRADO
json.NewEncoder(w).Encode(v) //nolint:errcheck

// CERTO
if err := json.NewEncoder(w).Encode(v); err != nil {
    log.Printf("encode response: %v", err)
}
```

---

## Resposta HTTP — padrão único

Use o helper central em `pkg/httputil/response.go` — não duplique `writeJSON` em cada package.

```go
httputil.JSON(w, http.StatusOK, payload)
httputil.Error(w, http.StatusBadRequest, "mensagem ao cliente")
```

---

## Testes

- Todo novo `service.go` e `repository.go` deve ter um `_test.go` correspondente.
- Services são testados com mocks da interface de repository (sem banco real).
- Repositories são testados com banco real via `testcontainers-go` ou banco de teste dedicado.
- Handlers são testados com `httptest.NewRecorder` e um mock do service.
- Não crie lógica sem teste quando estiver adicionando uma feature nova.

---

## Validação de input

- Valide no handler, antes de chamar o service.
- Para requests JSON, use uma struct com tags e valide campos obrigatórios explicitamente.
- Retorne `400 Bad Request` com mensagem descritiva para input inválido.

---

## Logging

- Use `log/slog` (Go 1.21+) com `slog.Default()` estruturado — não `fmt.Println` nem `log.Printf` no código de negócio.
- Nível `Info` para operações normais, `Warn` para degradação, `Error` para falhas inesperadas.
- Sempre inclua contexto relevante: `slog.String("cnpj", cnpj)`, `slog.Int("org_id", orgID)`.

---

## O que NÃO fazer

| Anti-pattern | Por quê |
|---|---|
| `pool *pgxpool.Pool` em services | Amarra a lógica ao banco — impossível testar com mock |
| Duplicar `writeJSON` em cada package | Qualquer mudança de formato quebra N lugares |
| `errors.Is(err, pgx.ErrNoRows) \|\| err != nil` | A segunda condição engole qualquer erro como "not found" |
| Retornar `nil` após falha de query | Propaga estado inválido silenciosamente |
| `return fmt.Errorf("not found")` sem `%w` | Perde a cadeia de contexto do erro original |
| Acesso direto a `os.Getenv` em services | Dificulta testes e viola SRP — use config injetada |
| Lógica de negócio em `main.go` | `main.go` só conecta dependências |

---

## Go — convenções deste projeto

- Module: `github.com/vantaggio/prospect-api`
- Go binary: `/home/filipeborsari/go/bin/go` (não está no PATH padrão)
- Executar API: `make api` ou `/home/filipeborsari/go/bin/go run ./cmd/server`
- Migrations: `make migrate`
- Lint: `make lint` (golangci-lint)

---

## Frontend — convenções

- Framework: Next.js 16 App Router, TypeScript strict
- Estilo: Tailwind 4 — sem biblioteca de componentes externa
- Auth: cookies httpOnly via BFF proxy (`src/lib/proxy.ts`) — nunca exponha tokens no JS do cliente
- Path aliases: `@/` → `src/`
- Lint: `pnpm lint` (ESLint 9)

---

## Makefile targets principais

```
make dev-up      # sobe Postgres + Redis
make api         # inicia o servidor Go
make web         # inicia o Next.js
make migrate     # roda migrations pendentes
make lint        # lint Go + frontend
make build       # compila binários Go
```
