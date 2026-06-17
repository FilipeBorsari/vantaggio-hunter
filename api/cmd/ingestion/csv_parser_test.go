package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Pure helper functions
// ---------------------------------------------------------------------------

func TestParseDate(t *testing.T) {
	cases := []struct {
		input string
		want  string // empty string means nil expected
	}{
		{"20230115", "2023-01-15"},
		{"19991231", "1999-12-31"},
		{"", ""},
		{"00000000", ""},
		{"   ", ""},
		{"invalid", ""},
		{"2023-01-15", ""}, // wrong format
	}
	for _, tc := range cases {
		got := parseDate(tc.input)
		if tc.want == "" {
			if got != nil {
				t.Errorf("parseDate(%q) = %v, want nil", tc.input, got)
			}
		} else {
			if got == nil {
				t.Errorf("parseDate(%q) = nil, want %q", tc.input, tc.want)
				continue
			}
			if got.Format("2006-01-02") != tc.want {
				t.Errorf("parseDate(%q) = %q, want %q", tc.input, got.Format("2006-01-02"), tc.want)
			}
		}
	}
}

func TestParseCapital(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"1.234,56", 1234.56},
		{"1234,56", 1234.56},
		{"1.000.000,00", 1000000.00},
		{"0,00", 0.00},
		{"", 0.00},
		{"  500,00  ", 500.00},
	}
	for _, tc := range cases {
		got := parseCapital(tc.input)
		if got != tc.want {
			t.Errorf("parseCapital(%q) = %f, want %f", tc.input, got, tc.want)
		}
	}
}

func TestParseInt16(t *testing.T) {
	cases := []struct {
		input string
		want  int16
	}{
		{"0", 0},
		{"1", 1},
		{"32767", 32767},
		{"-1", -1},
		{"", 0},
		{"abc", 0},
		{"  42  ", 42},
	}
	for _, tc := range cases {
		got := parseInt16(tc.input)
		if got != tc.want {
			t.Errorf("parseInt16(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"42", 42},
		{"-10", -10},
		{"", 0},
		{"abc", 0},
		{"  99  ", 99},
	}
	for _, tc := range cases {
		got := parseInt(tc.input)
		if got != tc.want {
			t.Errorf("parseInt(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestField(t *testing.T) {
	rec := []string{"  alpha  ", "beta", "\x00null\x00byte"}

	if got := field(rec, 0); got != "alpha" {
		t.Errorf("field(rec, 0) = %q, want alpha", got)
	}
	if got := field(rec, 1); got != "beta" {
		t.Errorf("field(rec, 1) = %q, want beta", got)
	}
	if got := field(rec, 2); got != "nullbyte" {
		t.Errorf("field(rec, 2) = %q, want nullbyte (null bytes stripped)", got)
	}
	// Out-of-bounds index returns empty string.
	if got := field(rec, 99); got != "" {
		t.Errorf("field(rec, 99) = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// File-based parsers (use temp files with ASCII/UTF-8 content)
// ---------------------------------------------------------------------------

// writeTempCSV writes content to a temp file and returns its path.
func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.csv")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := io.Copy(f, strings.NewReader(content)); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	return f.Name()
}

func TestParseCNAEs(t *testing.T) {
	content := "6201500;Desenvolvimento de programas de computador sob encomenda\n" +
		"6202300;Desenvolvimento e licenciamento de programas de computador customizaveis\n" +
		"invalid-only-one-field\n" // skipped: < 2 fields
	path := writeTempCSV(t, content)

	rows, err := ParseCNAEs(path)
	if err != nil {
		t.Fatalf("ParseCNAEs: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Code != "6201500" {
		t.Errorf("rows[0].Code = %q, want 6201500", rows[0].Code)
	}
	if !strings.Contains(rows[0].Description, "Desenvolvimento") {
		t.Errorf("rows[0].Description = %q", rows[0].Description)
	}
}

func TestParseCNAEs_FileNotFound(t *testing.T) {
	_, err := ParseCNAEs("/nonexistent/path/cnaes.csv")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseMunicipios(t *testing.T) {
	content := "1234;SAO PAULO\n" +
		"5678;CAMPINAS\n" +
		"0\n" // skipped: < 2 fields
	path := writeTempCSV(t, content)

	rows, err := ParseMunicipios(path)
	if err != nil {
		t.Fatalf("ParseMunicipios: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Code != 1234 {
		t.Errorf("rows[0].Code = %d, want 1234", rows[0].Code)
	}
	if rows[0].Name != "SAO PAULO" {
		t.Errorf("rows[0].Name = %q, want SAO PAULO", rows[0].Name)
	}
}

func TestStreamEmpresas(t *testing.T) {
	// columns: cnpj_basico;razao_social;natureza_juridica;qualif_responsavel;capital_social;porte;ente_federativo
	content := "12345678;ACME LTDA;2062;;1.000.000,00;3;\n" +
		";MISSING CNPJ;2062;;0,00;1;\n" + // skipped: empty cnpj_basico
		"short\n" // skipped: < 5 fields
	path := writeTempCSV(t, content)

	var collected []EmpresaRow
	err := StreamEmpresas(path, func(r EmpresaRow) error {
		collected = append(collected, r)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamEmpresas: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("got %d rows, want 1", len(collected))
	}
	if collected[0].CNPJBasico != "12345678" {
		t.Errorf("CNPJBasico = %q, want 12345678", collected[0].CNPJBasico)
	}
	if collected[0].RazaoSocial != "ACME LTDA" {
		t.Errorf("RazaoSocial = %q, want ACME LTDA", collected[0].RazaoSocial)
	}
	if collected[0].CapitalSocial != 1_000_000.00 {
		t.Errorf("CapitalSocial = %f, want 1000000.00", collected[0].CapitalSocial)
	}
	if collected[0].Porte != 3 {
		t.Errorf("Porte = %d, want 3", collected[0].Porte)
	}
}

func TestStreamEmpresas_FnError(t *testing.T) {
	content := "12345678;ACME;2062;;0,00;1;\n"
	path := writeTempCSV(t, content)

	boom := fmt.Errorf("processing error")
	err := StreamEmpresas(path, func(_ EmpresaRow) error { return boom })
	if err == nil {
		t.Fatal("expected error from fn, got nil")
	}
}

func TestStreamSimples(t *testing.T) {
	// columns: cnpj_basico;opcao_simples;dt_opcao;dt_exclusao;opcao_mei;dt_opcao_mei;dt_exclusao_mei
	content := "12345678;S;20200101;;\n" +
		"87654321;N;;\n" +
		";S;\n" // skipped: empty cnpj_basico
	path := writeTempCSV(t, content)

	var collected []SimplesRow
	err := StreamSimples(path, func(r SimplesRow) error {
		collected = append(collected, r)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSimples: %v", err)
	}
	if len(collected) != 2 {
		t.Fatalf("got %d rows, want 2", len(collected))
	}
	if !collected[0].OpcaoSimples {
		t.Errorf("row 0 OpcaoSimples = false, want true")
	}
	if collected[1].OpcaoSimples {
		t.Errorf("row 1 OpcaoSimples = true, want false")
	}
}

func TestStreamEstabelecimentos(t *testing.T) {
	// Build a full 30-field record:
	// 0:basico 1:ordem 2:dv 3:matriz 4:fantasia 5:situacao 6:dt_sit 7:motivo 8:nm_cidade 9:pais
	// 10:dt_inicio 11:cnae_principal 12:cnaes_sec 13:tipo_logr 14:logr 15:num 16:compl 17:bairro
	// 18:cep 19:uf 20:municipio 21:ddd1 22:tel1 23:ddd2 24:tel2 25:ddd_fax 26:fax 27:email 28:sit_esp 29:dt_sit_esp
	record := strings.Join([]string{
		"12345678", "0001", "00", "1", "EMPRESA TESTE",
		"2", "20230615", "", "SAO PAULO", "BR",
		"20100101", "6201500", "6209100,6202300", "RUA", "PRINCIPAL",
		"100", "APTO 1", "CENTRO", "01310100", "SP",
		"3550308", "11", "999999999", "", "",
		"", "", "empresa@test.com", "", "",
	}, ";")
	path := writeTempCSV(t, record+"\n")

	var collected []EstabelecimentoRow
	err := StreamEstabelecimentos(path, func(r EstabelecimentoRow) error {
		collected = append(collected, r)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamEstabelecimentos: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("got %d rows, want 1", len(collected))
	}

	row := collected[0]
	if row.CNPJ != "12345678000100" {
		t.Errorf("CNPJ = %q, want 12345678000100", row.CNPJ)
	}
	if row.CNPJBasico != "12345678" {
		t.Errorf("CNPJBasico = %q, want 12345678", row.CNPJBasico)
	}
	if row.UF != "SP" {
		t.Errorf("UF = %q, want SP", row.UF)
	}
	if row.NomeFantasia != "EMPRESA TESTE" {
		t.Errorf("NomeFantasia = %q, want EMPRESA TESTE", row.NomeFantasia)
	}
	if row.CNAEPrincipal != "6201500" {
		t.Errorf("CNAEPrincipal = %q, want 6201500", row.CNAEPrincipal)
	}
	if len(row.CNAEsSecundarios) != 2 {
		t.Errorf("CNAEsSecundarios len = %d, want 2", len(row.CNAEsSecundarios))
	}
	// Logradouro should be "TIPO NOME" joined
	if row.Logradouro != "RUA PRINCIPAL" {
		t.Errorf("Logradouro = %q, want RUA PRINCIPAL", row.Logradouro)
	}
	if row.Email != "empresa@test.com" {
		t.Errorf("Email = %q, want empresa@test.com", row.Email)
	}
	if row.SituacaoCadastral != 2 {
		t.Errorf("SituacaoCadastral = %d, want 2", row.SituacaoCadastral)
	}
}

func TestStreamEstabelecimentos_SkipsRowWithoutUF(t *testing.T) {
	// Row with empty UF must be skipped.
	record := strings.Join([]string{
		"12345678", "0001", "00", "1", "EMPRESA",
		"2", "20230615", "", "", "",
		"20100101", "6201500", "", "RUA", "PRINCIPAL",
		"100", "", "CENTRO", "01310100", "", // UF is empty
		"3550308", "11", "999999999", "", "",
		"", "", "", "", "",
	}, ";")
	path := writeTempCSV(t, record+"\n")

	var collected []EstabelecimentoRow
	_ = StreamEstabelecimentos(path, func(r EstabelecimentoRow) error {
		collected = append(collected, r)
		return nil
	})
	if len(collected) != 0 {
		t.Errorf("expected row with empty UF to be skipped, got %d rows", len(collected))
	}
}

func TestStreamSocios(t *testing.T) {
	// columns: cnpj_basico;tipo_socio;nome;cpf_cnpj;qualificacao;dt_entrada;pais;repr_cpf;repr_nome;repr_qual;faixa_etaria
	// fields 7,8,9 are repr_cpf, repr_nome, repr_qual (all empty); field 10 is faixa_etaria
	content := "12345678;2;JOAO DA SILVA;***111222**;49;20150301;BR;;;;5\n" +
		";1;SKIP ME;;49;;;\n" // skipped: empty cnpj_basico
	path := writeTempCSV(t, content)

	var collected []SocioRow
	err := StreamSocios(path, func(r SocioRow) error {
		collected = append(collected, r)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSocios: %v", err)
	}
	if len(collected) != 1 {
		t.Fatalf("got %d rows, want 1", len(collected))
	}
	if collected[0].CNPJBasico != "12345678" {
		t.Errorf("CNPJBasico = %q, want 12345678", collected[0].CNPJBasico)
	}
	if collected[0].NomeSocio != "JOAO DA SILVA" {
		t.Errorf("NomeSocio = %q, want JOAO DA SILVA", collected[0].NomeSocio)
	}
	if collected[0].Qualificacao != 49 {
		t.Errorf("Qualificacao = %d, want 49", collected[0].Qualificacao)
	}
	if collected[0].FaixaEtaria != 5 {
		t.Errorf("FaixaEtaria = %d, want 5", collected[0].FaixaEtaria)
	}
}

func TestNewCSVReader_UseSemicolonDelimiter(t *testing.T) {
	data := "a;b;c\nd;e;f\n"
	r := newCSVReader(bytes.NewBufferString(data))
	rec, err := r.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec) != 3 || rec[0] != "a" || rec[1] != "b" || rec[2] != "c" {
		t.Errorf("record = %v, want [a b c]", rec)
	}
}
