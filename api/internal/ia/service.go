package ia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

const (
	qualifyCost    = 10
	maxQualifyAge  = 30 * 24 * time.Hour
	maxAISemaphore = 5
)

var (
	ErrInsufficientCredits = domain.ErrInsufficientCredits
	ErrCompanyNotFound     = domain.ErrNotFound
)

type ServiceInterface interface {
	Qualify(ctx context.Context, orgID, userID, cnpj string) (*QualifyResult, error)
	ListQualifications(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error)
}

type QualifyResult struct {
	QualificationID string `json:"qualification_id"`
	CNPJ            string `json:"cnpj"`
	Score           int16  `json:"score"`
	Justification   string `json:"justification"`
	Model           string `json:"model"`
	CreditsUsed     int    `json:"credits_used"`
	FromCache       bool   `json:"from_cache,omitempty"`
}

type Service struct {
	repo       Repository
	credits    credits.ServiceInterface
	provider   LLMProvider
	semaphore  chan struct{}
}

func NewService(repo Repository, creditsSvc credits.ServiceInterface, provider LLMProvider) *Service {
	return &Service{
		repo:      repo,
		credits:   creditsSvc,
		provider:  provider,
		semaphore: make(chan struct{}, maxAISemaphore),
	}
}

const qualifySystemPrompt = `Você é um analista de prospecção B2B especializado no mercado brasileiro.
Analise os dados da empresa fornecida e retorne um score de 0 a 100 indicando
o potencial de conversão como lead, onde:
- 0-30: baixo potencial
- 31-60: potencial médio
- 61-80: alto potencial
- 81-100: potencial muito alto

Responda APENAS em JSON válido:
{"score": N, "justification": "texto em português explicando o score"}

Fatores positivos: capital social alto, situação ativa, CNAE de alto valor,
presença em grandes centros, empresa estabelecida (> 3 anos).
Fatores negativos: MEI, capital < 10k, situação inapta/baixada, setores saturados.`

func (s *Service) Qualify(ctx context.Context, orgID, userID, cnpj string) (*QualifyResult, error) {
	// Return cached result if recent enough.
	cached, err := s.repo.FindRecent(ctx, cnpj, orgID, maxQualifyAge)
	if err != nil {
		return nil, fmt.Errorf("check cache: %w", err)
	}
	if cached != nil {
		return &QualifyResult{
			QualificationID: cached.ID,
			CNPJ:            cached.CNPJ,
			Score:           cached.Score,
			Justification:   cached.Justification,
			Model:           cached.ModelUsed,
			CreditsUsed:     0,
			FromCache:       true,
		}, nil
	}

	company, err := s.repo.GetCompanyPromptData(ctx, cnpj)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrCompanyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}

	// Deduct credits within a transaction before calling AI.
	tx, err := s.credits.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	refID := cnpj
	if err := s.credits.Deduct(ctx, tx, orgID, userID, qualifyCost,
		domain.CreditTxAIQualification, &refID,
		fmt.Sprintf("Qualificação IA: %s", cnpj),
	); err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, domain.ErrInsufficientCredits) {
			return nil, ErrInsufficientCredits
		}
		return nil, fmt.Errorf("deduct credits: %w", err)
	}

	// Acquire semaphore slot.
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		_ = tx.Rollback(ctx)
		return nil, ctx.Err()
	}

	text, usage, aiErr := s.provider.Chat(ctx, qualifySystemPrompt, buildQualifyPrompt(company))
	if aiErr != nil {
		_ = tx.Rollback(ctx)
		slog.ErrorContext(ctx, "ai qualify failed", "cnpj", cnpj, "error", aiErr)
		return nil, fmt.Errorf("ai call: %w", aiErr)
	}

	score, justification, parseErr := parseQualifyResponse(text)
	if parseErr != nil {
		_ = tx.Rollback(ctx)
		slog.ErrorContext(ctx, "parse qualify response", "cnpj", cnpj, "raw", text, "error", parseErr)
		return nil, fmt.Errorf("parse ai response: %w", parseErr)
	}

	qual := &domain.AIQualification{
		CNPJ:          cnpj,
		OrgID:         orgID,
		UserID:        userID,
		Score:         score,
		Justification: justification,
		PromptUsed:    qualifySystemPrompt,
		ModelUsed:     s.provider.ModelName(),
		TokensInput:   usage.Input,
		TokensOutput:  usage.Output,
	}
	if err := s.repo.Save(ctx, qual); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("save qualification: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &QualifyResult{
		QualificationID: qual.ID,
		CNPJ:            qual.CNPJ,
		Score:           qual.Score,
		Justification:   qual.Justification,
		Model:           qual.ModelUsed,
		CreditsUsed:     qualifyCost,
	}, nil
}

func (s *Service) ListQualifications(ctx context.Context, orgID string, cnpj *string) ([]domain.AIQualification, error) {
	out, err := s.repo.List(ctx, orgID, cnpj)
	if err != nil {
		return nil, fmt.Errorf("list qualifications: %w", err)
	}
	return out, nil
}

func buildQualifyPrompt(c *CompanyPromptData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Empresa: %s\n", c.RazaoSocial))
	sb.WriteString(fmt.Sprintf("CNPJ: %s\n", c.CNPJ))
	sb.WriteString(fmt.Sprintf("UF: %s\n", c.UF))
	if c.Municipio != nil {
		sb.WriteString(fmt.Sprintf("Município: %s\n", *c.Municipio))
	}
	sb.WriteString(fmt.Sprintf("Situação Cadastral: %d\n", c.SituacaoCadastral))
	if c.CapitalSocial != nil {
		sb.WriteString(fmt.Sprintf("Capital Social: R$ %.2f\n", *c.CapitalSocial))
	}
	if c.DataInicio != nil {
		sb.WriteString(fmt.Sprintf("Data de Início: %s\n", *c.DataInicio))
	}
	if c.Porte != nil {
		sb.WriteString(fmt.Sprintf("Porte: %d\n", *c.Porte))
	}
	if c.OpcaoSimples != nil && *c.OpcaoSimples {
		sb.WriteString("Optante pelo Simples Nacional: Sim\n")
	}
	if c.PrimaryCNAE != nil {
		sb.WriteString(fmt.Sprintf("CNAE Principal: %s\n", *c.PrimaryCNAE))
	}
	return sb.String()
}

func parseQualifyResponse(raw string) (int16, string, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var parsed struct {
		Score         int    `json:"score"`
		Justification string `json:"justification"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return 0, "", fmt.Errorf("json unmarshal: %w (raw: %s)", err, raw)
	}
	if parsed.Score < 0 || parsed.Score > 100 {
		return 0, "", fmt.Errorf("score out of range: %d", parsed.Score)
	}
	if parsed.Justification == "" {
		return 0, "", fmt.Errorf("empty justification")
	}
	return int16(parsed.Score), parsed.Justification, nil
}
