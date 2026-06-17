# Vantaggio Prospect - PLAN.MD

## Visão do Produto
**Nome provisório:** Vantaggio PrHunter
**Proposta:** Transformar milhões de empresas da Receita Federal em oportunidades comerciais qualificadas por IA.
*O CRM é apenas um destino.*

---

## 1. Arquitetura
Como Engenheiro de Software liderando essa plataforma, a escolha de tecnologias garante escalabilidade para processamento de alto volume:
* **Frontend:** Next.js (SSR e SEO, além de uma interface limpa)
* **API:** Go (Alta concorrência e baixo consumo para bater na base da Receita)
* **Banco de Dados:** PostgreSQL (Tabelas relacionais, índices JSONB e particionamento para milhões de CNPJs)
* **Fila:** Redis (Gestão eficiente do processamento assíncrono de pesquisas e enriquecimento)
* **IA:** Gemini e OpenAI (Modelos para interpretação, templates e scoring)
* **CRM:** Chatwoot (Destino principal da V1, ja está implementado)
* **Mensageria:** Evolution API (Disparos de WhatsApp na V3)
* **Storage:** S3 (Armazenamento de relatórios CSV e exportações)

---

## 2. Banco de Dados
**Principais Tabelas:**
Proposta de Estrutura Robusta e Definitiva
1. Schema Transacional (Operação do Sistema SaaS)
Aqui focamos em restrições (Foreign Keys), normalização e "Soft Deletes" para não perder histórico.

Acesso e Multi-Tenancy:

tb_resellers

tb_organizations (com reseller_id)

tb_plans

tb_users (com organization_id, role, e flags is_active, deleted_at)

Core Receita Federal (Alta Performance e Índices):

tb_companies (Tabela massiva, particionada por estado ou ano de abertura)

tb_cnaes (Tabela de domínio com a lista de todos os CNAEs do IBGE)

tb_company_cnaes (Tabela associativa cruzando CNPJ e CNAE, indicando qual é o primário)

tb_partners

Inteligência e Enriquecimento:

tb_contacts (Pode ser uma tabela 1:1 com tb_companies para evitar joins caros)

tb_ai_qualifications (Substituindo ai_scores para guardar não só o score, mas o prompt usado, tokens consumidos e o JSON de retorno).

Motor de Pesquisa e CRM:

tb_searches (Guarda o payload JSON dos filtros aplicados)

tb_search_results (Associativa CNPJ <> Search ID)

tb_crm_integrations (Guarda as credenciais criptografadas do Chatwoot/Hubspot de cada organização)

tb_export_queue (Fila de transbordo registrando sucesso/falha do envio).

2. Schema Analítico (Modelagem Dimensional - Fato e Dimensão)
Estas tabelas (ou views materializadas atualizadas via cron/triggers) são construídas para responder perguntas de negócio instantaneamente.

As Dimensões (O Contexto - "Quem, Onde, O Quê"):

dim_tempo (Granularidade diária: Data, Dia da Semana, Mês, Trimestre, Ano. Essencial para gráficos de evolução).

dim_organizacao (Consolida dados do tenant: Nome, Status, Nome da Revenda, Nome do Plano).

dim_usuario (Nome e cargo de quem executou a ação).

dim_cnae (Apenas a descrição do setor econômico).

dim_geografia (Cidade, Estado, Região).

Os Fatos (Os Eventos Numéricos e Métricas - "Quanto"):

fato_consumo_creditos: Cada linha é um evento financeiro.

Chaves: FK Tempo, FK Organizacao, FK Usuario.

Métricas: Quantidade de créditos debitados, Tipo de Operação (Enriquecimento, Pesquisa, Exportação).

fato_funil_leads: Analisa a conversão da máquina.

Chaves: FK Tempo, FK Organizacao, FK Cnae, FK Geografia.

Métricas: Qtd_Leads_Encontrados, Qtd_Leads_Qualificados_IA, Qtd_Leads_Exportados_CRM, Custo_Medio_Creditos_Por_Lead.
---

## 3. APIs
Endpoints principais da engine em Go:
* **Auth:** JWT com escopo de permissões (Admin, Gestor, Operador, Revendedor).
* **Engine de Pesquisa:** * Recebe carga com até 5 CNAEs, localidade, capital, situação.
    * Fila via Redis para extrair a base de leads sem travar o client.
* **Consultas Avulsas (Avançada):**
    * Endpoint para busca síncrona por CPF/CNPJ, descontando os 10 créditos na hora.
* **Inteligência:**
    * `POST /ia/cnae-assistant`: Prompt -> Retorna CNAEs correspondentes.
    * `POST /ia/templates`: Contexto do lead -> Gera copys adaptadas.
* **Exportação:** Disparo de payloads webhooks formatados para CRMs.

---

## 4. Frontend
Aplicações e componentes no Next.js:
* **Sidebar & Topbar:** Navegação ágil e modularizada por permissão.
* **Dashboard:** Visão de negócios. KPIs (Saldo de créditos, Leads extraídos, Qualificados, Taxas de transferência, ROI). Widgets inferiores (Pesquisas recentes, Top CNAEs, Consumo diário, Funil).
* **Pesquisas (Core):** Construtor de queries intuitivo. Selects otimizados para estados/municípios.
* **Banco de Leads:** Tabela de alta performance (virtualização) com paginação no banco. Ações em lote: "Enriquecer", "Exportar", "Enviar ao CRM".
* **Módulo de Inteligência:** Interfaces de chat/formulário para o Assistente CNAE e Templates.
* **Painel de Revenda:** Visão administrativa de tenants (Planos, Clientes, Distribuição de Créditos).

---

## 5. Roadmap V1 / V2 / V3
*Essa progressão permite lançar rápido e ir sofisticando o produto.*

* **V1 (Core de Dados - Paridade com o Mercado):**
    Busca rápida na base Receita Federal, filtros estruturados, controle de acesso e cobrança via créditos, exportação básica de dados brutos e enriquecidos.
* **V2 (IA Qualificadora - O Diferencial):**
    Algoritmo Preditivo de Score (Ex: Mecânica em SP, 500k de capital, com Instagram/Site -> Score 92/100). Implementação do Assistente CNAE e Gerador de Mensagens.
* **V3 (Agente SDR Autônomo):**
    Fluxo ponta a ponta sem intervenção: Encontra Lead -> Qualifica -> Gera Copy -> Dispara via WhatsApp (Evolution API) -> Responde Dúvidas Primárias -> Transfere para Humano no CRM ao detectar interesse.

---

## 6. Modelo de Créditos
*Sistema rigoroso de consumo (Ledger).*
* **Pesquisa CNAE Base:** 1 crédito por lead retornado.
* **Enriquecimento Opcional:** 2 créditos por lead com sucesso (Scraping/APIs Externas).
* **Exportação ao CRM:** 1 crédito.
* **Consulta Avançada (Individual):** 10 créditos.

**Sugestão de Estrutura de Assinatura:**
* Starter (1.000 créditos) / Pro (10.000 créditos) / Enterprise (100.000 créditos).

---

## 7. Multi-tenancy & Usuários
Arquitetura construída para B2B e distribuição White-Label:
* **Hierarquia:** Master -> Revenda -> Cliente -> Usuários.
* **RBAC (Perfis):**
    * `Admin`: Acesso total à infra e criação de revendas.
    * `Revendedor`: Cria os próprios clientes e planos, distribuindo os créditos que comprou do Master.
    * `Gestor`: Administração da conta do cliente e integrações.
    * `Operador`: Focado apenas em pesquisar e exportar.

---

## 8. Integrações
* **CRM (Destino):** Chatwoot inicialmente. Depois Hubspot, Pipedrive, RD Station, Kommo e Salesforce. Ferramentas de automação como N8N ou Pipefy também podem ser conectadas via webhook para orquestrar dados comerciais ou atualizar fluxos internos com base nos leads exportados.
* **Comunicação:** Evolution API para escalar e gerenciar sessões do WhatsApp na etapa de SDR Autônomo.
* **Inteligência Artificial:** OpenAI e Gemini como motores de NLP e processamento preditivo.

---

## 9. Ordem de Implementação (Plano de Voo)
*Focando em construir o MVP (Produto Vendável) o mais rápido possível:*

1.  **Base Receita Federal:** Ingestão de dados e parse do CNPJ Aberto para o Postgres.
2.  **Banco de Leads:** API Go lendo as tabelas com alta performance.
3.  **Pesquisas:** Frontend (Next.js) conversando com a engine de busca.
4.  **Créditos:** Trava financeira para garantir a monetização desde o dia 1.
    *(Aqui já existe um produto pronto para os primeiros clientes).*
5.  **Dashboard:** Criação do apelo visual e métricas de ROI.
6.  **CRM Export:** Fechar o ciclo enviando os leads diretamente para a ponta.
7.  **Inteligência:** Iniciar o desenvolvimento dos diferenciais (Score, CNAE, Templates).
8.  **Revenda:** Escalar a distribuição de vendas através de parceiros e multi-tenancy.


Paleta sugerida
#080706
#FFFFFF

#E8621A
#F07D35

#DC5A14
#C8500F
#B43C0A
#8C2805

#737373
#1F1F1F
#121212

pode puxar pra cores um pouco mais vivas, mas sem fugir muito do padrão vantaggio.