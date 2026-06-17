---
name: clean-dev
description: Audita código Go e TypeScript em busca de violações de Clean Architecture e Clean Code definidas em CLAUDE.md. Dispara em "revise", "audite", "clean code", "clean arch", "/clean-dev", "verifique boas práticas", "code review de arquitetura".
---

# clean-dev

Audite o código modificado nesta sessão em busca de violações de Clean Architecture e Clean Code conforme as regras definidas em `CLAUDE.md`.

## O que fazer ao ser invocado

1. **Identifique os arquivos em escopo:**
   - Se foram passados argumentos (ex: `/clean-dev api/internal/auth/`), use esse caminho.
   - Caso contrário, execute `git diff --name-only HEAD` para listar arquivos modificados na sessão atual.
   - Se não houver diff (branch limpa), peça ao usuário qual arquivo ou diretório analisar.

2. **Leia `CLAUDE.md`** na raiz do projeto para recarregar as regras vigentes antes de analisar.

3. **Para cada arquivo Go (`.go`) no escopo, verifique:**

   **Clean Architecture:**
   - [ ] O service recebe `*pgxpool.Pool` diretamente? → Deve receber uma interface de repository.
   - [ ] O handler recebe uma struct concreta de service? → Deve receber uma interface.
   - [ ] Há lógica de negócio no handler (além de decode/encode)? → Mover para o service.
   - [ ] Há acesso a `os.Getenv` fora de `main.go`? → Injetar via config struct.
   - [ ] Structs de domínio definidas dentro do package de feature? → Devem estar em `internal/domain/`.

   **Error Handling:**
   - [ ] Há `//nolint:errcheck` em qualquer linha? → Tratar o erro ou documentar explicitamente por quê ignorar é seguro.
   - [ ] Há `_, _ =` ou `_ =` descartando erros de queries ou encoding? → Tratar.
   - [ ] `fmt.Errorf` sem `%w`? → Adicionar `%w` para preservar a cadeia.
   - [ ] `errors.Is(err, pgx.ErrNoRows) || err != nil`? → Tratar casos separadamente.
   - [ ] Função retornando `nil` após falha detectada? → Propagar o erro.

   **Duplicação:**
   - [ ] Função `writeJSON` ou similar definida localmente? → Usar `pkg/httputil`.
   - [ ] CORS middleware em arquivo fora de `pkg/middleware/`? → Mover.

   **Testes:**
   - [ ] Arquivo `.go` sem `_test.go` correspondente? → Reportar como pendente.

4. **Para cada arquivo TypeScript/TSX (`.ts`, `.tsx`) no escopo, verifique:**
   - [ ] Token de auth manipulado no lado do cliente (localStorage, cookie não-httpOnly)? → Deve passar pelo BFF em `src/lib/proxy.ts`.
   - [ ] `fetch` direto para a API Go sem passar pelo proxy? → Usar o helper de `src/lib/api.ts`.
   - [ ] Componente com mais de ~150 linhas misturando UI e lógica? → Sugerir extração de hook ou componente filho.

5. **Produza um relatório estruturado:**

```
## Relatório /clean-dev — <data>

### Arquivos analisados
- path/to/file.go

### Violações encontradas

#### HIGH
- [ ] `api/internal/auth/service.go:43` — Service recebe *pgxpool.Pool diretamente.
  Correção: criar `Repository` interface e `repository_postgres.go`.

#### MEDIUM
- [ ] `api/internal/auth/handler.go:58` — Erro de encode ignorado com //nolint:errcheck.
  Correção: `if err := json.NewEncoder(w).Encode(v); err != nil { log.Printf(...) }`.

#### LOW
- [ ] `api/internal/auth/handler.go` — Sem arquivo de teste correspondente.

### Sem violações
- path/to/clean_file.go ✓
```

6. **Se invocado com `--fix`:** aplique as correções de LOW e MEDIUM automaticamente (duplicate helpers, nolint:errcheck, %w faltando). Para HIGH, descreva o que fazer mas não refatore automaticamente sem confirmação do usuário — essas mudanças envolvem criação de novas interfaces e arquivos.

## Escopo de análise suportado

- `go` — arquivos Go em `api/`
- `ts` / `tsx` — arquivos TypeScript em `web/`
- Sem argumento → diff atual da sessão
- Com caminho → analisa o diretório ou arquivo especificado

## Referências

- [CLAUDE.md](../CLAUDE.md) — regras de desenvolvimento deste projeto
