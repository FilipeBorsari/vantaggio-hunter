# STEP-05 — Dashboard

## Objetivo
Implementar o dashboard com KPIs, gráficos e funil de conversão usando star schema analytics — dando visibilidade de ROI para o cliente e tornando o produto mais atraente.

## Pré-requisitos
- STEP-04 concluído (créditos e transações registradas)
- STEP-03 concluído (buscas registradas em `tb_searches`)

## Escopo
- ✅ Star schema analítico (dimensões + fatos)
- ✅ ETL job que sincroniza do schema transacional
- ✅ Endpoints analíticos de KPIs, funil e top CNAEs
- ✅ Frontend: DashboardPage com widgets visuais
- ❌ Export CRM no funil (será alimentado no STEP-06)
- ❌ Score de IA no funil (será alimentado no STEP-07)

---

## 1. Banco de Dados

### Migration 006 — Star Schema

```sql
-- migrations/006_analytics.sql

-- ========================
-- DIMENSÕES
-- ========================

CREATE TABLE dim_tempo (
  id          INT         PRIMARY KEY,   -- formato: YYYYMMDD
  data        DATE        NOT NULL UNIQUE,
  dia         SMALLINT    NOT NULL,
  mes         SMALLINT    NOT NULL,
  ano         SMALLINT    NOT NULL,
  trimestre   SMALLINT    NOT NULL,
  dia_semana  SMALLINT    NOT NULL,      -- 0=domingo, 6=sábado
  semana_ano  SMALLINT    NOT NULL,
  is_fim_semana BOOLEAN   NOT NULL
);

CREATE TABLE dim_organizacao (
  id          UUID        PRIMARY KEY,
  name        TEXT        NOT NULL,
  plan_name   TEXT,
  is_active   BOOLEAN     NOT NULL
);

CREATE TABLE dim_usuario (
  id          UUID        PRIMARY KEY,
  email       TEXT        NOT NULL,
  role        TEXT        NOT NULL,
  org_id      UUID        NOT NULL
);

CREATE TABLE dim_cnae (
  code        VARCHAR(10) PRIMARY KEY,
  description TEXT        NOT NULL,
  secao       VARCHAR(5),   -- seção econômica IBGE (ex: G)
  divisao     VARCHAR(5)    -- divisão (ex: 47)
);

CREATE TABLE dim_geografia (
  id              SERIAL      PRIMARY KEY,
  municipio_nome  TEXT        NOT NULL,
  uf              CHAR(2)     NOT NULL,
  regiao          VARCHAR(20),
  UNIQUE (municipio_nome, uf)
);

-- ========================
-- FATOS
-- ========================

-- Fato: consumo de créditos (granularidade: por transação)
CREATE TABLE fato_consumo_creditos (
  id              BIGSERIAL   PRIMARY KEY,
  tempo_id        INT         NOT NULL REFERENCES dim_tempo(id),
  org_id          UUID        NOT NULL REFERENCES dim_organizacao(id),
  usuario_id      UUID        REFERENCES dim_usuario(id),
  tipo            VARCHAR(20) NOT NULL,
  creditos        INT         NOT NULL,   -- valor absoluto (positivo)
  eh_entrada      BOOLEAN     NOT NULL    -- true=entrada, false=saída
);

CREATE INDEX idx_fato_consumo_org_tempo ON fato_consumo_creditos (org_id, tempo_id);

-- Fato: leads no funil (granularidade: por busca)
CREATE TABLE fato_funil_leads (
  id              BIGSERIAL   PRIMARY KEY,
  tempo_id        INT         NOT NULL REFERENCES dim_tempo(id),
  org_id          UUID        NOT NULL REFERENCES dim_organizacao(id),
  usuario_id      UUID        REFERENCES dim_usuario(id),
  cnae_id         VARCHAR(10) REFERENCES dim_cnae(code),
  geo_id          INT         REFERENCES dim_geografia(id),
  leads_extraidos INT         NOT NULL DEFAULT 0,
  leads_qualificados INT      NOT NULL DEFAULT 0,
  leads_exportados INT        NOT NULL DEFAULT 0,
  search_id       UUID        NOT NULL
);

CREATE INDEX idx_fato_funil_org_tempo ON fato_funil_leads (org_id, tempo_id);
```

### Seed de dim_tempo (gerar 3 anos)

```sql
-- Script para popular dim_tempo com 2024-01-01 até 2026-12-31
INSERT INTO dim_tempo (id, data, dia, mes, ano, trimestre, dia_semana, semana_ano, is_fim_semana)
SELECT
  TO_CHAR(d, 'YYYYMMDD')::INT,
  d::DATE,
  EXTRACT(DAY FROM d)::SMALLINT,
  EXTRACT(MONTH FROM d)::SMALLINT,
  EXTRACT(YEAR FROM d)::SMALLINT,
  EXTRACT(QUARTER FROM d)::SMALLINT,
  EXTRACT(DOW FROM d)::SMALLINT,
  EXTRACT(WEEK FROM d)::SMALLINT,
  EXTRACT(DOW FROM d) IN (0, 6)
FROM generate_series('2024-01-01'::date, '2026-12-31'::date, '1 day'::interval) d;
```

---

## 2. Backend (Go)

### 2.1 ETL Job

Localização: `api/internal/analytics/etl.go`

Rodar como goroutine periódica (a cada hora) ou via cron externo.

**Fluxo:**
```
1. Sincronizar dimensões (upsert):
   - dim_organizacao ← tb_organizations + tb_plans
   - dim_usuario     ← tb_users
   - dim_cnae        ← tb_cnaes

2. Processar fato_consumo_creditos:
   - SELECT transações não processadas ainda (usar coluna etl_processed_at)
   - INSERT INTO fato_consumo_creditos

3. Processar fato_funil_leads:
   - SELECT buscas status=done não processadas ainda
   - Para cada busca: determinar CNAE primário mais frequente nos resultados
   - INSERT INTO fato_funil_leads (leads_extraidos = result_count)
```

Adicionar coluna de controle no schema transacional:
```sql
ALTER TABLE tb_credit_transactions ADD COLUMN etl_processed_at TIMESTAMPTZ;
ALTER TABLE tb_searches ADD COLUMN etl_processed_at TIMESTAMPTZ;
```

### 2.2 Endpoints Analíticos

**GET /analytics/kpis**
```
Query:   ?period=7d | 30d | 90d | custom&from=2026-01-01&to=2026-03-31
Header:  Authorization: Bearer {token}
200:     {
           "period": "30d",
           "credits_consumed": 4500,
           "credits_purchased": 5000,
           "leads_extracted": 4500,
           "leads_qualified": 120,
           "leads_exported": 80,
           "conversion_rate": 0.0178,   -- leads_exportados / leads_extraidos
           "searches_count": 12
         }
```

**GET /analytics/daily-consumption**
```
Query:   ?period=30d
200:     [{ "date": "2026-05-15", "credits": 250, "leads": 250 }, ...]
```

**GET /analytics/top-cnaes**
```
Query:   ?period=30d&limit=10
200:     [{ "cnae_code": "4520-0/01", "description": "...", "leads": 1200 }, ...]
```

**GET /analytics/funnel**
```
Query:   ?period=30d
200:     {
           "stages": [
             { "name": "Extraídos",   "count": 4500 },
             { "name": "Qualificados","count": 120  },
             { "name": "Exportados",  "count": 80   }
           ]
         }
```

> Todas as queries analíticas leem do star schema — nunca do schema transacional diretamente.

---

## 3. Frontend (Next.js)

### Rota

```
/dashboard   ← rota principal pós-login
```

### 3.1 DashboardPage (`/app/dashboard/page.tsx`)

**Seletor de período** (no topo da página):
- Botões: "7 dias" | "30 dias" | "90 dias" | "Personalizado"
- Ao mudar período → refetch de todos os widgets

**Widget: KPI Cards (linha de 4 cards)**

| KPI | Valor | Delta |
|-----|-------|-------|
| Créditos consumidos | 4.500 | vs. período anterior |
| Leads extraídos | 4.500 | |
| Leads qualificados | 120 | |
| Taxa de conversão | 1,78% | |

**Widget: Gráfico de consumo diário**
- Linha temporal com `recharts LineChart`
- Eixo X: datas; Eixo Y: créditos/leads
- Duas linhas: créditos consumidos e leads extraídos

**Widget: Funil de conversão**
- `recharts FunnelChart` ou barras horizontais
- 3 etapas: Extraídos → Qualificados → Exportados
- Percentual de conversão entre etapas

**Widget: Top CNAEs**
- Tabela simples: CNAE, descrição, volume de leads
- Top 10 do período

**Widget: Buscas recentes**
- Últimas 5 buscas com status, filtros resumidos e resultado
- Link para abrir resultados completos

---

## 4. ETL Job Infra

- Rodar como goroutine com `ticker` de 1 hora dentro do servidor Go
- Log de execução com duração e registros processados
- Em erro: logar e continuar (não travar o servidor)
- Alternativa: expor `POST /internal/etl/run` para trigger manual (protegido por IP ou header secreto)

---

## 5. Testes e Validação

- [ ] Após executar ETL, `SELECT COUNT(*) FROM fato_consumo_creditos` > 0
- [ ] `GET /analytics/kpis?period=30d` retorna valores que batem com `SUM` direto em `tb_credit_transactions`
- [ ] `GET /analytics/funnel` mostra funil com leads_qualificados = 0 (correto — STEP-07 não implementado ainda)
- [ ] `GET /analytics/top-cnaes` retorna os CNAEs mais buscados do período
- [ ] `GET /analytics/daily-consumption` retorna array com uma entrada por dia do período
- [ ] Trocar período no frontend atualiza todos os widgets simultaneamente
- [ ] Dashboard carrega em < 1s (queries analíticas rápidas via star schema)
- [ ] Usuário de org A vê apenas dados da própria org no dashboard

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| ETL lento para grandes volumes | Processar em batches de 1000 registros; usar `etl_processed_at IS NULL` como cursor |
| KPIs divergem do schema transacional | Query de reconciliação periódica; alertar se diferença > 1% |
| dim_tempo sem datas futuras | Popular até 2030 no seed; adicionar cron anual para estender |
| Dashboard lento no carregamento inicial | Fazer requests dos widgets em paralelo no frontend (Promise.all) |
