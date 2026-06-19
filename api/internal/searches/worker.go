package searches

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vantaggio/prospect-api/internal/credits"
	"github.com/vantaggio/prospect-api/internal/domain"
)

const (
	queueKey     = "queue:searches"
	blpopTimeout = 5 * time.Second
)

type Worker struct {
	repo      Repository
	queue     *redis.Client
	creditSvc credits.ServiceInterface
	client    *http.Client
}

func NewWorker(repo Repository, queue *redis.Client, creditSvc credits.ServiceInterface) *Worker {
	return &Worker{
		repo:      repo,
		queue:     queue,
		creditSvc: creditSvc,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

const staleMinutes = 5

// Run processes searches from the Redis queue until ctx is cancelled.
// On startup it recovers any searches that were stuck in "processing" when
// the server last shut down, and re-checks every staleMinutes minutes.
func (w *Worker) Run(ctx context.Context) {
	w.recoverStale(ctx)
	recoveryTicker := time.NewTicker(staleMinutes * time.Minute)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryTicker.C:
			w.recoverStale(ctx)
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

func (w *Worker) recoverStale(ctx context.Context) {
	n, err := w.repo.RecoverStaleSearches(ctx, staleMinutes)
	if err != nil {
		slog.ErrorContext(ctx, "worker: recover stale searches", "error", err)
		return
	}
	if n > 0 {
		slog.InfoContext(ctx, "worker: recovered stale searches", "count", n)
	}

	// Re-enqueue all searches that are queued in the DB but may have lost their
	// Redis entry (e.g. after a server restart or stale recovery).
	ids, err := w.repo.ListQueuedSearchIDs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "worker: list queued searches for requeue", "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	if err := w.queue.RPush(ctx, queueKey, args...).Err(); err != nil {
		slog.ErrorContext(ctx, "worker: requeue searches", "error", err)
		return
	}
	slog.InfoContext(ctx, "worker: requeued searches", "count", len(ids))
}

func (w *Worker) process(ctx context.Context, searchID string) error {
	search, err := w.repo.GetByIDForWorker(ctx, searchID)
	if err != nil {
		return fmt.Errorf("get search: %w", err)
	}

	// Idempotency: skip if already completed (can happen when re-queuing on startup).
	if search.Status == domain.SearchStatusDone || search.Status == domain.SearchStatusFailed {
		return nil
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

	// Deduct credits atomically in a transaction.
	// Structured searches are capped at 1 000 credits regardless of result size.
	const maxStructuredCredits = 1000
	if count > 0 && w.creditSvc != nil {
		creditsToDeduct := count
		if search.Mode == domain.SearchModeStructured && creditsToDeduct > maxStructuredCredits {
			creditsToDeduct = maxStructuredCredits
		}

		tx, err := w.creditSvc.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin credit tx: %w", err)
		}

		refID := search.ID
		deductErr := w.creditSvc.Deduct(ctx, tx, search.OrgID, search.UserID,
			creditsToDeduct,
			domain.CreditTxSearch,
			&refID,
			fmt.Sprintf("Busca: %d leads retornados", count),
		)
		if deductErr != nil {
			_ = tx.Rollback(ctx)
			msg := deductErr.Error()
			failStatus := domain.SearchStatusFailed
			if err := w.repo.UpdateStatus(ctx, searchID, failStatus, nil, &msg); err != nil {
				slog.ErrorContext(ctx, "worker: update to failed (credits)", "search_id", searchID, "error", err)
			}
			if errors.Is(deductErr, domain.ErrInsufficientCredits) {
				return nil // expected business error, not a worker fault
			}
			return deductErr
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit credit tx: %w", err)
		}
	}

	doneStatus := domain.SearchStatusDone
	if err := w.repo.UpdateStatus(ctx, searchID, doneStatus, &count, nil); err != nil {
		return fmt.Errorf("update to done: %w", err)
	}
	slog.InfoContext(ctx, "worker: search done", "search_id", searchID, "count", count, "mode", search.Mode)

	if count > 0 {
		go w.generateEmbeddingsForSearch(ctx, searchID)
	}
	return nil
}

const maxEmbedPerSearch = 100

func (w *Worker) generateEmbeddingsForSearch(ctx context.Context, searchID string) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return
	}
	inputs, err := w.repo.GetCompanyEmbedInputs(ctx, searchID, maxEmbedPerSearch)
	if err != nil {
		slog.ErrorContext(ctx, "worker: get embed inputs", "search_id", searchID, "error", err)
		return
	}
	if len(inputs) == 0 {
		return
	}

	texts := make([]string, len(inputs))
	for i, inp := range inputs {
		texts[i] = inp.Text
	}

	vecs, err := w.embedTexts(ctx, texts)
	if err != nil {
		slog.ErrorContext(ctx, "worker: embed company texts", "search_id", searchID, "error", err)
		return
	}

	embeddings := make([]domain.CompanyEmbedding, len(inputs))
	for i, inp := range inputs {
		embeddings[i] = domain.CompanyEmbedding{CNPJ: inp.CNPJ, UF: inp.UF, Vector: vecs[i]}
	}
	if err := w.repo.SaveEmbeddings(ctx, embeddings); err != nil {
		slog.ErrorContext(ctx, "worker: save embeddings", "search_id", searchID, "error", err)
		return
	}
	slog.InfoContext(ctx, "worker: embeddings cached", "search_id", searchID, "count", len(embeddings))
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
		return w.repo.RunSemanticSearch(ctx, s.ID, s.Filters, vec, *s.QueryText)

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
	vecs, err := w.embedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (w *Worker) embedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	payload, err := json.Marshal(embeddingRequest{Input: texts, Model: model})
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
	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("openai returned %d embeddings, expected %d", len(result.Data), len(texts))
	}

	vecs := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(vecs) {
			vecs[d.Index] = d.Embedding
		}
	}
	return vecs, nil
}
