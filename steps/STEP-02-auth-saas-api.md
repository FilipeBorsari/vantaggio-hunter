# STEP-02 — Auth + Gestão SaaS + Lead Bank API

## Objetivo
Implementar autenticação JWT com RBAC, painel admin para gestão de orgs/usuários e os endpoints de consulta de empresas — tornando o sistema multi-tenant e acessível.

## Pré-requisitos
- STEP-00 concluído (infraestrutura)
- STEP-01 concluído (dados de empresas no Postgres)
- `JWT_SECRET` configurado no `.env`

## Escopo
- ✅ Schema transacional: orgs, usuários, planos
- ✅ Auth JWT (login, refresh, logout)
- ✅ Middleware RBAC com extração de `org_id`
- ✅ Painel admin para criação de orgs e usuários (sem self-service público)
- ✅ Endpoints de consulta de empresas (paginados)
- ✅ Página de login no frontend
- ✅ Layout principal: Sidebar + Topbar
- ❌ Créditos (STEP-04)
- ❌ Buscas assíncronas com queue (STEP-03)
- ❌ Reseller (STEP-08)

---

## 1. Banco de Dados

### Migration 003 — Schema transacional

```sql
-- migrations/003_auth_saas.sql

-- Planos de assinatura
CREATE TABLE tb_plans (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(50)  NOT NULL,   -- 'Starter', 'Pro', 'Enterprise'
  credits     INT          NOT NULL,   -- créditos incluídos no plano
  price_cents INT          NOT NULL,   -- preço em centavos
  active      BOOLEAN      NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Organizações (tenants)
CREATE TABLE tb_organizations (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name         VARCHAR(255) NOT NULL,
  plan_id      UUID        REFERENCES tb_plans(id),
  is_active    BOOLEAN     NOT NULL DEFAULT true,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Usuários
CREATE TABLE tb_users (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID        NOT NULL REFERENCES tb_organizations(id),
  email        VARCHAR(255) NOT NULL UNIQUE,
  password_hash TEXT        NOT NULL,
  role         VARCHAR(20)  NOT NULL CHECK (role IN ('admin','manager','operator')),
  is_active    BOOLEAN      NOT NULL DEFAULT true,
  deleted_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_org_id ON tb_users (org_id);
CREATE INDEX idx_users_email  ON tb_users (email);

-- Seed: planos padrão
INSERT INTO tb_plans (name, credits, price_cents) VALUES
  ('Starter',    1000,   9900),
  ('Pro',        10000,  4900),  -- por mil: mais barato
  ('Enterprise', 100000, 19900);

-- Seed: organização master (o próprio Vantaggio)
INSERT INTO tb_organizations (id, name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Vantaggio');

-- Seed: usuário admin master (alterar senha após primeiro deploy)
INSERT INTO tb_users (org_id, email, password_hash, role) VALUES
  ('00000000-0000-0000-0000-000000000001',
   'admin@vantaggio.com.br',
   '$2a$12$PLACEHOLDER_HASH_CHANGE_ON_DEPLOY',
   'admin');
```

---

## 2. Backend (Go)

### Estrutura de arquivos

```
api/internal/
├── auth/
│   ├── handler.go      # POST /auth/login, /auth/refresh
│   ├── service.go      # lógica de JWT, bcrypt
│   └── middleware.go   # extração de claims do token
├── admin/
│   ├── handler.go      # rotas /admin/*
│   └── service.go
└── companies/
    ├── handler.go      # GET /companies, GET /companies/:cnpj
    └── service.go
```

### 2.1 Endpoints de Auth

**POST /auth/login**
```
Body:    { "email": "string", "password": "string" }
200:     { "access_token": "...", "refresh_token": "...", "expires_in": 86400 }
401:     { "error": "credenciais inválidas" }
```
- Buscar usuário por email
- Verificar `bcrypt.CompareHashAndPassword`
- Verificar `is_active = true` e `deleted_at IS NULL`
- Gerar JWT com claims: `{ user_id, org_id, role, exp }`

**POST /auth/refresh**
```
Body:    { "refresh_token": "string" }
200:     { "access_token": "...", "expires_in": 86400 }
401:     { "error": "token inválido ou expirado" }
```

**POST /auth/logout** *(opcional — invalida refresh token)*
```
Header:  Authorization: Bearer {access_token}
204:     sem body
```

### 2.2 Middleware de Auth

```go
// Extrair do JWT e injetar no contexto:
// - userID  (uuid.UUID)
// - orgID   (uuid.UUID)
// - role    (string)

// Helper de autorização por role:
func RequireRole(roles ...string) func(http.Handler) http.Handler
```

Uso nas rotas:
```go
r.Group(func(r chi.Router) {
    r.Use(middleware.Authenticate)
    r.Use(middleware.RequireRole("admin"))
    r.Post("/admin/organizations", adminHandler.CreateOrg)
})
```

### 2.3 Endpoints Admin (role: admin)

**POST /admin/organizations**
```
Body:    { "name": "string", "plan_id": "uuid" }
201:     { "id": "uuid", "name": "string", "plan_id": "uuid", "created_at": "..." }
```

**POST /admin/organizations/:id/users**
```
Body:    { "email": "string", "password": "string", "role": "manager|operator" }
201:     { "id": "uuid", "email": "string", "role": "string" }
409:     se email já existe
```

**GET /admin/organizations**
```
Query:   ?page=1&limit=20
200:     { "data": [{ "id", "name", "plan", "user_count", "created_at" }], "total": N }
```

**PATCH /admin/organizations/:id/users/:userId**
```
Body:    { "is_active": bool }   -- ativar/desativar usuário
204:     sem body
```

### 2.4 Endpoints de Empresas (autenticados)

**GET /companies**
```
Query:   ?cnae=4520-0%2F01&uf=SP&city=São+Paulo&capital_min=100000&status=2&page=1&limit=50
200:     {
           "data": [{
             "cnpj": "...",
             "razao_social": "...",
             "municipio": "...",
             "uf": "...",
             "capital_social": 150000,
             "situacao": 2,
             "cnaes": [{ "code": "...", "description": "...", "is_primary": true }]
           }],
           "total": N,
           "page": 1,
           "limit": 50
         }
```
- Filtros são todos opcionais e combinados com AND
- Multi-CNAE: `cnae=4520-0%2F01,4711-3%2F01` (vírgula separando)
- Sem créditos neste step — consulta é livre (créditos virão no STEP-04)

**GET /companies/:cnpj**
```
200:     { objeto completo da empresa + cnaes + sócios }
404:     { "error": "empresa não encontrada" }
```

---

## 3. Frontend (Next.js)

### Rotas

```
/login                    ← pública
/admin                    ← privada, apenas role=admin
  /admin/organizations    ← listar orgs
  /admin/organizations/new ← criar org + primeiro usuário
/companies                ← privada, todos os roles (Lead Bank simples neste step)
```

### Componentes

**`/app/login/page.tsx`**
- Formulário: email + senha
- `POST /auth/login` → salvar `access_token` e `refresh_token` em httpOnly cookies
- Redirect para `/companies` em caso de sucesso

**`/components/layout/AppLayout.tsx`**
- Sidebar com navegação (ícones + labels)
- Topbar com nome do usuário + botão de logout
- Itens da sidebar visíveis conforme role:
  - admin: Admin, Companies
  - manager: Companies
  - operator: Companies

**`/app/admin/organizations/page.tsx`**
- Tabela de organizações com nome, plano, total de usuários
- Botão "Nova Organização"

**`/app/admin/organizations/new/page.tsx`**
- Formulário: nome da org + seleção de plano
- Após criar org: formulário inline para criar primeiro usuário (email + senha + role)

**`/app/companies/page.tsx`** *(Lead Bank simplificado)*
- Filtros: CNAE (texto livre por enquanto), UF (select), status
- Tabela com virtualização (`@tanstack/react-virtual`)
- Colunas: CNPJ, Razão Social, Município/UF, Capital, Situação

---

## 4. Integrações / Infra

- Senhas: `bcrypt` com custo 12
- JWT: access token expira em 24h, refresh token em 30 dias
- Refresh token: armazenar hash no banco (tabela `tb_refresh_tokens`) para possibilitar revogação
- CORS: permitir apenas origem do frontend em produção

### Tabela de refresh tokens (adicionar na migration 003)
```sql
CREATE TABLE tb_refresh_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES tb_users(id) ON DELETE CASCADE,
  token_hash  TEXT        NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON tb_refresh_tokens (user_id);
```

---

## 5. Testes e Validação

- [ ] `POST /auth/login` com credenciais válidas retorna tokens JWT
- [ ] `POST /auth/login` com senha errada retorna 401
- [ ] `GET /companies` sem token retorna 401
- [ ] `GET /companies` com token de `operator` retorna lista paginada
- [ ] `POST /admin/organizations` com token de `operator` retorna 403
- [ ] `POST /admin/organizations` com token de `admin` cria org e retorna 201
- [ ] Criar usuário na org criada → novo usuário consegue logar
- [ ] Usuário de org A não vê dados de org B (isolamento de tenant — validar via `org_id` no JWT)
- [ ] Filtro `?uf=SP&cnae=4520-0%2F01` retorna apenas empresas do estado e CNAE corretos
- [ ] Paginação: `?page=2&limit=10` retorna registros 11-20
- [ ] Frontend: login redireciona corretamente; logout limpa tokens; sidebar oculta admin para operator

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| JWT_SECRET fraco em produção | Gerar com `openssl rand -base64 32` e nunca commitar |
| SQL injection nos filtros de `/companies` | Usar parâmetros posicionais `$1`, `$2`... NUNCA interpolação de string |
| Admin seed com senha placeholder | Documentar que a primeira ação pós-deploy é trocar a senha via endpoint |
| Multi-CNAE query lenta sem índice | Índice `idx_company_cnaes_code` já criado no STEP-01 cobre este caso |
| Refresh token roubado | Implementar rotação de refresh token: cada uso gera novo token e invalida o anterior |
