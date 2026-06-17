# STEP-07 — Inteligência (IA)

## Objetivo
Implementar os três diferenciadores de IA do produto: score de qualificação por CNPJ, assistente de CNAE por prompt e gerador de mensagens — transformando o Vantaggio de um buscador em um qualificador inteligente.

## Pré-requisitos
- STEP-04 concluído (créditos para debitar nas operações de IA)
- STEP-03 concluído (resultados de busca para qualificar)
- `OPENAI_API_KEY` e/ou `GEMINI_API_KEY` no `.env`

## Escopo
- ✅ `tb_ai_qualifications` para armazenar resultados e controlar custos
- ✅ Score de qualificação 0-100 por empresa (10 créditos)
- ✅ Assistente de CNAE: prompt → lista de CNAEs relevantes
- ✅ Gerador de templates: contexto do lead → copy personalizada
- ✅ Abstração de provider (OpenAI / Gemini configurável)
- ✅ Badge de score na Lead Bank table
- ✅ Frontend: Intelligence page
- ❌ WhatsApp SDR autônomo (V3 — fora do escopo)
- ❌ Enriquecimento via scraping (adiado — alto custo operacional)

---

## 1. Banco de Dados

### Migration 008 — IA

```sql
-- migrations/008_intelligence.sql

CREATE TABLE tb_ai_qualifications (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  cnpj            VARCHAR(14) NOT NULL,
  org_id          UUID        NOT NULL REFERENCES tb_organizations(id),
  user_id         UUID        NOT NULL REFERENCES tb_users(id),
  score           SMALLINT    NOT NULL CHECK (score BETWEEN 0 AND 100),
  justification   TEXT        NOT NULL,
  prompt_used     TEXT        NOT NULL,
  model_used      VARCHAR(50) NOT NULL,
  tokens_input    INT         NOT NULL,
  tokens_output   INT         NOT NULL,
  raw_response    JSONB       NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Índice para buscar qualificação mais recente de um CNPJ por org
CREATE INDEX idx_ai_qual_cnpj_org ON tb_ai_qualifications (cnpj, org_id, created_at DESC);

-- Manter apenas a qualificação mais recente por CNPJ+org nos joins de Lead Bank
-- (score mais antigo que 30 dias deve ser re-qualificado)
```

---

## 2. Backend (Go)

### 2.1 Abstração de Provider

Localização: `api/internal/ia/provider.go`

```go
type LLMProvider interface {
    // Chat: prompt estruturado → resposta JSON
    Chat(ctx context.Context, systemPrompt, userPrompt string) (string, TokenUsage, error)

    // Embed: texto → vetor (usado internamente; já implementado no STEP-01)
    Embed(ctx context.Context, text string) ([]float32, error)
}

type TokenUsage struct {
    Input  int
    Output int
}

// Implementações concretas:
// - OpenAIProvider  (gpt-4o-mini por padrão)
// - GeminiProvider  (gemini-1.5-flash por padrão)

// Factory baseada em variável de ambiente:
// AI_PROVIDER=openai (padrão) | gemini
func NewProvider(cfg Config) LLMProvider
```

### 2.2 Endpoints

**POST /ia/qualify/:cnpj**
```
Header:  Authorization: Bearer {token}
Body:    {
           "context": {               // opcional: adicionar contexto extra
             "has_website": true,
             "has_instagram": true,
             "notes": "cliente indicado por João"
           }
         }
201:     {
           "qualification_id": "uuid",
           "cnpj": "...",
           "score": 87,
           "justification": "Empresa com capital acima de 500k, CNAE principal de alto valor...",
           "model": "gpt-4o-mini",
           "credits_used": 10
         }
402:     se créditos insuficientes
404:     se CNPJ não encontrado em tb_companies
```

**Regras de negócio:**
- Debitar 10 créditos ANTES de chamar a IA
- Se a IA falhar: estornar os créditos (transação com rollback)
- Se qualificação de mesmo CNPJ para mesma org já existir há < 30 dias: retornar a existente (sem debitar novamente)

**Prompt de qualificação (sistema):**
```
Você é um analista de prospecção B2B especializado no mercado brasileiro.
Analise os dados da empresa fornecida e retorne um score de 0 a 100 indicando
o potencial de conversão como lead, onde:
- 0-30: baixo potencial
- 31-60: potencial médio
- 61-80: alto potencial
- 81-100: potencial muito alto

Responda APENAS em JSON válido:
{"score": N, "justification": "texto em português explicando o score"}

Fatores positivos: capital social alto, situação ativa, CNAE de alto valor,
presença em grandes centros, empresa estabelecida (> 3 anos).
Fatores negativos: MEI, capital < 10k, situação inapta/baixada, setores saturados.
```

---

**POST /ia/cnae-assistant**
```
Header:  Authorization: Bearer {token}
Body:    { "prompt": "mecânicas de luxo em capitais do sudeste" }
200:     {
           "cnaes": [
             { "code": "4520-0/01", "description": "Manutenção e reparação...", "relevance": 0.95 },
             { "code": "4530-7/03", "description": "Comércio a varejo...",      "relevance": 0.72 }
           ],
           "explanation": "Para mecânicas de luxo, os CNAEs mais relevantes são...",
           "tokens_used": 350
         }
400:     se prompt vazio
```

**Prompt de sistema para CNAE Assistant:**
```
Você é um especialista na Classificação Nacional de Atividades Econômicas (CNAE) do Brasil.
Dado um perfil de empresa descrito pelo usuário, retorne os códigos CNAE mais relevantes.

Responda APENAS em JSON válido:
{
  "cnaes": [{ "code": "XXXX-X/XX", "description": "...", "relevance": 0.0-1.0 }],
  "explanation": "texto explicativo em português"
}

Use apenas códigos CNAE reais da tabela IBGE 2.3. Retorne no máximo 8 CNAEs.
```

---

**POST /ia/templates**
```
Header:  Authorization: Bearer {token}
Body:    {
           "cnpj": "12345678000195",
           "template_type": "whatsapp" | "email" | "linkedin",
           "tone": "formal" | "casual" | "consultivo",
           "custom_context": "empresa procura reduzir custos de manutenção"
         }
200:     {
           "copy": "Olá [Nome], vi que a [Razão Social] atua no setor de...",
           "subject": "Oportunidade para [Razão Social]",  // apenas para email
           "tokens_used": 280
         }
400:     se CNPJ não informado
```

**Prompt de sistema para Templates:**
```
Você é um especialista em prospecção comercial e copywriting para o mercado B2B brasileiro.
Crie uma mensagem de prospecção personalizada usando os dados da empresa fornecida.
A mensagem deve ser específica para o setor e porte da empresa.
Não use frases genéricas. Máximo de 200 palavras.

Responda APENAS em JSON válido:
{ "copy": "texto da mensagem", "subject": "assunto (apenas para email)" }
```

---

**GET /ia/qualifications**
```
Query:   ?cnpj=12345678000195 (opcional)
200:     [{ "cnpj", "score", "justification", "created_at" }]
```

---

### 2.3 Integração com Lead Bank

Modificar `GET /searches/:id` para incluir score quando disponível:
```json
{
  "results": [{
    "cnpj": "...",
    "razao_social": "...",
    "score": 87,            // null se não qualificado
    "score_age_days": 5     // dias desde a última qualificação
  }]
}
```

---

## 3. Frontend (Next.js)

### 3.1 Intelligence Page (`/app/intelligence/page.tsx`)

**Tab 1: Assistente de CNAE**
- Campo de texto grande: "Descreva o tipo de empresa que você quer encontrar..."
- Botão "Sugerir CNAEs"
- Loading skeleton durante processamento
- Resultado: cards de CNAE com código, descrição e barra de relevância (0-100%)
- Botão "Usar esses CNAEs na busca" → pré-preenche o formulário de busca estruturada

**Tab 2: Gerador de Templates**
- Input de CNPJ (com busca e autocomplete)
- Select de tipo: WhatsApp / Email / LinkedIn
- Select de tom: Formal / Casual / Consultivo
- Textarea opcional: "Contexto adicional"
- Botão "Gerar"
- Resultado: textarea com o copy gerado + botão "Copiar"

### 3.2 Badge de Score na Lead Bank Table

- Coluna "Score IA" na tabela de resultados de busca
- Se score disponível: badge colorido (🔴 < 40 | 🟡 40-70 | 🟢 > 70) + número
- Se não qualificado: botão "Qualificar" (10 créditos) que chama `POST /ia/qualify/:cnpj`
- Ao clicar "Qualificar": desabilitar botão durante call, atualizar badge ao receber resposta
- Ação em lote: "Qualificar selecionados" (mostrar custo total antes de confirmar)

### 3.3 Sidebar: link "Inteligência" com ícone de sparkles

---

## 4. Integrações / Infra

- `AI_PROVIDER=openai` ou `gemini` — provider padrão
- `AI_CHAT_MODEL=gpt-4o-mini` — modelo de chat
- `AI_EMBEDDING_MODEL=text-embedding-3-small` — modelo de embedding
- Timeout de 30s para calls de IA
- Retry automático em caso de rate limit (429): backoff exponencial até 3 tentativas

---

## 5. Testes e Validação

- [ ] `POST /ia/cnae-assistant` com "padarias artesanais premium" retorna CNAEs plausíveis (1518-5/01, 5611-2/03...)
- [ ] `POST /ia/qualify/:cnpj` para empresa ativa com capital > 500k retorna score > 60
- [ ] `POST /ia/qualify/:cnpj` deduz 10 créditos do saldo
- [ ] Segunda qualificação do mesmo CNPJ em < 30 dias retorna resultado em cache (sem debitar)
- [ ] Qualificação com créditos insuficientes retorna 402 sem chamar a IA
- [ ] `POST /ia/templates` com cnpj + tipo whatsapp retorna copy personalizada com nome da empresa
- [ ] Badge de score aparece na Lead Bank table após qualificação
- [ ] Qualificação em lote de 5 empresas debita 50 créditos e exibe todos os badges
- [ ] Falha na API OpenAI → créditos não são debitados (rollback)

---

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Resposta da IA não é JSON válido | Parsear com fallback; retry com instrução mais explícita se falhar |
| Custo de tokens não controlado | Monitorar `tokens_input + tokens_output` em `tb_ai_qualifications`; alertar se custo/dia exceder threshold |
| Score inconsistente para mesma empresa | Temperatura baixa no modelo (0.2) + prompt determinístico; aceitar variação de ±5 pontos |
| Rate limit OpenAI em qualificações em lote | Semáforo no backend: máximo 5 calls simultâneas de IA por org |
| Provider indisponível | Fallback automático para provider alternativo (OpenAI → Gemini) se configurado |
