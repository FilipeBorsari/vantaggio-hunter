# STEP-00 — Infraestrutura

## Objetivo
Criar o esqueleto do monorepo com Docker, variáveis de ambiente e scripts de desenvolvimento — base obrigatória para todos os steps seguintes.

## Pré-requisitos
- Docker + Docker Compose instalados
- Go 1.22+ instalado
- Node.js 20+ instalado
- Nenhum step anterior

## Escopo
- ✅ Estrutura de pastas do monorepo
- ✅ `docker-compose.yml` com Postgres + Redis
- ✅ `.env.example` com todas as variáveis necessárias
- ✅ Makefile com comandos de desenvolvimento
- ✅ CI básico (lint + build)
- ❌ Nenhuma lógica de negócio
- ❌ Migrations de dados (isso é STEP-01 e STEP-02)

---

## 1. Estrutura de Pastas

```
vantaggio-prospect/
├── api/                        # Go backend
│   ├── cmd/
│   │   ├── server/             # Entrypoint da API HTTP
│   │   │   └── main.go
│   │   └── ingestion/          # CLI de importação CNPJ (STEP-01)
│   │       └── main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── companies/
│   │   ├── searches/
│   │   ├── credits/
│   │   ├── exports/
│   │   └── ia/
│   ├── pkg/
│   │   ├── db/                 # Conexão Postgres
│   │   ├── redis/              # Conexão Redis
│   │   └── middleware/
│   ├── migrations/             # SQL migrations numeradas
│   ├── go.mod
│   └── go.sum
├── web/                        # Next.js frontend
│   ├── src/
│   │   ├── app/                # App Router
│   │   ├── components/
│   │   │   ├── ui/             # Primitivos (button, input, table...)
│   │   │   └── layout/         # Sidebar, Topbar
│   │   ├── lib/                # fetch helpers, formatters
│   │   └── types/
│   ├── public/
│   ├── package.json
│   └── next.config.ts
├── infra/
│   ├── docker-compose.yml
│   └── docker-compose.prod.yml (esqueleto vazio por ora)
├── steps/                      # Estes arquivos
├── .env.example
├── .env                        # Nunca commitado
├── Makefile
├── PLAN.md
└── .gitignore
```

---

## 2. docker-compose.yml

```yaml
# infra/docker-compose.yml
services:
  postgres:
    image: pgvector/pgvector:pg16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: vantaggio
      POSTGRES_USER: vantaggio
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U vantaggio"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

> **Importante:** usar a imagem `pgvector/pgvector:pg16` — ela já inclui a extensão pgvector instalada. O `CREATE EXTENSION vector` ainda precisa ser executado via migration.

---

## 3. Variáveis de Ambiente (.env.example)

```bash
# Banco de dados
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=vantaggio
POSTGRES_USER=vantaggio
POSTGRES_PASSWORD=changeme
DATABASE_URL=postgres://vantaggio:changeme@localhost:5432/vantaggio?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=change-this-to-a-random-256-bit-secret
JWT_EXPIRY_HOURS=24
JWT_REFRESH_EXPIRY_DAYS=30

# API
PORT=8080
ENV=development

# IA (necessário a partir do STEP-01 para embeddings e STEP-07 para IA)
OPENAI_API_KEY=sk-...
GEMINI_API_KEY=AIza...
AI_EMBEDDING_MODEL=text-embedding-3-small   # OpenAI, 1536 dims
AI_CHAT_MODEL=gpt-4o-mini                   # padrão; pode ser gemini-1.5-flash

# Chatwoot (necessário no STEP-06)
CHATWOOT_BASE_URL=https://...
CHATWOOT_API_KEY=...

# S3 (necessário a partir do STEP-06)
AWS_REGION=us-east-1
AWS_BUCKET=vantaggio-exports
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## 4. Makefile

```makefile
.PHONY: dev api web migrate migrate-down lint build

# Sobe todos os serviços Docker
dev-up:
	docker compose -f infra/docker-compose.yml up -d

# Para os serviços Docker
dev-down:
	docker compose -f infra/docker-compose.yml down

# Roda o servidor Go
api:
	cd api && go run ./cmd/server

# Roda o frontend Next.js
web:
	cd web && npm run dev

# Executa migrations (usa goose ou migrate)
migrate:
	cd api && go run ./cmd/migrate up

migrate-down:
	cd api && go run ./cmd/migrate down

# Lint
lint:
	cd api && golangci-lint run
	cd web && npm run lint

# Build de produção
build:
	cd api && go build -o bin/server ./cmd/server
	cd web && npm run build
```

---

## 5. Configuração Inicial Go

```bash
# Inicializar módulo Go
cd api
go mod init github.com/vantaggio/prospect-api

# Dependências iniciais
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get github.com/golang-jwt/jwt/v5
go get github.com/go-chi/chi/v5
go get github.com/pressly/goose/v3   # migrations
```

---

## 6. Configuração Inicial Next.js

```bash
cd web
npx create-next-app@latest . --typescript --tailwind --app --src-dir --import-alias "@/*"

# Dependências adicionais
npm install @tanstack/react-table @tanstack/react-virtual
npm install react-hook-form zod @hookform/resolvers
npm install lucide-react
npm install recharts                 # gráficos do dashboard
```

---

## 7. .gitignore

```
.env
*.env.local
api/bin/
web/.next/
web/node_modules/
*.csv                   # arquivos CNPJ Aberto (muito grandes)
postgres_data/
```

---

## 8. Testes e Validação

- [ ] `docker compose -f infra/docker-compose.yml up -d` inicia sem erros
- [ ] `docker compose ps` mostra postgres e redis como "healthy"
- [ ] `psql $DATABASE_URL -c "SELECT 1"` conecta com sucesso
- [ ] `redis-cli -u $REDIS_URL ping` retorna PONG
- [ ] `make api` compila o servidor Go (mesmo sem rotas ainda)
- [ ] `make web` inicia o Next.js em localhost:3000
- [ ] `.env` não aparece em `git status`

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Porta 5432 já em uso localmente | Mapear para 5433 no docker-compose e ajustar DATABASE_URL |
| Imagem pgvector não disponível offline | Fazer `docker pull pgvector/pgvector:pg16` antes de começar |
| Credenciais vazadas no `.env` | `.gitignore` explícito + pre-commit hook verificando `.env` |
| Conflito de versões Go | Usar `go.toolchain` no `go.mod` para fixar versão |
