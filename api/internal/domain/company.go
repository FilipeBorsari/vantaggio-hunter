package domain

type CNAE struct {
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
}

type Company struct {
	CNPJ              string   `json:"cnpj"`
	RazaoSocial       string   `json:"razao_social"`
	NomeFantasia      *string  `json:"nome_fantasia,omitempty"`
	Municipio         *string  `json:"municipio,omitempty"`
	UF                string   `json:"uf"`
	CapitalSocial     *float64 `json:"capital_social,omitempty"`
	SituacaoCadastral int      `json:"situacao_cadastral"`
	CNAEs             []CNAE   `json:"cnaes"`
}

type CompanyDetail struct {
	CNPJ              string    `json:"cnpj"`
	RazaoSocial       string    `json:"razao_social"`
	NomeFantasia      *string   `json:"nome_fantasia,omitempty"`
	Logradouro        *string   `json:"logradouro,omitempty"`
	Numero            *string   `json:"numero,omitempty"`
	Complemento       *string   `json:"complemento,omitempty"`
	Bairro            *string   `json:"bairro,omitempty"`
	CEP               *string   `json:"cep,omitempty"`
	Municipio         *string   `json:"municipio,omitempty"`
	UF                string    `json:"uf"`
	CapitalSocial     *float64  `json:"capital_social,omitempty"`
	SituacaoCadastral int       `json:"situacao_cadastral"`
	Porte             *int      `json:"porte,omitempty"`
	OpcaoSimples      *bool     `json:"opcao_simples,omitempty"`
	DataInicio        *string   `json:"data_inicio,omitempty"`
	DDDTelefone1      *string   `json:"ddd_telefone1,omitempty"`
	Telefone1         *string   `json:"telefone1,omitempty"`
	Email             *string   `json:"email,omitempty"`
	CNAEs             []CNAE    `json:"cnaes"`
	Partners          []Partner `json:"socios"`
}

type Partner struct {
	Nome         string  `json:"nome"`
	CPFCNPJSocio *string `json:"cpf_cnpj_socio,omitempty"`
	Qualificacao *int16  `json:"qualificacao,omitempty"`
	DataEntrada  *string `json:"data_entrada,omitempty"`
}

type CompanyListResponse struct {
	Data  []Company `json:"data"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

// CompanyEmbedInput holds the data needed to generate a company's embedding.
type CompanyEmbedInput struct {
	CNPJ string
	UF   string
	Text string // pre-built embedding text
}

// CompanyEmbedding pairs a company identifier with its generated vector.
type CompanyEmbedding struct {
	CNPJ   string
	UF     string
	Vector []float32
}
