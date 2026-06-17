# Skill: Skill Creator

> Guia para criação de skills eficazes para agentes de IA.

## Objetivo

Orientar a criação de novas skills para agentes de IA neste repositório, garantindo que sejam claras, acionáveis e eficazes para os casos de uso pretendidos.

## O que é uma Skill?

Uma **skill** é um documento instrucional que ensina um agente de IA a executar um conjunto específico de tarefas dentro de um contexto bem definido. Uma boa skill:

- Tem **escopo claro e delimitado**
- Fornece **instruções passo a passo** acionáveis
- Inclui **exemplos concretos** e templates
- Define **quando usar** e **quando não usar**
- É **independente** — pode ser lida e executada sem contexto adicional

## Estrutura de uma Skill

Toda skill deve seguir esta estrutura:

```markdown
# Skill: <Nome da Skill>

> <Tagline de uma linha descrevendo o propósito>

## Objetivo
<O que esta skill faz e por que é útil>

## Quando Usar
<Lista de cenários onde esta skill é aplicável>

## [Seções específicas da skill]
<Instruções, processos, templates, exemplos>

## Boas Práticas
<Dicas e armadilhas comuns>

## Referências (opcional)
<Links e recursos relevantes>
```

## Localização das Skills

As skills ficam em `.github/skills/<nome-da-skill>/SKILL.md`.

```
.github/
└── skills/
    ├── domain-analysis/
    │   └── SKILL.md
    ├── skill-creator/
    │   └── SKILL.md
    └── <nova-skill>/
        └── SKILL.md
```

## Processo de Criação

### Passo 1: Defina o Problema

Responda antes de escrever:

1. **Qual problema esta skill resolve?**
2. **Quem vai usar esta skill?** (agente de IA, desenvolvedor, etc.)
3. **Qual é o resultado esperado após executar a skill?**
4. **Esta skill sobrepõe com alguma skill existente?**

Se houver sobreposição com skill existente, considere expandir a skill existente ao invés de criar uma nova.

### Passo 2: Defina o Escopo

Uma skill deve ter **escopo único e coeso**. Exemplos:

| ✅ Bom Escopo | ❌ Escopo Ruim |
|--------------|---------------|
| Análise de domínio DDD | "Fazer tudo relacionado a DDD" |
| Criação de migration TypeORM | "Gerenciar banco de dados" |
| Revisão de Pull Request | "Desenvolvimento completo de feature" |

### Passo 3: Escreva as Instruções

**Princípios para instruções eficazes:**

1. **Seja específico**: "Execute `npm run migration:generate -- --name=CreateBillingTable`" em vez de "Gere uma migration"
2. **Use listas numeradas** para sequências obrigatórias
3. **Use listas com marcadores** para opções ou itens sem ordem
4. **Inclua exemplos de código** sempre que possível
5. **Antecipe erros comuns** e como resolvê-los
6. **Defina critérios de sucesso** — como saber que a tarefa foi concluída corretamente

### Passo 4: Crie Templates

Inclua templates prontos que o agente pode usar diretamente:

```markdown
## Template: <Nome>

\`\`\`typescript
// Código template aqui
\`\`\`
```

### Passo 5: Valide a Skill

Antes de finalizar, verifique:

- [ ] O objetivo está claro em uma leitura rápida
- [ ] As instruções são suficientemente detalhadas para execução autônoma
- [ ] Os exemplos cobrem os casos de uso mais comuns
- [ ] A skill não tem dependências implícitas não documentadas
- [ ] O formato é consistente com as outras skills do repositório

## Boas Práticas

### O que fazer ✅

- **Escreva em português brasileiro** — padrão do time
- **Use exemplos do próprio repositório** para ilustrar conceitos
- **Seja imperativo** — "Crie", "Execute", "Verifique" em vez de "Você pode criar"
- **Referencie arquivos específicos** quando relevante (ex: `src/modules/billing/`)
- **Mantenha as skills atualizadas** quando o projeto evolui

### O que evitar ❌

- **Não seja vago** — "faça o correto" não instrui ninguém
- **Não duplique conteúdo** do AGENTS.md — referencie ao invés de copiar
- **Não crie skills muito longas** — se estiver grande demais, divida em skills menores
- **Não presuma conhecimento implícito** — seja explícito sobre contexto e requisitos

## Exemplo de Skill Bem Estruturada

```markdown
# Skill: Criar Migration TypeORM

> Guia para criação de migrations de banco de dados no padrão do projeto.

## Objetivo
Criar migrations TypeORM seguindo as convenções do projeto para alterações
seguras e rastreáveis no esquema do banco de dados.

## Quando Usar
- Adicionar nova tabela ao banco de dados
- Adicionar ou remover colunas de tabela existente
- Criar índices ou constraints

## Processo

### 1. Gere a migration
\`\`\`bash
NAME=CreateBillingTable npm run migration:generate
\`\`\`

### 2. Revise o arquivo gerado
Verifique em `src/shared/database/migrations/` se o SQL gerado está correto.

### 3. Execute localmente
\`\`\`bash
npm run migration:run
\`\`\`

## Boas Práticas
- Nomeie migrations descritivamente: `AddDueDateToBilling`, `CreatePaymentMethodTable`
- Nunca altere uma migration já executada em produção
- Sempre implemente o método `down()` para rollback
```

## Referências

- [agents.md](https://agents.md/) — Formato e boas práticas para AGENTS.md
- [`AGENTS.md`](../../AGENTS.md) — Instruções gerais para agentes neste repositório
- [`docs/archtecture-guideline.md`](../../docs/archtecture-guideline.md) — Diretrizes de arquitetura