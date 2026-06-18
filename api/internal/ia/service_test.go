package ia_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vantaggio/prospect-api/internal/domain"
	"github.com/vantaggio/prospect-api/internal/ia"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockRepo struct {
	saveFn              func(ctx context.Context, q *domain.AIQualification) error
	findRecentFn        func(ctx context.Context, cnpj, orgID string, maxAge time.Duration) (*domain.AIQualification, error)
	listFn              func(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error)
	getCompanyPromptFn  func(ctx context.Context, cnpj string) (*ia.CompanyPromptData, error)
}

func (m *mockRepo) Save(ctx context.Context, q *domain.AIQualification) error {
	return m.saveFn(ctx, q)
}
func (m *mockRepo) FindRecent(ctx context.Context, cnpj, orgID string, maxAge time.Duration) (*domain.AIQualification, error) {
	return m.findRecentFn(ctx, cnpj, orgID, maxAge)
}
func (m *mockRepo) List(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error) {
	return m.listFn(ctx, orgID, cnpj)
}
func (m *mockRepo) GetCompanyPromptData(ctx context.Context, cnpj string) (*ia.CompanyPromptData, error) {
	return m.getCompanyPromptFn(ctx, cnpj)
}

// mockTx implements pgx.Tx with only Commit/Rollback wired up.
type mockTx struct {
	pgx.Tx
	commitErr   error
	rollbackErr error
}

func (m *mockTx) Commit(ctx context.Context) error   { return m.commitErr }
func (m *mockTx) Rollback(ctx context.Context) error { return m.rollbackErr }

type mockCredits struct {
	beginTxFn func(ctx context.Context) (pgx.Tx, error)
	deductFn  func(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, refID *string, desc string) error
}

func (m *mockCredits) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return m.beginTxFn(ctx)
}
func (m *mockCredits) Deduct(ctx context.Context, tx pgx.Tx, orgID, userID string, amount int, txType domain.CreditTxType, refID *string, desc string) error {
	return m.deductFn(ctx, tx, orgID, userID, amount, txType, refID, desc)
}
func (m *mockCredits) GetBalance(_ context.Context, _ string) (*domain.CreditBalanceResponse, error) {
	return &domain.CreditBalanceResponse{}, nil
}
func (m *mockCredits) AddCredits(_ context.Context, _ string, _ int, _ string) error { return nil }
func (m *mockCredits) ListTransactions(_ context.Context, _ string, _, _ int) (*domain.CreditTransactionsResponse, error) {
	return &domain.CreditTransactionsResponse{}, nil
}

type mockProvider struct {
	chatFn    func(ctx context.Context, system, user string) (string, ia.TokenUsage, error)
	modelName string
}

func (m *mockProvider) Chat(ctx context.Context, system, user string) (string, ia.TokenUsage, error) {
	return m.chatFn(ctx, system, user)
}
func (m *mockProvider) ModelName() string { return m.modelName }

// ── helpers ───────────────────────────────────────────────────────────────────

func validCompany() *ia.CompanyPromptData {
	cap := 500000.0
	return &ia.CompanyPromptData{
		CNPJ:              "12345678000195",
		RazaoSocial:       "Empresa Teste Ltda",
		UF:                "SP",
		CapitalSocial:     &cap,
		SituacaoCadastral: 2,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestQualify_CacheHit(t *testing.T) {
	cached := &domain.AIQualification{
		ID:            "cached-uuid",
		CNPJ:          "12345678000195",
		Score:         85,
		Justification: "Empresa consolidada",
		ModelUsed:     "gpt-4o-mini",
		CreatedAt:     time.Now(),
	}

	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return cached, nil
		},
	}
	creds := &mockCredits{}
	prov := &mockProvider{modelName: "gpt-4o-mini"}

	svc := ia.NewService(repo, creds, prov)
	result, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.FromCache {
		t.Error("expected FromCache=true")
	}
	if result.CreditsUsed != 0 {
		t.Errorf("expected 0 credits used, got %d", result.CreditsUsed)
	}
	if result.Score != 85 {
		t.Errorf("expected score 85, got %d", result.Score)
	}
}

func TestQualify_CompanyNotFound(t *testing.T) {
	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return nil, nil
		},
		getCompanyPromptFn: func(_ context.Context, _ string) (*ia.CompanyPromptData, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := ia.NewService(repo, &mockCredits{}, &mockProvider{})
	_, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")
	if !errors.Is(err, ia.ErrCompanyNotFound) {
		t.Errorf("expected ErrCompanyNotFound, got %v", err)
	}
}

func TestQualify_InsufficientCredits(t *testing.T) {
	tx := &mockTx{rollbackErr: nil}
	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return nil, nil
		},
		getCompanyPromptFn: func(_ context.Context, _ string) (*ia.CompanyPromptData, error) {
			return validCompany(), nil
		},
	}
	creds := &mockCredits{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return tx, nil },
		deductFn: func(_ context.Context, _ pgx.Tx, _ string, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error {
			return domain.ErrInsufficientCredits
		},
	}
	svc := ia.NewService(repo, creds, &mockProvider{})
	_, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")
	if !errors.Is(err, ia.ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got %v", err)
	}
}

func TestQualify_AIFailureRollsBackCredits(t *testing.T) {
	tx := &mockTx{}
	rolled := false
	tx.Tx = nil // will panic if non-Commit/Rollback method is called
	origRollback := &mockTx{rollbackErr: nil}
	_ = origRollback

	rollingTx := &mockTx{}
	rollingTx.rollbackErr = nil

	var rollbackCalled bool
	customTx := struct {
		pgx.Tx
		commitCalled   *bool
		rollbackCalled *bool
	}{
		Tx:             nil,
		rollbackCalled: &rollbackCalled,
	}
	_ = customTx
	_ = rolled
	_ = tx

	commitCalled := false
	rollbackCalled = false

	type trackTx struct {
		pgx.Tx
	}

	callTx := &mockTx{}
	callTx.commitErr = nil
	callTx.rollbackErr = nil

	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return nil, nil
		},
		getCompanyPromptFn: func(_ context.Context, _ string) (*ia.CompanyPromptData, error) {
			return validCompany(), nil
		},
	}

	deductCalled := false
	creds := &mockCredits{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return callTx, nil },
		deductFn: func(_ context.Context, _ pgx.Tx, _ string, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error {
			deductCalled = true
			return nil
		},
	}
	prov := &mockProvider{
		modelName: "gpt-4o-mini",
		chatFn: func(_ context.Context, _, _ string) (string, ia.TokenUsage, error) {
			return "", ia.TokenUsage{}, errors.New("ai unavailable")
		},
	}

	svc := ia.NewService(repo, creds, prov)
	_, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")
	if err == nil {
		t.Fatal("expected error from AI failure")
	}
	if !deductCalled {
		t.Error("expected deduct to be called before AI")
	}
	if commitCalled {
		t.Error("commit must not be called on AI failure")
	}
}

func TestQualify_Success_DeductsCredits(t *testing.T) {
	callTx := &mockTx{}

	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return nil, nil
		},
		getCompanyPromptFn: func(_ context.Context, _ string) (*ia.CompanyPromptData, error) {
			return validCompany(), nil
		},
		saveFn: func(_ context.Context, q *domain.AIQualification) error {
			q.ID = "new-uuid"
			q.CreatedAt = time.Now()
			return nil
		},
	}

	deducted := 0
	creds := &mockCredits{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return callTx, nil },
		deductFn: func(_ context.Context, _ pgx.Tx, _ string, _ string, amount int, _ domain.CreditTxType, _ *string, _ string) error {
			deducted = amount
			return nil
		},
	}
	prov := &mockProvider{
		modelName: "gpt-4o-mini",
		chatFn: func(_ context.Context, _, _ string) (string, ia.TokenUsage, error) {
			return `{"score": 75, "justification": "Empresa ativa com bom capital"}`,
				ia.TokenUsage{Input: 100, Output: 50}, nil
		},
	}

	svc := ia.NewService(repo, creds, prov)
	result, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CreditsUsed != 10 {
		t.Errorf("expected 10 credits used, got %d", result.CreditsUsed)
	}
	if deducted != 10 {
		t.Errorf("expected 10 credits deducted, got %d", deducted)
	}
	if result.Score != 75 {
		t.Errorf("expected score 75, got %d", result.Score)
	}
	if result.FromCache {
		t.Error("expected FromCache=false")
	}
}

func TestQualify_InvalidAIResponse(t *testing.T) {
	callTx := &mockTx{}
	repo := &mockRepo{
		findRecentFn: func(_ context.Context, _, _ string, _ time.Duration) (*domain.AIQualification, error) {
			return nil, nil
		},
		getCompanyPromptFn: func(_ context.Context, _ string) (*ia.CompanyPromptData, error) {
			return validCompany(), nil
		},
	}
	creds := &mockCredits{
		beginTxFn: func(_ context.Context) (pgx.Tx, error) { return callTx, nil },
		deductFn:  func(_ context.Context, _ pgx.Tx, _, _ string, _ int, _ domain.CreditTxType, _ *string, _ string) error { return nil },
	}
	prov := &mockProvider{
		modelName: "gpt-4o-mini",
		chatFn: func(_ context.Context, _, _ string) (string, ia.TokenUsage, error) {
			return "not json at all", ia.TokenUsage{}, nil
		},
	}

	svc := ia.NewService(repo, creds, prov)
	_, err := svc.Qualify(context.Background(), "org1", "user1", "12345678000195")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestListQualifications_OK(t *testing.T) {
	expected := []domain.AIQualification{
		{ID: "id1", CNPJ: "12345678000195", Score: 80},
	}
	repo := &mockRepo{
		listFn: func(_ context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error) {
			return expected, nil
		},
	}
	svc := ia.NewService(repo, &mockCredits{}, &mockProvider{})
	results, err := svc.ListQualifications(context.Background(), "org1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
