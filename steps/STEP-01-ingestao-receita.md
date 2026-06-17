# STEP-01 — Ingestão Receita Federal

## Objetivo
Importar os CSVs do CNPJ Aberto para PostgreSQL e gerar embeddings vetoriais por empresa — criando a base de dados que alimenta todas as buscas do produto.

## Pré-requisitos
- STEP-00 concluído (Postgres rodando com imagem pgvector)
- Arquivos CSV do CNPJ Aberto já extraídos na raiz do projeto (fornecidos pelo dev)
- `OPENAI_API_KEY` ou `GEMINI_API_KEY` configurados no `.env`

## Escopo
- ✅ DDL das tabelas de dados da Receita Federal
- ✅ Extensão pgvector habilitada
- ✅ Script CLI de importação em Go (idempotente)
- ✅ Geração de embeddings por empresa
- ✅ Índices para busca estruturada e vetorial
- ❌ API HTTP (isso é STEP-02)
- ❌ Autenticação ou multi-tenancy
- ❌ Download ou parsing dos CSVs originais

---

## 1. Banco de Dados

### 1.1 Migration 001 — Extensão e tabelas base

```sql
-- migrations/001_receita_federal.sql

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela de CNAEs (domínio)
CREATE TABLE tb_cnaes (
  code        VARCHAR(10) PRIMARY KEY,  -- ex: "4520-0/01"
  description TEXT NOT NULL
);

-- Tabela principal de empresas (particionada por UF)
CREATE TABLE tb_companies (
  cnpj                VARCHAR(14)  NOT NULL,
  razao_social        TEXT         NOT NULL,
  nome_fantasia       TEXT,
  situacao_cadastral  SMALLINT     NOT NULL,  -- 2=ativa, 4=inapta, 8=baixada...
  data_situacao       DATE,
  natureza_juridica   VARCHAR(10),
  logradouro          TEXT,
  numero              VARCHAR(10),
  complemento         TEXT,
  bairro              TEXT,
  cep                 VARCHAR(8),
  uf                  CHAR(2)      NOT NULL,
  municipio_id        INT,
  municipio_nome      TEXT,
  ddd_telefone1       VARCHAR(4),
  telefone1           VARCHAR(10),
  email               TEXT,
  capital_social      NUMERIC(18,2),
  porte               SMALLINT,             -- 1=ME, 3=EPP, 5=demais
  opcao_simples       BOOLEAN,
  data_inicio         DATE,
  embedding           vector(1536),         -- OpenAI text-embedding-3-small
  embedding_updated_at TIMESTAMPTZ,
  PRIMARY KEY (cnpj, uf)
) PARTITION BY LIST (uf);

-- Criar partições por UF (uma por estado)
CREATE TABLE tb_companies_ac PARTITION OF tb_companies FOR VALUES IN ('AC');
CREATE TABLE tb_companies_al PARTITION OF tb_companies FOR VALUES IN ('AL');
CREATE TABLE tb_companies_am PARTITION OF tb_companies FOR VALUES IN ('AM');
CREATE TABLE tb_companies_ap PARTITION OF tb_companies FOR VALUES IN ('AP');
CREATE TABLE tb_companies_ba PARTITION OF tb_companies FOR VALUES IN ('BA');
CREATE TABLE tb_companies_ce PARTITION OF tb_companies FOR VALUES IN ('CE');
CREATE TABLE tb_companies_df PARTITION OF tb_companies FOR VALUES IN ('DF');
CREATE TABLE tb_companies_es PARTITION OF tb_companies FOR VALUES IN ('ES');
CREATE TABLE tb_companies_go PARTITION OF tb_companies FOR VALUES IN ('GO');
CREATE TABLE tb_companies_ma PARTITION OF tb_companies FOR VALUES IN ('MA');
CREATE TABLE tb_companies_mg PARTITION OF tb_companies FOR VALUES IN ('MG');
CREATE TABLE tb_companies_ms PARTITION OF tb_companies FOR VALUES IN ('MS');
CREATE TABLE tb_companies_mt PARTITION OF tb_companies FOR VALUES IN ('MT');
CREATE TABLE tb_companies_pa PARTITION OF tb_companies FOR VALUES IN ('PA');
CREATE TABLE tb_companies_pb PARTITION OF tb_companies FOR VALUES IN ('PB');
CREATE TABLE tb_companies_pe PARTITION OF tb_companies FOR VALUES IN ('PE');
CREATE TABLE tb_companies_pi PARTITION OF tb_companies FOR VALUES IN ('PI');
CREATE TABLE tb_companies_pr PARTITION OF tb_companies FOR VALUES IN ('PR');
CREATE TABLE tb_companies_rj PARTITION OF tb_companies FOR VALUES IN ('RJ');
CREATE TABLE tb_companies_rn PARTITION OF tb_companies FOR VALUES IN ('RN');
CREATE TABLE tb_companies_ro PARTITION OF tb_companies FOR VALUES IN ('RO');
CREATE TABLE tb_companies_rr PARTITION OF tb_companies FOR VALUES IN ('RR');
CREATE TABLE tb_companies_rs PARTITION OF tb_companies FOR VALUES IN ('RS');
CREATE TABLE tb_companies_sc PARTITION OF tb_companies FOR VALUES IN ('SC');
CREATE TABLE tb_companies_se PARTITION OF tb_companies FOR VALUES IN ('SE');
CREATE TABLE tb_companies_sp PARTITION OF tb_companies FOR VALUES IN ('SP');
CREATE TABLE tb_companies_to PARTITION OF tb_companies FOR VALUES IN ('TO');

-- Associação CNPJ <> CNAE
CREATE TABLE tb_company_cnaes (
  cnpj         VARCHAR(14) NOT NULL,
  cnae_code    VARCHAR(10) NOT NULL REFERENCES tb_cnaes(code),
  is_primary   BOOLEAN     NOT NULL DEFAULT false,
  PRIMARY KEY (cnpj, cnae_code)
);

-- Sócios/Parceiros
CREATE TABLE tb_partners (
  id              BIGSERIAL    PRIMARY KEY,
  cnpj            VARCHAR(14)  NOT NULL,
  nome_socio      TEXT         NOT NULL,
  cpf_cnpj_socio  VARCHAR(14),
  qualificacao    SMALLINT,
  data_entrada    DATE,
  pais            VARCHAR(3),
  faixa_etaria    SMALLINT
);
```

### 1.2 Migration 002 — Índices

```sql
-- migrations/002_indices.sql

-- Índices estruturados
CREATE INDEX idx_companies_uf         ON tb_companies (uf);
CREATE INDEX idx_companies_municipio  ON tb_companies (municipio_id);
CREATE INDEX idx_companies_situacao   ON tb_companies (situacao_cadastral);
CREATE INDEX idx_companies_capital    ON tb_companies (capital_social);
CREATE INDEX idx_companies_porte      ON tb_companies (porte);

-- Índice GIN para busca por CNAE
CREATE INDEX idx_company_cnaes_code ON tb_company_cnaes (cnae_code);
CREATE INDEX idx_company_cnaes_cnpj ON tb_company_cnaes (cnpj);

-- Índice vetorial HNSW (criado APÓS a ingestão para ser mais rápido)
-- Executar manualmente depois que todos os embeddings estiverem prontos:
-- CREATE INDEX idx_companies_embedding ON tb_companies
--   USING hnsw (embedding vector_cosine_ops)
--   WITH (m = 16, ef_construction = 64);
```

> **Nota:** criar o índice HNSW **depois** da ingestão completa é muito mais eficiente do que mantê-lo durante os inserts.

---

## 2. Script de Importação (Go CLI)

Localização: `api/cmd/ingestion/main.go`

### Fluxo do script

```
CSVs na raiz do projeto
        │
        ▼
  Parse linha a linha (encoding ISO-8859-1 → UTF-8)
        │
        ▼
  Batch INSERT de 1000 registros (ON CONFLICT DO UPDATE)
  em tb_companies, tb_cnaes, tb_company_cnaes, tb_partners
        │
        ▼
  Para cada empresa SEM embedding:
    - Montar texto: "{razao_social} | {cnae_desc} | {municipio} {uf} | {situacao}"
    - Chamar OpenAI Embeddings API (batch de 100 textos por request)
    - UPDATE tb_companies SET embedding = $vec, embedding_updated_at = now()
        │
        ▼
  Criar índice HNSW ao final
```

### Assinatura da CLI

```bash
# Importar todos os arquivos CSV de um diretório
go run ./api/cmd/ingestion \
  --csv-dir ./data \
  --batch-size 1000 \
  --embedding-batch 100 \
  --skip-embeddings=false
```

Flags:
- `--csv-dir` — pasta com os arquivos CSV (padrão: `./data`)
- `--batch-size` — linhas por batch de INSERT (padrão: 1000)
- `--embedding-batch` — empresas por chamada de embedding API (padrão: 100)
- `--skip-embeddings` — pular geração de embeddings, útil para testar a importação primeiro

### Idempotência

Usar `ON CONFLICT (cnpj, uf) DO UPDATE SET ...` em todos os inserts. Rodar o script novamente atualiza registros existentes sem duplicar.

### Formato do texto para embedding

```
{razao_social} | {cnae_primario_descricao} | {municipio_nome} {uf} | situacao:{situacao_cadastral} capital:{capital_social}
```

Exemplo:
```
MECANICA SILVA LTDA | Manutenção e reparação de automóveis e camionetes | São Paulo SP | situacao:2 capital:150000.00
```

---

## 3. Backend (Go)

Neste step não há endpoint HTTP. O único entregável Go é a CLI de importação.

Estrutura interna sugerida:

```
api/cmd/ingestion/
├── main.go           # parse de flags, orquestração
├── csv_parser.go     # leitura e normalização dos CSVs
├── importer.go       # batch inserts no Postgres
└── embedder.go       # chamadas à API de embeddings
```

---

## 4. Frontend (Next.js)

Nenhum componente frontend neste step.

---

## 5. Integrações / Infra

- **Fonte de dados:** arquivos CSV do CNPJ Aberto (CNPJ, ESTABELECIMENTOS, SOCIOS, SIMPLES, CNAES)
- **Encoding:** os CSVs da Receita Federal usam ISO-8859-1 — converter para UTF-8 durante parsing
- **Separador:** ponto e vírgula (`;`)
- **Volume estimado:** ~60M de estabelecimentos, ~22GB descomprimido
- **Tempo estimado de ingestão:** 2-4h dependendo do hardware
- **Tempo estimado de embeddings:** com batch de 100 e rate limit da OpenAI, ~8-12h para todos os registros; considerar rodar com `--skip-embeddings=true` primeiro e gerar embeddings incrementalmente

---

## 6. Testes e Validação

- [ ] Migration 001 e 002 executam sem erro
- [ ] `SELECT COUNT(*) FROM tb_companies` retorna valor > 0 após importação
- [ ] `SELECT COUNT(*) FROM tb_company_cnaes WHERE is_primary = true` ≈ total de empresas
- [ ] Query estruturada retorna resultados corretos:
  ```sql
  SELECT c.razao_social, c.municipio_nome
  FROM tb_companies c
  JOIN tb_company_cnaes cc ON cc.cnpj = c.cnpj
  WHERE cc.cnae_code = '4520-0/01' AND c.uf = 'SP'
    AND c.capital_social >= 100000 AND c.situacao_cadastral = 2
  LIMIT 10;
  ```
- [ ] Query vetorial retorna resultados semanticamente relevantes:
  ```sql
  SELECT razao_social, municipio_nome, uf,
         embedding <=> '[...]'::vector AS distance
  FROM tb_companies
  WHERE uf = 'SP' AND situacao_cadastral = 2
  ORDER BY distance
  LIMIT 10;
  ```
- [ ] Query acima retorna em < 300ms com índice HNSW criado
- [ ] Rodar o script duas vezes → `COUNT(*)` permanece o mesmo (idempotência)

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| CSVs com encoding ISO-8859-1 quebrando caracteres | Usar `golang.org/x/text/encoding/charmap` para conversão explícita |
| Rate limit na API de embeddings | Implementar retry com backoff exponencial + checkpoint de progresso em arquivo |
| Ingestão interrompida no meio | Script deve registrar última linha processada por arquivo para retomar |
| Índice HNSW consome RAM excessiva durante build | Criar o índice fora do horário de uso; ajustar `ef_construction` para 32 se necessário |
| Volume de dados maior do que o esperado | Particionamento por UF isola o problema — pode-se importar estado a estado |
