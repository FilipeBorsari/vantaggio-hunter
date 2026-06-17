# Testes — Auth

Cobertura do módulo de autenticação: `internal/auth/`.

---

## service_test.go

Testa a lógica de negócio em `auth.Service` usando um stub de `auth.Repository`.

| Teste | O que verifica |
|---|---|
| `TestLogin_Success` | Credenciais corretas retornam `TokenPair` com `access_token`, `refresh_token` e `expires_in=86400` |
| `TestLogin_UserNotFound` | Erro do repositório é propagado sem transformação |
| `TestLogin_InactiveUser` | Usuário com `is_active=false` retorna `ErrInvalidCredentials` |
| `TestLogin_WrongPassword` | Senha incorreta retorna `ErrInvalidCredentials` |
| `TestLogin_StoreTokenError` | Falha ao persistir refresh token aborta o login |
| `TestRefresh_Success` | Token válido e não expirado gera novo `TokenPair` |
| `TestRefresh_TokenNotFound` | Token desconhecido retorna `ErrTokenInvalid` |
| `TestRefresh_ExpiredToken` | Token com `expires_at` no passado retorna `ErrTokenInvalid` |
| `TestRefresh_DeleteOldTokenError_StillSucceeds` | Falha na rotação (delete) apenas loga warning — não aborta o refresh |
| `TestLogout_Success` | Deleta todos os refresh tokens do usuário sem erro |
| `TestLogout_RepoError` | Erro do repositório é propagado |
| `TestHashPassword_IsVerifiable` | Hash gerado é verificável via `bcrypt.CompareHashAndPassword` |
| `TestHashPassword_DifferentEachTime` | Mesmo input gera hashes diferentes (salt aleatório) |

---

## handler_test.go

Testa os handlers HTTP usando `httptest.NewRecorder` e um stub de `auth.ServiceInterface`.

| Teste | Endpoint | O que verifica |
|---|---|---|
| `TestLoginHandler_Success` | `POST /auth/login` | 200 + `TokenPair` no body |
| `TestLoginHandler_BadBody` | `POST /auth/login` | 400 para JSON inválido |
| `TestLoginHandler_MissingFields` | `POST /auth/login` | 400 quando email ou password vazios |
| `TestLoginHandler_WrongCredentials` | `POST /auth/login` | 401 quando service retorna erro |
| `TestLoginHandler_OtherServiceError` | `POST /auth/login` | 401 para qualquer erro do service |
| `TestRefreshHandler_Success` | `POST /auth/refresh` | 200 + novo `TokenPair` |
| `TestRefreshHandler_BadBody` | `POST /auth/refresh` | 400 para JSON inválido |
| `TestRefreshHandler_MissingToken` | `POST /auth/refresh` | 400 quando `refresh_token` vazio |
| `TestRefreshHandler_InvalidToken` | `POST /auth/refresh` | 401 quando token inválido |
| `TestLogoutHandler_Success` | `POST /auth/logout` | 204 No Content |
| `TestLogoutHandler_ServiceError` | `POST /auth/logout` | 500 quando service falha |
| `TestLogoutHandler_NoUserIDInContext` | `POST /auth/logout` | 204 mesmo sem `user_id` no contexto (sem panic) |

---

## middleware_test.go

Testa os middlewares `Authenticate` e `RequireRole`.

### Authenticate

| Teste | O que verifica |
|---|---|
| `TestAuthenticate_NoHeader` | 401 sem header `Authorization` |
| `TestAuthenticate_WrongScheme` | 401 com scheme diferente de `Bearer` (ex: `Basic`) |
| `TestAuthenticate_MalformedToken` | 401 para string que não é JWT |
| `TestAuthenticate_ExpiredToken` | 401 para JWT com `exp` no passado |
| `TestAuthenticate_WrongSecret` | 401 para JWT assinado com secret diferente |
| `TestAuthenticate_ValidToken_SetsContext` | 200 + `user_id`, `org_id`, `role` injetados no contexto |

### RequireRole

| Teste | O que verifica |
|---|---|
| `TestRequireRole_AllowedSingleRole` | 200 quando role do contexto está na lista permitida |
| `TestRequireRole_AllowedMultipleRoles` | 200 para qualquer role da lista (admin, manager) |
| `TestRequireRole_Denied` | 403 quando role não está na lista |
| `TestRequireRole_NoRoleInContext` | 403 quando contexto não tem role |
| `TestRequireRole_EmptyAllowedList` | 403 quando lista de roles permitidas está vazia |

---

## Como rodar

```bash
make test
# ou apenas o pacote auth:
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./internal/auth/...
```
