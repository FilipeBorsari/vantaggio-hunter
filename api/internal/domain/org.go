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

// OrgDetail is the detailed view of an org for super admin.
type OrgDetail struct {
	Org   Org         `json:"org"`
	Stats OrgStats    `json:"stats"`
	Users []OrgUser   `json:"users"`
}

type OrgStats struct {
	Balance       int `json:"balance"`
	TotalSearches int `json:"total_searches"`
	Exports       int `json:"exports"`
}

// OrgUser is a user as seen by super admin or org admin.
type OrgUser struct {
	UserID             string  `json:"user_id"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	Role               string  `json:"role"`
	IsActive           bool    `json:"is_active"`
	CreditLimit        *int    `json:"credit_limit"`
	SearchesThisMonth  int     `json:"searches_this_month"`
	ExportsThisMonth   int     `json:"exports_this_month"`
	CreditsConsumed    int     `json:"credits_consumed"`
	LastActiveAt       *string `json:"last_active_at"`
}

// AdminDashboard is the global dashboard for super admin.
type AdminDashboard struct {
	TotalOrgs             int          `json:"total_orgs"`
	ActiveOrgs            int          `json:"active_orgs"`
	TotalSearches         int          `json:"total_searches"`
	TotalExports          int          `json:"total_exports"`
	TotalCreditsConsumed  int          `json:"total_credits_consumed"`
	Orgs                  []OrgSummary `json:"orgs"`
}

type OrgSummary struct {
	OrgID    string `json:"org_id"`
	Name     string `json:"name"`
	Searches int    `json:"searches"`
	Exports  int    `json:"exports"`
	Balance  int    `json:"balance"`
	IsActive bool   `json:"is_active"`
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

// Invitation represents a pending user invite.
type Invitation struct {
	ID         string  `json:"invitation_id"`
	OrgID      string  `json:"org_id"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	Token      string  `json:"-"`
	InvitedBy  string  `json:"invited_by"`
	AcceptedAt *string `json:"accepted_at"`
	ExpiresAt  string  `json:"expires_at"`
	CreatedAt  string  `json:"created_at"`
}

// AuditLog is one entry in tb_audit_logs.
type AuditLog struct {
	ID        int64   `json:"id"`
	OrgID     *string `json:"org_id"`
	ActorID   string  `json:"actor_id"`
	Action    string  `json:"action"`
	TargetID  *string `json:"target_id"`
	Metadata  any     `json:"metadata"`
	CreatedAt string  `json:"created_at"`
}

// SellerProfile is the profile response for a seller (/me/profile).
type SellerProfile struct {
	UserID                string  `json:"user_id"`
	Name                  string  `json:"name"`
	Email                 string  `json:"email"`
	OrgName               string  `json:"org_name"`
	CreditsConsumedMonth  int     `json:"credits_consumed_this_month"`
	CreditLimit           *int    `json:"credit_limit"`
}

// OrgCosts holds cost breakdown by seller.
type OrgCosts struct {
	Period               string         `json:"period"`
	TotalCreditsConsumed int            `json:"total_credits_consumed"`
	BySeller             []SellerCost   `json:"by_seller"`
}

type SellerCost struct {
	UserID          string `json:"user_id"`
	Name            string `json:"name"`
	Searches        int    `json:"searches"`
	Exports         int    `json:"exports"`
	CreditsConsumed int    `json:"credits_consumed"`
}
