---
name: skill-creator
description: Guia para criar AI agent skills eficazes. Use quando usuários quiserem criar uma nova skill (ou atualizar uma skill existente) que estenda as capacidades de um AI agent com conhecimento especializado, workflows ou integrações com ferramentas. Funciona com qualquer agent que suporte o formato SKILL.md (Claude Code, Cursor, Roo, Cline, Windsurf, etc.). Dispara em "create skill", "new skill", "package knowledge", "skill for".
---

# Skill Creator

Esta skill fornece orientações para criar skills eficazes e agnósticas de agent.

## Sobre Skills

Skills são pacotes modulares e autocontidos que estendem as capacidades de AI agents ao fornecer conhecimento especializado, workflows e ferramentas. Pense nelas como “guias de onboarding” para domínios ou tarefas específicas — elas transformam um agent de uso geral em um agent especializado, equipado com conhecimento procedural.

### O que Skills oferecem

1. **Workflows especializados** – Procedimentos multi-step para domínios específicos
2. **Tool integrations** – Instruções para trabalhar com formatos de arquivo ou APIs específicas
3. **Domain expertise** – Conhecimento específico da empresa, schemas, lógica de negócio
4. **Bundled resources** – Scripts, referências e assets para tarefas complexas e repetitivas

## Princípios Fundamentais

### Concisão é essencial

A context window é um bem público. Skills compartilham contexto com tudo o mais que o agent precisa.

**Premissa padrão: o agent já é muito inteligente.** Adicione apenas o contexto que ele ainda não possui. Questione cada informação:  
“Esse agent realmente precisa disso?” e “Esse parágrafo justifica o custo em tokens?”

Prefira exemplos concisos em vez de explicações verbosas.

### Anatomia de uma Skill

Toda skill consiste em um arquivo SKILL.md obrigatório e recursos opcionais:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter metadata (required)
│   │   ├── name: (required)
│   │   └── description: (required)
│   └── Markdown instructions (required)
└── Bundled Resources (optional)
    ├── scripts/          - Executable code (Python/Bash/etc.)
    ├── references/       - Documentation loaded into context as needed
    └── assets/           - Files used in output (templates, icons, fonts, etc.)
```

#### SKILL.md (required)

Todo SKILL.md consiste em:

- **Frontmatter** (YAML): Contém os campos `name` e `description`. Esses são os únicos campos lidos para determinar quando a skill será utilizada — seja claro e abrangente.
- **Body** (Markdown): Instruções e orientações para uso da skill. Só é carregado **APÓS** a skill ser acionada.

#### Bundled Resources (optional)

##### Scripts (`scripts/`)

Código executável para tarefas que exigem confiabilidade determinística ou que são reescritas repetidamente.

- **Quando incluir**: Quando o mesmo código está sendo reescrito várias vezes
- **Exemplo**: `scripts/rotate_pdf.py` para tarefas de rotação de PDF
- **Benefícios**: Eficiência de tokens, comportamento determinístico

##### References (`references/`)

Documentação e material de referência carregados no contexto conforme necessário.

- **Quando incluir**: Para documentação que o agent deve consultar durante o trabalho
- **Exemplos**: `references/schema.md` para schemas de banco de dados, `references/api_docs.md` para especificações de API
- **Benefícios**: Mantém o SKILL.md enxuto, carregado apenas quando necessário

##### Assets (`assets/`)

Arquivos que não devem ser carregados no contexto, mas utilizados no output final.

- **Quando incluir**: Quando a skill precisa de arquivos para o resultado final
- **Exemplos**: `assets/logo.png` para brand assets, `assets/template.html` para boilerplate HTML

### Progressive Disclosure

Skills utilizam um sistema de carregamento em três níveis:

1. **Metadata (name + description)** – Sempre no contexto (~100 palavras)
2. **SKILL.md body** – Quando a skill é acionada (< 5k palavras)
3. **Bundled resources** – Conforme necessário (ilimitado)

Mantenha o body do SKILL.md abaixo de 500 linhas. Divida o conteúdo em arquivos separados ao se aproximar desse limite.

## Processo de Criação de Skill

### Passo 1: Entender a Skill

Esclareça com exemplos concretos:

- “Que funcionalidade esta skill deve suportar?”
- “Você pode dar exemplos de como essa skill seria usada?”
- “O que deve disparar essa skill?”

### Passo 2: Planejar Conteúdos Reutilizáveis

Analise cada exemplo:

1. Considere como executar do zero
2. Identifique scripts, references e assets úteis

### Passo 3: Criar a Skill

Crie o diretório da skill:

```
skill-name/
├── SKILL.md
├── scripts/     (se necessário)
├── references/  (se necessário)
└── assets/      (se necessário)
```

### Passo 4: Escrever o SKILL.md

#### Frontmatter

```yaml
---
name: skill-name
description: O que a skill faz e quando utilizá-la. Inclua triggers e contextos específicos. Máx. 1024 caracteres.
---
```

**Diretrizes para description:**

- Inclua tanto o que a skill faz quanto quando usá-la
- Inclua trigger phrases
- Máx. 1024 caracteres, sem XML tags
- Escreva em terceira pessoa

#### Body

Escreva instruções para uso da skill. Inclua:

- Quick start guide
- Workflow passo a passo
- Links para arquivos de reference quando necessário

### Passo 5: Testar e Iterar

1. Use a skill em tarefas reais
2. Observe dificuldades ou ineficiências
3. Atualize o SKILL.md ou os recursos
4. Teste novamente

## Checklist de Qualidade

Antes de finalizar:

- [ ] Description é específica sobre quando usar (máx. 1024 chars)
- [ ] Nome da pasta em kebab-case
- [ ] Instruções acionáveis e não ambíguas
- [ ] Escopo focado (uma responsabilidade)
- [ ] Body do SKILL.md < 500 linhas
- [ ] References estão a um nível do SKILL.md

## Output Messages

Ao criar uma skill, informe o usuário:

```
✅ Skill criada com sucesso!

📁 Location: .github/skills/[name]/SKILL.md
🎯 Purpose: [descrição breve]
🔧 How to test: [prompt de exemplo que deve disparar a skill]

💡 Tip: O agent usará esta skill automaticamente quando detectar [context].
```