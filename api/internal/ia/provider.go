package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type TokenUsage struct {
	Input  int
	Output int
}

// LLMProvider abstracts chat calls to an LLM backend.
type LLMProvider interface {
	// Chat sends a structured prompt and returns the raw text response.
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, TokenUsage, error)
	// ModelName returns the model identifier used.
	ModelName() string
}

type Config struct {
	Provider    string // "openai" | "gemini"
	ChatModel   string
	APIKey      string
	Temperature float64
	Timeout     time.Duration
}

func NewProvider(cfg Config) LLMProvider {
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ChatModel == "" {
		cfg.ChatModel = "gpt-4o-mini"
	}

	switch cfg.Provider {
	case "gemini":
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		}
		return &geminiProvider{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
	default: // openai
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		return &openaiProvider{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// OpenAI provider
// ──────────────────────────────────────────────────────────────────────────────

type openaiProvider struct {
	cfg    Config
	client *http.Client
}

func (p *openaiProvider) ModelName() string { return p.cfg.ChatModel }

func (p *openaiProvider) Chat(ctx context.Context, system, user string) (string, TokenUsage, error) {
	body := map[string]any{
		"model":       p.cfg.ChatModel,
		"temperature": p.cfg.Temperature,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	return p.doRequest(ctx, "https://api.openai.com/v1/chat/completions", body)
}

func (p *openaiProvider) doRequest(ctx context.Context, url string, body any) (string, TokenUsage, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := range maxRetries {
		text, usage, err := p.sendOpenAI(ctx, url, body)
		if err == nil {
			return text, usage, nil
		}
		lastErr = err
		if attempt < maxRetries-1 {
			// exponential backoff only on rate-limit errors
			if isRateLimit(err) {
				wait := time.Duration(1<<uint(attempt)) * time.Second
				slog.WarnContext(ctx, "openai rate limit, retrying", "attempt", attempt+1, "wait", wait)
				select {
				case <-ctx.Done():
					return "", TokenUsage{}, ctx.Err()
				case <-time.After(wait):
				}
				continue
			}
		}
		break
	}
	return "", TokenUsage{}, lastErr
}

type rateLimitError struct{ msg string }

func (e *rateLimitError) Error() string { return e.msg }

func isRateLimit(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}

func (p *openaiProvider) sendOpenAI(ctx context.Context, url string, body any) (string, TokenUsage, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", TokenUsage{}, &rateLimitError{msg: "rate limit (429)"}
	}
	if resp.StatusCode != http.StatusOK {
		return "", TokenUsage{}, fmt.Errorf("openai status %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", TokenUsage{}, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", TokenUsage{}, fmt.Errorf("no choices in openai response")
	}

	return result.Choices[0].Message.Content, TokenUsage{
		Input:  result.Usage.PromptTokens,
		Output: result.Usage.CompletionTokens,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Gemini provider
// ──────────────────────────────────────────────────────────────────────────────

type geminiProvider struct {
	cfg    Config
	client *http.Client
}

func (p *geminiProvider) ModelName() string { return p.cfg.ChatModel }

func (p *geminiProvider) Chat(ctx context.Context, system, user string) (string, TokenUsage, error) {
	if p.cfg.ChatModel == "gpt-4o-mini" {
		p.cfg.ChatModel = "gemini-1.5-flash"
	}
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		p.cfg.ChatModel, p.cfg.APIKey,
	)

	body := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": system}},
		},
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": user}}},
		},
		"generationConfig": map[string]any{
			"temperature": p.cfg.Temperature,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", TokenUsage{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", TokenUsage{}, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", TokenUsage{}, fmt.Errorf("no content in gemini response")
	}

	return result.Candidates[0].Content.Parts[0].Text, TokenUsage{
		Input:  result.UsageMetadata.PromptTokenCount,
		Output: result.UsageMetadata.CandidatesTokenCount,
	}, nil
}
