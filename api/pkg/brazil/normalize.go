package brazil

import "strings"

// NormalizeCNPJ strips formatting characters (., /, -) from a CNPJ string,
// returning only the 14 digits used for database lookups.
func NormalizeCNPJ(cnpj string) string {
	return onlyDigits(cnpj)
}

// NormalizeCNAE strips formatting characters (., -, /) from a CNAE code string,
// returning only the 7 digits used for database lookups.
func NormalizeCNAE(cnae string) string {
	return onlyDigits(cnae)
}

func onlyDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
