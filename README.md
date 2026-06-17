# Vantaggio Hunter

> Transforme milhões de CNPJs da Receita Federal em oportunidades comerciais qualificadas por IA.

---

## O que é

Vantaggio Hunter é uma plataforma B2B SaaS de prospecção que cruza a base pública do **CNPJ Aberto** com modelos de linguagem para entregar leads qualificados, prontos para serem exportados direto para o CRM do time de vendas.

O diferencial não é só encontrar empresas — é **qualificar, pontuar e disparar automaticamente**, fechando o ciclo do SDR sem intervenção humana.

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| API | Go (chi, pgx, goose, JWT) |
| Frontend | Next.js 16 — App Router, TailwindCSS, TanStack Table |
| Banco | PostgreSQL 16 + pgvector |
| Fila | Redis (BLPOP para workers assíncronos) |
| IA | OpenAI (embeddings + chat) / Gemini |
| CRM | Chatwoot (V1) |
| Storage | AWS S3 (exports CSV) |
| Mensageria | Evolution API — WhatsApp (V3) |

---

## Funcionalidades

### V1 — Core de Dados (MVP)
- Busca por CNAE, estado, município, capital social e situação cadastral
- Engine de pesquisa assíncrona via Redis queue
- Sistema de créditos com ledger imutável
- Exportação de leads para Chatwoot
- Multi-tenancy com hierarquia Master → Revenda → Cliente → Usuário
- RBAC: Admin, Revendedor, Gestor, Operador

### V2 — IA Qualificadora
- Score preditivo de qualificação por empresa
- Assistente CNAE: converte linguagem natural em códigos CNAE
- Gerador de templates de prospecção personalizados por perfil do lead

### V3 — SDR Autônomo
- Ciclo completo sem intervenção: encontra → qualifica → gera copy → dispara WhatsApp → transfere para humano no CRM ao detectar interesse

---

## Modelo de Créditos

| Operação | Custo |
|----------|-------|
| Lead retornado na pesquisa | 1 crédito |
| Enriquecimento de lead | 2 créditos |
| Exportação para CRM | 1 crédito |
| Consulta avançada (CNPJ/CPF individual) | 10 créditos |

Planos sugeridos: **Starter** 1k · **Pro** 10k · **Enterprise** 100k créditos.

---

## Estrutura do Projeto

```
vantaggio-hunter/
├── api/                  # Go backend
│   ├── cmd/
│   │   ├── server/       # Entrypoint HTTP
│   │   ├── ingestion/    # CLI de importação CNPJ
│   │   └── migrate/      # Runner de migrations (goose)
│   ├── internal/         # Domínios: auth, companies, searches, credits, exports, ia
│   ├── pkg/              # db, redis, middleware
│   └── migrations/       # SQL migrations numeradas
├── web/                  # Next.js frontend
│   └── src/
│       ├── app/          # App Router
│       ├── components/   # ui/ + layout/
│       ├── lib/          # fetch helpers, formatters
│       └── types/
├── infra/
│   ├── docker-compose.yml       # Postgres + Redis dev
│   └── docker-compose.prod.yml
├── steps/                # Plano de implementação por etapa
├── .env.example
├── Makefile
└── PLAN.md
```

---

## Rodando localmente

### Pré-requisitos

- Docker + Docker Compose
- Go 1.23+
- Node.js 20+

### 1. Variáveis de ambiente

```bash
cp .env.example .env
# edite .env com suas credenciais
```

### 2. Subir banco e cache

```bash
make dev-up
```

### 3. Rodar migrations

```bash
make migrate
```

### 4. API Go

```bash
make api
# → http://localhost:8080/health
```

### 5. Frontend Next.js

```bash
make web
# → http://localhost:3000
```

---

## Comandos úteis

```bash
make dev-up        # sobe Postgres + Redis
make dev-down      # para os containers
make api           # roda o servidor Go
make web           # roda o Next.js
make migrate       # executa migrations pendentes
make migrate-down  # reverte a última migration
make build         # build de produção (api + web)
make lint          # lint Go + Next.js
```

---

## Licença

Proprietário — todos os direitos reservados © Vantaggio.
