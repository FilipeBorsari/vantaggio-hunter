package domain

type Plan struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Credits    int    `json:"credits"`
	PriceCents int    `json:"price_cents"`
}

type Org struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	PlanID    *string `json:"plan_id"`
	PlanName  *string `json:"plan_name,omitempty"`
	IsActive  bool    `json:"is_active"`
	UserCount int     `json:"user_count"`
	CreatedAt string  `json:"created_at"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type OrgListResponse struct {
	Data  []Org `json:"data"`
	Total int   `json:"total"`
}
