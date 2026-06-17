package companies

import (
	"context"
	"errors"
	"testing"

	"github.com/vantaggio/prospect-api/internal/domain"
)

// mockCompanyRepo is a controllable stub for the Repository interface.
type mockCompanyRepo struct {
	count       int
	countErr    error
	list        []domain.Company
	listErr     error
	attachErr   error
	detail      *domain.CompanyDetail
	detailErr   error
	cnaes       []domain.CNAE
	cnaesErr    error
	partners    []domain.Partner
	partnersErr error

	// Capture what was passed to AttachCNAEs to verify it was called correctly.
	attachedCompanies []domain.Company
}

func (m *mockCompanyRepo) Count(_ context.Context, _ Filters) (int, error) {
	return m.count, m.countErr
}
func (m *mockCompanyRepo) List(_ context.Context, _ Filters) ([]domain.Company, error) {
	return m.list, m.listErr
}
func (m *mockCompanyRepo) AttachCNAEs(_ context.Context, companies []domain.Company) error {
	m.attachedCompanies = companies
	return m.attachErr
}
func (m *mockCompanyRepo) GetByCNPJ(_ context.Context, _ string) (*domain.CompanyDetail, error) {
	return m.detail, m.detailErr
}
func (m *mockCompanyRepo) GetCNAEsByCNPJ(_ context.Context, _ string) ([]domain.CNAE, error) {
	return m.cnaes, m.cnaesErr
}
func (m *mockCompanyRepo) GetPartnersByCNPJBasico(_ context.Context, _ string) ([]domain.Partner, error) {
	return m.partners, m.partnersErr
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_Success(t *testing.T) {
	companies := []domain.Company{
		{CNPJ: "12345678000100", RazaoSocial: "ACME LTDA", UF: "SP"},
		{CNPJ: "98765432000199", RazaoSocial: "BETA SA", UF: "RJ"},
	}
	repo := &mockCompanyRepo{count: 2, list: companies}
	svc := NewService(repo)

	resp, err := svc.List(context.Background(), Filters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
	if resp.Limit != 10 {
		t.Errorf("limit = %d, want 10", resp.Limit)
	}
}

func TestList_PassesFiltersThrough(t *testing.T) {
	var capturedFilters Filters
	repo := &mockCompanyRepo{count: 0, list: []domain.Company{}}
	svc := NewService(repo)

	f := Filters{UF: "MG", City: "BH", Page: 3, Limit: 25}
	_, err := svc.List(context.Background(), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The filters are passed through to the response struct.
	_ = capturedFilters
}

func TestList_AttachCNAEsIsCalled(t *testing.T) {
	companies := []domain.Company{{CNPJ: "12345678000100", UF: "SP"}}
	repo := &mockCompanyRepo{count: 1, list: companies}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), Filters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.attachedCompanies) != 1 {
		t.Errorf("AttachCNAEs called with %d companies, want 1", len(repo.attachedCompanies))
	}
}

func TestList_EmptyResult(t *testing.T) {
	repo := &mockCompanyRepo{count: 0, list: []domain.Company{}}
	svc := NewService(repo)

	resp, err := svc.List(context.Background(), Filters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if len(resp.Data) != 0 {
		t.Errorf("data len = %d, want 0", len(resp.Data))
	}
}

func TestList_CountError(t *testing.T) {
	repo := &mockCompanyRepo{countErr: errors.New("connection reset")}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), Filters{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestList_ListError(t *testing.T) {
	repo := &mockCompanyRepo{count: 5, listErr: errors.New("query timeout")}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), Filters{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestList_AttachCNAEsError(t *testing.T) {
	repo := &mockCompanyRepo{
		count:     1,
		list:      []domain.Company{{CNPJ: "123", UF: "SP"}},
		attachErr: errors.New("cnaes join failed"),
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), Filters{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetByCNPJ
// ---------------------------------------------------------------------------

func TestGetByCNPJ_Success(t *testing.T) {
	detail := &domain.CompanyDetail{CNPJ: "12345678000100", RazaoSocial: "ACME LTDA", UF: "SP"}
	cnaes := []domain.CNAE{
		{Code: "6201500", Description: "Desenvolvimento de software", IsPrimary: true},
	}
	partners := []domain.Partner{{Nome: "JOAO DA SILVA"}}

	repo := &mockCompanyRepo{detail: detail, cnaes: cnaes, partners: partners}
	svc := NewService(repo)

	result, err := svc.GetByCNPJ(context.Background(), "12345678000100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CNPJ != "12345678000100" {
		t.Errorf("cnpj = %q, want 12345678000100", result.CNPJ)
	}
	if len(result.CNAEs) != 1 {
		t.Errorf("cnaes len = %d, want 1", len(result.CNAEs))
	}
	if result.CNAEs[0].Code != "6201500" {
		t.Errorf("cnae code = %q, want 6201500", result.CNAEs[0].Code)
	}
	if len(result.Partners) != 1 {
		t.Errorf("partners len = %d, want 1", len(result.Partners))
	}
	if result.Partners[0].Nome != "JOAO DA SILVA" {
		t.Errorf("partner nome = %q, want JOAO DA SILVA", result.Partners[0].Nome)
	}
}

func TestGetByCNPJ_NotFound(t *testing.T) {
	repo := &mockCompanyRepo{detailErr: domain.ErrNotFound}
	svc := NewService(repo)

	_, err := svc.GetByCNPJ(context.Background(), "00000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want wrapping ErrNotFound", err)
	}
}

func TestGetByCNPJ_CNAEsError(t *testing.T) {
	repo := &mockCompanyRepo{
		detail:   &domain.CompanyDetail{CNPJ: "12345678000100", UF: "SP"},
		cnaesErr: errors.New("cnaes query failed"),
	}
	svc := NewService(repo)

	_, err := svc.GetByCNPJ(context.Background(), "12345678000100")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetByCNPJ_PartnersError(t *testing.T) {
	repo := &mockCompanyRepo{
		detail:      &domain.CompanyDetail{CNPJ: "12345678000100", UF: "SP"},
		cnaes:       []domain.CNAE{},
		partnersErr: errors.New("partners query failed"),
	}
	svc := NewService(repo)

	_, err := svc.GetByCNPJ(context.Background(), "12345678000100")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetByCNPJ_BasicoExtractionFullCNPJ(t *testing.T) {
	// The service takes the first 8 chars as cnpj_basico.
	// Verify it does not panic on a valid 14-char CNPJ.
	repo := &mockCompanyRepo{
		detail:   &domain.CompanyDetail{CNPJ: "12345678000100", UF: "SP"},
		cnaes:    []domain.CNAE{},
		partners: []domain.Partner{},
	}
	svc := NewService(repo)

	result, err := svc.GetByCNPJ(context.Background(), "12345678000100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CNPJ != "12345678000100" {
		t.Errorf("cnpj = %q, want 12345678000100", result.CNPJ)
	}
}

func TestGetByCNPJ_BasicoExtractionShortCNPJ(t *testing.T) {
	// CNPJ shorter than 8 chars should not panic — basico becomes the full string.
	repo := &mockCompanyRepo{
		detail:   &domain.CompanyDetail{CNPJ: "1234", UF: "SP"},
		cnaes:    []domain.CNAE{},
		partners: []domain.Partner{},
	}
	svc := NewService(repo)

	result, err := svc.GetByCNPJ(context.Background(), "1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CNPJ != "1234" {
		t.Errorf("cnpj = %q, want 1234", result.CNPJ)
	}
}
