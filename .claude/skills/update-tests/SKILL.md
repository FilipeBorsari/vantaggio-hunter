# Skill: Update Tests

> Garante que testes e documentação estejam sincronizados após qualquer mudança de código.

## Objetivo

Após implementar uma nova feature, corrigir um bug ou modificar código existente, verificar se a suite de testes e os documentos em `docs/tests/` refletem o estado atual do código.

## Quando Usar

- Após criar um novo `service.go`, `handler.go` ou `repository.go`
- Após adicionar um novo endpoint ou rota
- Após mudar a assinatura de um método em uma interface
- Após adicionar um novo erro de domínio em `internal/domain/errors.go`
- Após modificar lógica de parsing no `cmd/ingestion/csv_parser.go`
- Sempre que `make test` estiver passando mas a cobertura do novo código for zero

## Processo

### 1. Identifique o que mudou

Leia os arquivos modificados e responda:
- Quais métodos/funções foram adicionados ou removidos?
- Alguma interface mudou de assinatura?
- Algum novo erro de domínio foi criado?
- Algum novo campo foi adicionado a uma struct?

### 2. Localize os arquivos de teste correspondentes

| Código modificado | Arquivo de teste |
|---|---|
| `internal/auth/service.go` | `internal/auth/service_test.go` |
| `internal/auth/handler.go` | `internal/auth/handler_test.go` |
| `internal/auth/middleware.go` | `internal/auth/middleware_test.go` |
| `internal/companies/service.go` | `internal/companies/service_test.go` |
| `internal/companies/handler.go` | `internal/companies/handler_test.go` |
| `internal/admin/service.go` | `internal/admin/service_test.go` |
| `internal/admin/handler.go` | `internal/admin/handler_test.go` |
| `pkg/httputil/response.go` | `pkg/httputil/response_test.go` |
| `cmd/ingestion/csv_parser.go` | `cmd/ingestion/csv_parser_test.go` |
| Novo pacote | Criar `<pacote>/_test.go` seguindo os padrões abaixo |

### 3. Verifique os mocks

Se uma interface mudou, os stubs nos arquivos `_test.go` precisam ser atualizados:

```go
// Exemplo: novo método adicionado à interface Repository
// Adicionar ao mockRepo correspondente:
func (m *mockRepo) NovoMetodo(_ context.Context, _ string) error {
    return m.novoMetodoErr
}
```

### 4. Escreva os testes que faltam

**Padrões obrigatórios deste projeto** (ver `api/TESTING.md`):

- **Services**: mock da interface `Repository` definido no próprio `_test.go`, sem banco real
- **Handlers**: `httptest.NewRecorder` + mock do `ServiceInterface`; parâmetros chi injetados via:
  ```go
  rctx := chi.NewRouteContext()
  rctx.URLParams.Add("id", "valor")
  r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
  ```
- **Middleware JWT**: `t.Setenv("JWT_SECRET", "test-secret")` + `makeTestToken`
- **bcrypt em testes**: use `bcrypt.MinCost` (custo 4), nunca custo 12
- **Novos erros de domínio**: teste que o service os propaga e que o handler os mapeia para o status HTTP correto

**Casos mínimos para cada novo método de service:**
- [ ] Sucesso com dados válidos
- [ ] Erro do repositório propagado
- [ ] Casos de borda específicos do negócio (ex: email duplicado, usuário inativo)

**Casos mínimos para cada novo handler:**
- [ ] 200/201/204 com body válido
- [ ] 400 para body inválido ou campos obrigatórios ausentes
- [ ] Status de erro correto quando o service falha

### 5. Rode os testes

```bash
make test
```

Todos devem passar antes de continuar. Se falhar, corrija o código ou o teste — nunca ignore.

Para rodar apenas o pacote afetado:
```bash
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./<pacote>/...
```

### 6. Atualize a documentação em docs/tests/

Identifique o(s) contexto(s) afetado(s) e edite o README correspondente:

| Contexto | Documento |
|---|---|
| `internal/auth/` | `docs/tests/auth/README.md` |
| `internal/companies/` | `docs/tests/companies/README.md` |
| `internal/admin/` | `docs/tests/admin/README.md` |
| `pkg/httputil/` | `docs/tests/httputil/README.md` |
| `cmd/ingestion/` | `docs/tests/ingestion/README.md` |
| Novo pacote | Criar `docs/tests/<nome>/README.md` com a mesma estrutura |

Cada entrada na tabela de testes deve ter:
- Nome exato do teste (`TestNomeDoTeste`)
- O que a função verifica em linguagem clara

### 7. Reporte o resultado

Liste ao usuário:
- Testes adicionados (nome + arquivo)
- Testes modificados (nome + motivo)
- Documentos atualizados
- Confirmação de que `make test` passou

## Boas Práticas

### ✅ Fazer

- Testar o caminho feliz **e** os erros para todo novo comportamento
- Usar `t.Helper()` em funções auxiliares de teste
- Nomear testes no padrão `Test<Contexto>_<Cenário>` (ex: `TestLogin_InactiveUser`)
- Manter mocks simples — um campo por comportamento configurável

### ❌ Evitar

- Silenciar erros com `_ =` nos testes
- Criar testes que dependem de ordem de execução
- Usar `time.Sleep` — use channels ou mocks de clock
- Testar implementações concretas de repositório sem banco real (use testcontainers)

## Referências

- [`api/TESTING.md`](../../../api/TESTING.md) — filosofia e convenções da suite
- [`docs/tests/`](../../../docs/tests/) — documentação por contexto
- [`CLAUDE.md`](../../../CLAUDE.md) — regras gerais de desenvolvimento
