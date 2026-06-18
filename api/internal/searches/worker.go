package searches

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

	"github.com/redis/go-redis/v9"
	"github.com/vantaggio/prospect-api/internal/domain"
)

const (
	queueKey    = "queue:searches"
	blpopTimeout = 5 * time.Second
)

type Worker struct {
	repo   Repository
	queue  *redis.Client
	client *http.Client
}

func NewWorker(repo Repository, queue *redis.Client) *Worker {
	return &Worker{
		repo:   repo,
		queue:  queue,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Run processes searches from the Redis queue until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := w.queue.BLPop(ctx, blpopTimeout, queueKey).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// BLPop timeout — normal, loop again
			continue
		}
		if len(res) < 2 {
			continue
		}
		searchID := res[1]
		if err := w.process(ctx, searchID); err != nil {
			slog.ErrorContext(ctx, "worker: process search failed", "search_id", searchID, "error", err)
		}
	}
}

func (w *Worker) process(ctx context.Context, searchID string) error {
	search, err := w.repo.GetByIDForWorker(ctx, searchID)
	if err != nil {
		return fmt.Errorf("get search: %w", err)
	}

	procStatus := domain.SearchStatusProc
	if err := w.repo.UpdateStatus(ctx, searchID, procStatus, nil, nil); err != nil {
		return fmt.Errorf("update to processing: %w", err)
	}

	count, execErr := w.execute(ctx, search)

	if execErr != nil {
		msg := execErr.Error()
		failStatus := domain.SearchStatusFailed
		if err := w.repo.UpdateStatus(ctx, searchID, failStatus, nil, &msg); err != nil {
			slog.ErrorContext(ctx, "worker: update to failed", "search_id", searchID, "error", err)
		}
		return execErr
	}

	doneStatus := domain.SearchStatusDone
	if err := w.repo.UpdateStatus(ctx, searchID, doneStatus, &count, nil); err != nil {
		return fmt.Errorf("update to done: %w", err)
	}
	slog.InfoContext(ctx, "worker: search done", "search_id", searchID, "count", count, "mode", search.Mode)
	return nil
}

func (w *Worker) execute(ctx context.Context, s *domain.Search) (int, error) {
	switch s.Mode {
	case domain.SearchModeStructured:
		return w.repo.RunStructuredSearch(ctx, s.ID, s.Filters)

	case domain.SearchModeSemantic:
		if s.QueryText == nil || *s.QueryText == "" {
			return 0, fmt.Errorf("query text is empty for semantic search")
		}
		vec, err := w.embedText(ctx, *s.QueryText)
		if err != nil {
			return 0, fmt.Errorf("embed query: %w", err)
		}
		return w.repo.RunSemanticSearch(ctx, s.ID, s.Filters, vec)

	default:
		return 0, fmt.Errorf("unknown search mode: %s", s.Mode)
	}
}

// ---- OpenAI embedding ----

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (w *Worker) embedText(ctx context.Context, text string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	payload, err := json.Marshal(embeddingRequest{Input: []string{text}, Model: model})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call openai: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result embeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal embedding response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai error (%s): %s", result.Error.Type, result.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai unexpected status %d", resp.StatusCode)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("openai returned empty embedding")
	}
	return result.Data[0].Embedding, nil
}
