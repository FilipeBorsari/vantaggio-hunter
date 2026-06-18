---
name: semantic-commit
description: Cria commits semânticos seguindo o padrão deste projeto. Use para "faça o commit", "commite as mudanças", "cria um commit", "/semantic-commit". O prefixo deve ser feat: para novas funcionalidades ou melhorias, fix: para correções de bug. Nunca use outros prefixos como refactor:, chore:, docs: etc.
---

# semantic-commit

Cria um commit semântico com as mudanças staged ou modificadas, seguindo o padrão do projeto.

## Regras de prefixo

| Prefixo | Quando usar |
|---|---|
| `feat:` | Nova funcionalidade ou melhoria visível ao usuário |
| `fix:` | Correção de bug ou comportamento incorreto |
| `refactor:` | Reestruturação de código sem mudança de comportamento |
| `chore:` | Tarefas de manutenção, dependências, configuração de tooling |
| `docs:` | Alterações apenas em documentação |
| `test:` | Adição ou correção de testes |
| `style:` | Formatação, lint, sem alteração de lógica |
| `perf:` | Melhoria de performance |

Escolha o prefixo que melhor descreve **a intenção principal** da mudança.

## Workflow

1. **Verifique o status:**
   ```
   git status --short
   git diff --stat HEAD
   ```

2. **Identifique o que será commitado:**
   - Se o usuário especificou arquivos, use apenas eles
   - Caso contrário, use os arquivos modificados relevantes à tarefa (nunca `git add -A` sem verificar)
   - Nunca inclua: `.env`, arquivos de log (`*.log`), `IMPROVEMENTS.md` salvo pedido explícito

3. **Escolha o prefixo** com base na tabela acima.

4. **Escreva a mensagem:**
   - Formato: `<prefixo>: <o que foi feito em português>`
   - Uma linha clara descrevendo a mudança, sem detalhes de implementação
   - Se necessário, adicione corpo com bullets explicando o que mudou

5. **Crie o commit (sem Co-Authored-By):**
   ```
   git commit -m "$(cat <<'EOF'
   feat: descrição do que foi feito

   - detalhe 1
   - detalhe 2
   EOF
   )"
   ```

6. **Confirme ao usuário** com o hash e o título do commit.

## Exemplos

```
feat: adiciona endpoint de busca por CNPJ com filtros avançados
feat: extrai repository pattern e cria camada de domínio
feat: configura slog estruturado e middleware de CORS via env
fix: corrige condição ErrNoRows que engolia erros de banco
fix: valida campos obrigatórios no handler de login
```
