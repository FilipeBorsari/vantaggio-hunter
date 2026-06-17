# Testes — HTTPUtil

Cobertura dos helpers de resposta HTTP: `pkg/httputil/`.

---

## response_test.go

Testa as funções `JSON` e `Error` do pacote `httputil`.

### JSON(w, status, payload)

| Teste | O que verifica |
|---|---|
| `TestJSON_SetsStatusAndContentType` | Status HTTP correto e `Content-Type: application/json` |
| `TestJSON_EncodesPayload` | Struct serializada corretamente no body |
| `TestJSON_SlicePayload` | Slice serializado corretamente no body |

### Error(w, status, msg)

| Teste | O que verifica |
|---|---|
| `TestError_SetsStatusAndMessage` | Status correto e `{"error": "mensagem"}` no body |
| `TestError_ContentType` | `Content-Type: application/json` presente |
| `TestError_Codes` | Todos os status comuns (400, 401, 403, 404, 409, 500) são repassados corretamente |

---

## Contrato esperado

Toda resposta de erro da API deve ter o formato:

```json
{ "error": "mensagem legível ao cliente" }
```

Nunca vazar detalhes de infraestrutura (stack trace, query SQL, etc.) no campo `error`.

---

## Como rodar

```bash
make test
# ou apenas o pacote httputil:
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./pkg/httputil/...
```
