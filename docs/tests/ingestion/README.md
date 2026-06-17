# Testes — Ingestion (CSV Parser)

Cobertura do pipeline de ingestão de dados da Receita Federal: `cmd/ingestion/csv_parser.go`.

---

## csv_parser_test.go

Os testes cobrem duas categorias: **funções puras** (sem I/O) e **funções de streaming** (leem arquivos temporários criados via `os.CreateTemp`).

### Funções puras

| Teste | Função | O que verifica |
|---|---|---|
| `TestParseDate` | `parseDate` | Formato `20060102` válido, string vazia, `"00000000"`, espaços, formato errado → `nil` |
| `TestParseCapital` | `parseCapital` | Formato BR `"1.234,56"`, sem separador de milhar, zero, string vazia, espaços extras |
| `TestParseInt16` | `parseInt16` | Positivo, negativo, zero, vazio, string inválida, espaços |
| `TestParseInt` | `parseInt` | Positivo, negativo, zero, vazio, string inválida, espaços |
| `TestField` | `field` | Trim de espaços, remoção de null bytes (`\x00`), índice fora do range retorna `""` |
| `TestNewCSVReader_UseSemicolonDelimiter` | `newCSVReader` | Delimitador `;` configurado corretamente |

### Parsers com arquivo temporário

Cada teste cria um `.csv` temporário em `t.TempDir()` (limpeza automática) com conteúdo ASCII puro (subconjunto de ISO-8859-1).

| Teste | Função | O que verifica |
|---|---|---|
| `TestParseCNAEs` | `ParseCNAEs` | 2 linhas válidas parseadas; linha com 1 campo ignorada |
| `TestParseCNAEs_FileNotFound` | `ParseCNAEs` | Erro para arquivo inexistente |
| `TestParseMunicipios` | `ParseMunicipios` | Code (int) e Name parseados; linha incompleta ignorada |
| `TestStreamEmpresas` | `StreamEmpresas` | `CNPJBasico`, `RazaoSocial`, `CapitalSocial` (formato BR), `Porte`; linha sem CNPJ ignorada |
| `TestStreamEmpresas_FnError` | `StreamEmpresas` | Erro retornado pelo callback `fn` interrompe o streaming |
| `TestStreamSimples` | `StreamSimples` | `OpcaoSimples=true` para `"S"`, `false` para `"N"`; linha sem CNPJ ignorada |
| `TestStreamEstabelecimentos` | `StreamEstabelecimentos` | CNPJ montado de basico+ordem+dv, `Logradouro` = "TIPO NOME" concatenado, CNAEs secundários parseados de CSV, email em lowercase, `SituacaoCadastral` |
| `TestStreamEstabelecimentos_SkipsRowWithoutUF` | `StreamEstabelecimentos` | Linha com UF vazia é ignorada |
| `TestStreamSocios` | `StreamSocios` | `NomeSocio`, `Qualificacao`, `FaixaEtaria` (campo 10); linha sem CNPJ ignorada |

---

## Convenções do CSV da Receita Federal

| Campo | Formato | Tratamento |
|---|---|---|
| Datas | `YYYYMMDD` ou `00000000` | `parseDate` → `*time.Time` ou `nil` |
| Capital social | `1.234.567,89` (BR) | `parseCapital` → `float64` |
| Encoding | ISO-8859-1 | `openISO` aplica decoder antes de ler |
| Delimitador | `;` | `newCSVReader` configura `Comma = ';'` |
| Campos ausentes | Menos campos que o esperado | `field(rec, i)` retorna `""` sem panic |

---

## Como rodar

```bash
make test
# ou apenas o pacote ingestion:
cd api && /home/filipeborsari/go/bin/gotestsum --format testdox -- ./cmd/ingestion/...
```
