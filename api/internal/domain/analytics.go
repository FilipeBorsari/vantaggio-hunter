package domain

type AnalyticsKPIs struct {
	Period           string  `json:"period"`
	CreditsConsumed  int     `json:"credits_consumed"`
	CreditsPurchased int     `json:"credits_purchased"`
	LeadsExtracted   int     `json:"leads_extracted"`
	LeadsQualified   int     `json:"leads_qualified"`
	LeadsExported    int     `json:"leads_exported"`
	ConversionRate   float64 `json:"conversion_rate"`
	SearchesCount    int     `json:"searches_count"`
}

type DailyPoint struct {
	Date    string `json:"date"`
	Credits int    `json:"credits"`
	Leads   int    `json:"leads"`
}

type TopCNAE struct {
	CNAECode    string `json:"cnae_code"`
	Description string `json:"description"`
	Leads       int    `json:"leads"`
}

type FunnelStage struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type FunnelResponse struct {
	Stages []FunnelStage `json:"stages"`
}
