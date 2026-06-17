-- +goose Up

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela de CNAEs (domínio — código de 7 dígitos do CNPJ Aberto)
CREATE TABLE IF NOT EXISTS tb_cnaes (
    code        VARCHAR(7) PRIMARY KEY,
    description TEXT       NOT NULL
);

-- Tabela principal de empresas (particionada por UF)
CREATE TABLE IF NOT EXISTS tb_companies (
    cnpj                 VARCHAR(14)   NOT NULL,
    razao_social         TEXT          NOT NULL DEFAULT '',
    nome_fantasia        TEXT,
    situacao_cadastral   SMALLINT      NOT NULL DEFAULT 0,
    data_situacao        DATE,
    natureza_juridica    VARCHAR(4),
    logradouro           TEXT,
    numero               VARCHAR(10),
    complemento          TEXT,
    bairro               TEXT,
    cep                  VARCHAR(8),
    uf                   CHAR(2)       NOT NULL,
    municipio_id         INT,
    municipio_nome       TEXT,
    ddd_telefone1        VARCHAR(4),
    telefone1            VARCHAR(10),
    email                TEXT,
    capital_social       NUMERIC(18,2),
    porte                SMALLINT,
    opcao_simples        BOOLEAN,
    data_inicio          DATE,
    embedding            vector(1536),
    embedding_updated_at TIMESTAMPTZ,
    PRIMARY KEY (cnpj, uf)
) PARTITION BY LIST (uf);

CREATE TABLE IF NOT EXISTS tb_companies_ac PARTITION OF tb_companies FOR VALUES IN ('AC');
CREATE TABLE IF NOT EXISTS tb_companies_al PARTITION OF tb_companies FOR VALUES IN ('AL');
CREATE TABLE IF NOT EXISTS tb_companies_am PARTITION OF tb_companies FOR VALUES IN ('AM');
CREATE TABLE IF NOT EXISTS tb_companies_ap PARTITION OF tb_companies FOR VALUES IN ('AP');
CREATE TABLE IF NOT EXISTS tb_companies_ba PARTITION OF tb_companies FOR VALUES IN ('BA');
CREATE TABLE IF NOT EXISTS tb_companies_ce PARTITION OF tb_companies FOR VALUES IN ('CE');
CREATE TABLE IF NOT EXISTS tb_companies_df PARTITION OF tb_companies FOR VALUES IN ('DF');
CREATE TABLE IF NOT EXISTS tb_companies_es PARTITION OF tb_companies FOR VALUES IN ('ES');
CREATE TABLE IF NOT EXISTS tb_companies_go PARTITION OF tb_companies FOR VALUES IN ('GO');
CREATE TABLE IF NOT EXISTS tb_companies_ma PARTITION OF tb_companies FOR VALUES IN ('MA');
CREATE TABLE IF NOT EXISTS tb_companies_mg PARTITION OF tb_companies FOR VALUES IN ('MG');
CREATE TABLE IF NOT EXISTS tb_companies_ms PARTITION OF tb_companies FOR VALUES IN ('MS');
CREATE TABLE IF NOT EXISTS tb_companies_mt PARTITION OF tb_companies FOR VALUES IN ('MT');
CREATE TABLE IF NOT EXISTS tb_companies_pa PARTITION OF tb_companies FOR VALUES IN ('PA');
CREATE TABLE IF NOT EXISTS tb_companies_pb PARTITION OF tb_companies FOR VALUES IN ('PB');
CREATE TABLE IF NOT EXISTS tb_companies_pe PARTITION OF tb_companies FOR VALUES IN ('PE');
CREATE TABLE IF NOT EXISTS tb_companies_pi PARTITION OF tb_companies FOR VALUES IN ('PI');
CREATE TABLE IF NOT EXISTS tb_companies_pr PARTITION OF tb_companies FOR VALUES IN ('PR');
CREATE TABLE IF NOT EXISTS tb_companies_rj PARTITION OF tb_companies FOR VALUES IN ('RJ');
CREATE TABLE IF NOT EXISTS tb_companies_rn PARTITION OF tb_companies FOR VALUES IN ('RN');
CREATE TABLE IF NOT EXISTS tb_companies_ro PARTITION OF tb_companies FOR VALUES IN ('RO');
CREATE TABLE IF NOT EXISTS tb_companies_rr PARTITION OF tb_companies FOR VALUES IN ('RR');
CREATE TABLE IF NOT EXISTS tb_companies_rs PARTITION OF tb_companies FOR VALUES IN ('RS');
CREATE TABLE IF NOT EXISTS tb_companies_sc PARTITION OF tb_companies FOR VALUES IN ('SC');
CREATE TABLE IF NOT EXISTS tb_companies_se PARTITION OF tb_companies FOR VALUES IN ('SE');
CREATE TABLE IF NOT EXISTS tb_companies_sp PARTITION OF tb_companies FOR VALUES IN ('SP');
CREATE TABLE IF NOT EXISTS tb_companies_to PARTITION OF tb_companies FOR VALUES IN ('TO');
CREATE TABLE IF NOT EXISTS tb_companies_ex PARTITION OF tb_companies FOR VALUES IN ('EX');
CREATE TABLE IF NOT EXISTS tb_companies_other PARTITION OF tb_companies DEFAULT;

-- Associação CNPJ <> CNAE (sem FK em cnae_code para tolerar dados da Receita com CNAEs obsoletos)
CREATE TABLE IF NOT EXISTS tb_company_cnaes (
    cnpj       VARCHAR(14) NOT NULL,
    cnae_code  VARCHAR(7)  NOT NULL,
    is_primary BOOLEAN     NOT NULL DEFAULT false,
    PRIMARY KEY (cnpj, cnae_code)
);

-- Sócios (cnpj_basico = 8 dígitos conforme arquivo SOCIOCSV)
CREATE TABLE IF NOT EXISTS tb_partners (
    id             BIGSERIAL  PRIMARY KEY,
    cnpj_basico    VARCHAR(8) NOT NULL,
    nome_socio     TEXT       NOT NULL DEFAULT '',
    cpf_cnpj_socio VARCHAR(14),
    qualificacao   SMALLINT,
    data_entrada   DATE,
    pais           VARCHAR(3),
    faixa_etaria   SMALLINT
);

-- +goose Down

DROP TABLE IF EXISTS tb_partners;
DROP TABLE IF EXISTS tb_company_cnaes;
DROP TABLE IF EXISTS tb_companies;
DROP TABLE IF EXISTS tb_cnaes;
