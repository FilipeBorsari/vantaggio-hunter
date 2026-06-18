package main

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ---- domain types ----

type CNAERow struct {
	Code        string
	Description string
}

type MunicipioRow struct {
	Code int
	Name string
}

type EmpresaRow struct {
	CNPJBasico       string
	RazaoSocial      string
	NaturezaJuridica string
	CapitalSocial    float64
	Porte            int16
}

type SimplesRow struct {
	CNPJBasico   string
	OpcaoSimples bool
}

type EstabelecimentoRow struct {
	CNPJ              string // 14-digit full CNPJ
	CNPJBasico        string // 8-digit, for staging lookup
	NomeFantasia      string
	SituacaoCadastral int16
	DataSituacao      *time.Time
	DataInicio        *time.Time
	CNAEPrincipal     string
	CNAEsSecundarios  []string
	Logradouro        string // "TIPO NOME" joined
	Numero            string
	Complemento       string
	Bairro            string
	CEP               string
	UF                string
	MunicipioCode     int
	DDD1              string
	Telefone1         string
	Email             string
}

type SocioRow struct {
	CNPJBasico    string
	TipoSocio     int16
	NomeSocio     string
	CPFCNPJSocio  string
	Qualificacao  int16
	DataEntrada   *time.Time
	Pais          string
	FaixaEtaria   int16
}

// ---- helpers ----

func openISO(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// Wrap with ISO-8859-1 → UTF-8 decoder
	decoded := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	return &isoReadCloser{Reader: decoded, closer: f}, nil
}

type isoReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *isoReadCloser) Close() error { return r.closer.Close() }

func newCSVReader(r io.Reader) *csv.Reader {
	cr := csv.NewReader(r)
	cr.Comma = ';'
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // allow variable number of fields
	return cr
}

func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "00000000" {
		return nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return nil
	}
	return &t
}

// parseCapital converts "1.234.567,89" or "1234567,89" → float64
func parseCapital(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")  // remove thousand separators
	s = strings.ReplaceAll(s, ",", ".") // decimal separator
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt16(s string) int16 {
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseInt(s, 10, 16)
	return int16(v)
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	v, _ := strconv.Atoi(s)
	return v
}

func field(rec []string, i int) string {
	if i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(rec[i], "\x00", ""))
}

// ---- parsers ----

// ParseCNAEs reads the entire CNAE CSV into memory (small table, ~1300 rows).
func ParseCNAEs(path string) ([]CNAERow, error) {
	rc, err := openISO(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var rows []CNAERow
	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed lines
		}
		if len(rec) < 2 {
			continue
		}
		rows = append(rows, CNAERow{
			Code:        field(rec, 0),
			Description: field(rec, 1),
		})
	}
	return rows, nil
}

// ParseMunicipios reads the entire municipality CSV into memory (~5600 rows).
func ParseMunicipios(path string) ([]MunicipioRow, error) {
	rc, err := openISO(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var rows []MunicipioRow
	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 2 {
			continue
		}
		rows = append(rows, MunicipioRow{
			Code: parseInt(field(rec, 0)),
			Name: field(rec, 1),
		})
	}
	return rows, nil
}

// StreamEmpresas calls fn for each parsed row — Empresas CSV columns:
// 0:cnpj_basico 1:razao_social 2:natureza_juridica 3:qualif_responsavel
// 4:capital_social 5:porte 6:ente_federativo
func StreamEmpresas(path string, fn func(EmpresaRow) error) error {
	rc, err := openISO(path)
	if err != nil {
		return err
	}
	defer rc.Close()

	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 5 {
			continue
		}
		row := EmpresaRow{
			CNPJBasico:       field(rec, 0),
			RazaoSocial:      field(rec, 1),
			NaturezaJuridica: field(rec, 2),
			CapitalSocial:    parseCapital(field(rec, 4)),
			Porte:            parseInt16(field(rec, 5)),
		}
		if row.CNPJBasico == "" {
			continue
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

// StreamSimples calls fn for each parsed row — Simples CSV columns:
// 0:cnpj_basico 1:opcao_simples(S/N) 2:dt_opcao 3:dt_exclusao
// 4:opcao_mei(S/N) 5:dt_opcao_mei 6:dt_exclusao_mei
func StreamSimples(path string, fn func(SimplesRow) error) error {
	rc, err := openISO(path)
	if err != nil {
		return err
	}
	defer rc.Close()

	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 2 {
			continue
		}
		row := SimplesRow{
			CNPJBasico:   field(rec, 0),
			OpcaoSimples: strings.ToUpper(field(rec, 1)) == "S",
		}
		if row.CNPJBasico == "" {
			continue
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

// StreamEstabelecimentos calls fn for each parsed row — Estabelecimentos columns:
// 0:cnpj_basico 1:cnpj_ordem 2:cnpj_dv 3:matriz_filial 4:nome_fantasia
// 5:situacao_cadastral 6:dt_situacao 7:motivo 8:nm_cidade_ext 9:pais
// 10:dt_inicio 11:cnae_principal 12:cnaes_secundarios
// 13:tipo_logradouro 14:logradouro 15:numero 16:complemento 17:bairro
// 18:cep 19:uf 20:municipio 21:ddd1 22:tel1 23:ddd2 24:tel2
// 25:ddd_fax 26:fax 27:email 28:sit_especial 29:dt_sit_especial
func StreamEstabelecimentos(path string, fn func(EstabelecimentoRow) error) error {
	rc, err := openISO(path)
	if err != nil {
		return err
	}
	defer rc.Close()

	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 20 {
			continue
		}

		basico := field(rec, 0)
		ordem := field(rec, 1)
		dv := field(rec, 2)
		cnpj := basico + ordem + dv

		tipoLog := field(rec, 13)
		logr := field(rec, 14)
		if tipoLog != "" && logr != "" {
			logr = tipoLog + " " + logr
		} else if tipoLog != "" {
			logr = tipoLog
		}

		var secundarios []string
		if s := field(rec, 12); s != "" {
			for _, code := range strings.Split(s, ",") {
				code = strings.TrimSpace(code)
				if code != "" {
					secundarios = append(secundarios, code)
				}
			}
		}

		row := EstabelecimentoRow{
			CNPJ:              cnpj,
			CNPJBasico:        basico,
			NomeFantasia:      field(rec, 4),
			SituacaoCadastral: parseInt16(field(rec, 5)),
			DataSituacao:      parseDate(field(rec, 6)),
			DataInicio:        parseDate(field(rec, 10)),
			CNAEPrincipal:     field(rec, 11),
			CNAEsSecundarios:  secundarios,
			Logradouro:        logr,
			Numero:            field(rec, 15),
			Complemento:       field(rec, 16),
			Bairro:            field(rec, 17),
			CEP:               field(rec, 18),
			UF:                strings.ToUpper(field(rec, 19)),
			MunicipioCode:     parseInt(field(rec, 20)),
			DDD1:              field(rec, 21),
			Telefone1:         field(rec, 22),
			Email:             strings.ToLower(field(rec, 27)),
		}
		if cnpj == "" || row.UF == "" {
			continue
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

// StreamSocios calls fn for each parsed row — Socios columns:
// 0:cnpj_basico 1:tipo_socio 2:nome 3:cpf_cnpj 4:qualificacao
// 5:dt_entrada 6:pais 7:repr_legal_cpf 8:repr_legal_nome
// 9:repr_legal_qualif 10:faixa_etaria
func StreamSocios(path string, fn func(SocioRow) error) error {
	rc, err := openISO(path)
	if err != nil {
		return err
	}
	defer rc.Close()

	r := newCSVReader(rc)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 5 {
			continue
		}
		row := SocioRow{
			CNPJBasico:   field(rec, 0),
			TipoSocio:    parseInt16(field(rec, 1)),
			NomeSocio:    field(rec, 2),
			CPFCNPJSocio: field(rec, 3),
			Qualificacao:  parseInt16(field(rec, 4)),
			DataEntrada:   parseDate(field(rec, 5)),
			Pais:          field(rec, 6),
			FaixaEtaria:   parseInt16(field(rec, 10)),
		}
		if row.CNPJBasico == "" {
			continue
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}
