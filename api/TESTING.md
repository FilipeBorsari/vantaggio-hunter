# TESTING.md — Vantaggio Hunter API

Guia completo da suite de testes do `api/`.

---

## Como rodar

```bash
# Todos os testes (da raiz do monorepo)
make test

# Equivalente direto (dentro de api/)
/home/filipeborsari/go/bin/gotestsum --format testdox

# Um pacote específico
/home/filipeborsari/go/bin/gotestsum --format testdox -- ./internal/auth/...

# Com cobertura
/home/filipeborsari/go/bin/go test -cover ./...

# Relatório de cobertura em HTML
/home/filipeborsari/go/bin/go test -coverprofile=coverage.out ./...
/home/filipeborsari/go/bin/go tool cover -html=coverage.out
```

---

## Estrutura da suite

```
api/
├── pkg/httputil/
│   └── response_test.go        ← helpers JSON/Error
├── internal/auth/
│   ├── service_test.go         ← Login, Refresh, Logout, HashPassword
│   ├── handler_test.go         ← handlers HTTP de autenticação
│   └── middleware_test.go      ← JWT middleware + RequireRole
├── internal/companies/
│   ├── service_test.go         ← List, GetByCNPJ
│   └── handler_test.go         ← parâmetros de query, caps, chi params
├── internal/admin/
│   ├── service_test.go         ← CRUD orgs/users, hash de senha
│   └── handler_test.go         ← handlers + utilitário intParam
└── cmd/ingestion/
    └── csv_parser_test.go      ← parsers puros + streaming via temp files
```

Documentação detalhada por contexto em [`docs/tests/`](../docs/tests/).

---

## Filosofia

### 1. Testes unitários com mocks — sem banco real

Toda a lógica de negócio vive em `service.go`. Cada service é testado com
um stub da interface `Repository` definido no próprio `_test.go`. Isso
garante que o teste é rápido (sem I/O de rede), determinístico e
independente de estado externo.

```
Handler test  →  mock ServiceInterface
Service test  →  mock Repository interface
```

### 2. Handlers via `httptest`

```go
r := httptest.NewRequest(http.MethodPost, "/auth/login", body)
w := httptest.NewRecorder()
h.Login(w, r)

if w.Code != http.StatusOK { ... }
```

Para rotas com parâmetros chi (`{cnpj}`, `{id}`):

```go
rctx := chi.NewRouteContext()
rctx.URLParams.Add("cnpj", "12345678000100")
r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
```

### 3. Middleware JWT

```go
t.Setenv("JWT_SECRET", "test-secret")
token := makeTestToken(t, "user-42", "org-7", "admin", secret, time.Hour)
r.Header.Set("Authorization", "Bearer "+token)
```

### 4. Parsers CSV com arquivos temporários

```go
path := writeTempCSV(t, "6201500;Desenvolvimento de software\n")
rows, err := ParseCNAEs(path)
```

### 5. Bcrypt em testes

Use `bcrypt.MinCost` (custo 4) — nunca custo 12:

```go
b, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
```

---

## Variáveis de ambiente

| Variável | Onde usada | Como setada |
|---|---|---|
| `JWT_SECRET` | `auth/service_test.go`, `auth/middleware_test.go` | `t.Setenv(...)` — revertida ao fim do teste |

Nenhuma variável de banco (`DATABASE_URL`, `REDIS_URL`) é necessária.

---

## Próximos passos

- **Testes de repositório**: usar `testcontainers-go` com Postgres real
- **Testes e2e**: `httptest.NewServer` com o roteador completo
