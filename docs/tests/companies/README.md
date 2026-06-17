# Testes — Companies

Cobertura do módulo de busca de empresas: `internal/companies/`.

---

## service_test.go

Testa a lógica de negócio em `companies.Service` usando um stub de `companies.Repository`.

### List

| Teste | O que verifica |
|---|---|
| `TestList_Success` | Retorna `CompanyListResponse` com `total`, `data`, `page` e `limit` corretos |
| `TestList_PassesFiltersThrough` | Filtros são repassados ao repositório sem modificação |
| `TestList_AttachCNAEsIsCalled` | `AttachCNAEs` é chamado com a lista retornada por `List` |
| `TestList_EmptyResult` | Resultado vazio retorna `total=0` e `data=[]` sem erro |
| `TestList_CountError` | Erro em `Count` é propagado e aborta a operação |
| `TestList_ListError` | Erro em `List` é propagado e aborta a operação |
| `TestList_AttachCNAEsError` | Erro em `AttachCNAEs` é propagado |

### GetByCNPJ

| Teste | O que verifica |
|---|---|
| `TestGetByCNPJ_Success` | Retorna `CompanyDetail` com CNAEs e parceiros preenchidos |
| `TestGetByCNPJ_NotFound` | `ErrNotFound` do repositório é propagado encadeado via `%w` |
| `TestGetByCNPJ_CNAEsError` | Erro em `GetCNAEsByCNPJ` é propagado |
| `TestGetByCNPJ_PartnersError` | Erro em `GetPartnersByCNPJBasico` é propagado |
| `TestGetByCNPJ_BasicoExtractionFullCNPJ` | CNPJ de 14 dígitos: extrai os 8 primeiros como `cnpj_basico` |
| `TestGetByCNPJ_BasicoExtractionShortCNPJ` | CNPJ com menos de 8 chars não causa panic (basico = CNPJ completo) |

---

## handler_test.go

Testa os handlers HTTP usando `httptest.NewRecorder`, um spy de `companies.ServiceInterface` e injeção de parâmetros chi via contexto.

### GET /companies

| Teste | O que verifica |
|---|---|
| `TestListHandler_DefaultFilters` | Sem query params: `page=1`, `limit=50` |
| `TestListHandler_ParsesQueryParams` | `uf`, `city`, `page`, `limit`, `status`, `capital_min` parseados corretamente |
| `TestListHandler_ParsesCNAEParam` | `cnae=6201500,6202300,4711301` gera slice de 3 elementos |
| `TestListHandler_LimitCappedAt200` | `limit=9999` é reduzido para `200` antes de chamar o service |
| `TestListHandler_InvalidPageDefaultsTo1` | `page=0`, `page=-1`, `page=abc` todos usam default `1` |
| `TestListHandler_ServiceError` | Erro do service retorna 500 |
| `TestListHandler_ResponseShape` | JSON de resposta inclui `data`, `total`, `page`, `limit` |

### GET /companies/{cnpj}

| Teste | O que verifica |
|---|---|
| `TestGetByCNPJHandler_Success` | 200 + `CompanyDetail` no body |
| `TestGetByCNPJHandler_NotFound` | 404 quando service retorna `ErrNotFound` direto |
| `TestGetByCNPJHandler_WrappedNotFound` | 404 quando `ErrNotFound` está encadeado via `fmt.Errorf("%w")` |
| `TestGetByCNPJHandler_InternalError` | 500 para qualquer outro erro |

---

## Como rodar

```bash
make test
# ou apenas o pacote companies:
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./internal/companies/...
```
