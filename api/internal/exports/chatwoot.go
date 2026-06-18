package exports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/vantaggio/prospect-api/internal/domain"
)

// ChatwootClient sends leads to a Chatwoot instance.
type ChatwootClient struct {
	baseURL   string
	apiKey    string
	accountID int
	inboxID   *int
	http      *http.Client
}

func NewChatwootClient(baseURL, apiKey string, accountID int, inboxID *int, httpClient *http.Client) *ChatwootClient {
	return &ChatwootClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		accountID: accountID,
		inboxID:   inboxID,
		http:      httpClient,
	}
}

type chatwootContact struct {
	ID int `json:"id"`
}

type chatwootSearchResult struct {
	Payload []chatwootContact `json:"payload"`
}

// ExportCompany creates or finds a contact then opens a conversation in Chatwoot.
// Returns the contact ID and conversation ID on success.
func (c *ChatwootClient) ExportCompany(ctx context.Context, company *domain.CompanyDetail, searchID string) error {
	contactID, err := c.findOrCreateContact(ctx, company)
	if err != nil {
		return fmt.Errorf("find or create contact: %w", err)
	}
	if err := c.createConversation(ctx, contactID, searchID); err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	return nil
}

func (c *ChatwootClient) findOrCreateContact(ctx context.Context, company *domain.CompanyDetail) (int, error) {
	// Try to find by email first, then by phone.
	if company.Email != nil && *company.Email != "" {
		id, found, err := c.searchContact(ctx, *company.Email)
		if err != nil {
			return 0, err
		}
		if found {
			return id, nil
		}
	}

	phone := c.buildPhone(company)
	if phone != "" {
		id, found, err := c.searchContact(ctx, phone)
		if err != nil {
			return 0, err
		}
		if found {
			return id, nil
		}
	}

	return c.createContact(ctx, company, phone)
}

func (c *ChatwootClient) searchContact(ctx context.Context, q string) (int, bool, error) {
	endpoint := fmt.Sprintf("%s/api/v1/accounts/%d/contacts/search?q=%s",
		c.baseURL, c.accountID, url.QueryEscape(q))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, false, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("api_access_token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("search contact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, nil
	}

	var result chatwootSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, false, fmt.Errorf("decode search response: %w", err)
	}
	if len(result.Payload) > 0 {
		return result.Payload[0].ID, true, nil
	}
	return 0, false, nil
}

func (c *ChatwootClient) createContact(ctx context.Context, company *domain.CompanyDetail, phone string) (int, error) {
	primaryCNAE := ""
	for _, cn := range company.CNAEs {
		if cn.IsPrimary {
			primaryCNAE = cn.Description
			break
		}
	}

	body := map[string]any{
		"name":         company.RazaoSocial,
		"phone_number": phone,
		"additional_attributes": map[string]any{
			"cnpj":          company.CNPJ,
			"municipio":     strVal(company.Municipio),
			"uf":            company.UF,
			"cnae":          primaryCNAE,
			"capital_social": company.CapitalSocial,
		},
	}
	if company.Email != nil {
		body["email"] = *company.Email
	}

	endpoint := fmt.Sprintf("%s/api/v1/accounts/%d/contacts", c.baseURL, c.accountID)
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("build create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("create contact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("create contact: status %d: %s", resp.StatusCode, string(b))
	}

	var contact chatwootContact
	if err := json.NewDecoder(resp.Body).Decode(&contact); err != nil {
		return 0, fmt.Errorf("decode create contact response: %w", err)
	}
	return contact.ID, nil
}

func (c *ChatwootClient) createConversation(ctx context.Context, contactID int, searchID string) error {
	if c.inboxID == nil {
		return nil
	}

	body := map[string]any{
		"inbox_id":   *c.inboxID,
		"contact_id": contactID,
		"additional_attributes": map[string]any{
			"origem":    "Vantaggio Hunter",
			"search_id": searchID,
		},
	}

	endpoint := fmt.Sprintf("%s/api/v1/accounts/%d/conversations", c.baseURL, c.accountID)
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build conversation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create conversation: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *ChatwootClient) buildPhone(company *domain.CompanyDetail) string {
	if company.DDDTelefone1 != nil && company.Telefone1 != nil {
		return "+" + *company.DDDTelefone1 + *company.Telefone1
	}
	if company.Telefone1 != nil {
		return *company.Telefone1
	}
	return ""
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
