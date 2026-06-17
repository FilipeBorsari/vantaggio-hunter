# Testes — Admin

Cobertura do módulo administrativo (orgs, users, planos): `internal/admin/`.

---

## service_test.go

Testa a lógica de negócio em `admin.Service` usando um stub de `admin.Repository`.

### ListPlans

| Teste | O que verifica |
|---|---|
| `TestListPlans_ReturnsPlans` | Retorna a lista de planos do repositório |
| `TestListPlans_RepoError` | Erro do repositório é propagado |

### CreateOrg

| Teste | O que verifica |
|---|---|
| `TestCreateOrg_Success` | Cria org com `plan_id` e retorna o objeto criado |
| `TestCreateOrg_NoPlan` | Cria org sem plano (`plan_id=nil`) |
| `TestCreateOrg_RepoError` | Erro do repositório é propagado |

### CreateUser

| Teste | O que verifica |
|---|---|
| `TestCreateUser_Success` | Cria usuário e retorna `User` com email e role corretos |
| `TestCreateUser_EmailAlreadyExists` | `ErrEmailAlreadyExists` do repo é propagado |
| `TestCreateUser_RepoError` | Qualquer outro erro do repo é propagado |
| `TestCreateUser_PasswordIsHashed` | A senha nunca é armazenada em plaintext — o repositório recebe o hash bcrypt |

### ListOrgs

| Teste | O que verifica |
|---|---|
| `TestListOrgs_Success` | Retorna `OrgListResponse` com `data` e `total` corretos |
| `TestListOrgs_ListError` | Erro em `ListOrgs` do repo é propagado |
| `TestListOrgs_CountError` | Erro em `CountOrgs` do repo é propagado |

### SetUserActive

| Teste | O que verifica |
|---|---|
| `TestSetUserActive_Activate` | Ativa usuário sem erro |
| `TestSetUserActive_Deactivate` | Desativa usuário sem erro |
| `TestSetUserActive_RepoError` | Erro do repositório é propagado |

---

## handler_test.go

Testa os handlers HTTP usando `httptest.NewRecorder`, um stub de `admin.ServiceInterface` e injeção de parâmetros chi via contexto.

### GET /admin/plans

| Teste | O que verifica |
|---|---|
| `TestListPlansHandler_Success` | 200 + array de planos no body |
| `TestListPlansHandler_ServiceError` | 500 quando service falha |

### POST /admin/organizations

| Teste | O que verifica |
|---|---|
| `TestCreateOrgHandler_Success` | 201 + `Org` no body |
| `TestCreateOrgHandler_BadBody` | 400 para JSON inválido |
| `TestCreateOrgHandler_MissingName` | 400 quando `name` está vazio |
| `TestCreateOrgHandler_ServiceError` | 500 quando service falha |

### POST /admin/organizations/{id}/users

| Teste | O que verifica |
|---|---|
| `TestCreateUserHandler_Success` | 201 + `User` no body |
| `TestCreateUserHandler_BadBody` | 400 para JSON inválido |
| `TestCreateUserHandler_MissingFields` | 400 quando `email`, `password` ou `role` estão vazios (3 combinações) |
| `TestCreateUserHandler_EmailAlreadyExists` | 409 Conflict para email duplicado |
| `TestCreateUserHandler_ServiceError` | 500 para outros erros |

### GET /admin/organizations

| Teste | O que verifica |
|---|---|
| `TestListOrgsHandler_Success` | 200 + `OrgListResponse` no body |
| `TestListOrgsHandler_LimitCappedAt100` | `limit=999` é aceito sem erro (cap aplicado antes do service) |
| `TestListOrgsHandler_ServiceError` | 500 quando service falha |

### PATCH /admin/organizations/{id}/users/{userId}

| Teste | O que verifica |
|---|---|
| `TestSetUserActiveHandler_Activate` | 204 ao ativar (`is_active: true`) |
| `TestSetUserActiveHandler_Deactivate` | 204 ao desativar (`is_active: false`) |
| `TestSetUserActiveHandler_BadBody` | 400 para JSON inválido |
| `TestSetUserActiveHandler_ServiceError` | 500 quando service falha |

### Utilitário intParam

| Teste | Entrada → Saída |
|---|---|
| `TestIntParam` | `""→default`, `"0"→default`, `"-5"→default`, `"abc"→default`, `"42"→42`, `"100"→100` |

---

## Como rodar

```bash
make test
# ou apenas o pacote admin:
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./internal/admin/...
```
