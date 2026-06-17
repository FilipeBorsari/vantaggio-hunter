package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Embedder struct {
	pool      *pgxpool.Pool
	apiKey    string
	model     string
	batchSize int
	client    *http.Client
}

func NewEmbedder(pool *pgxpool.Pool, apiKey, model string, batchSize int) *Embedder {
	return &Embedder{
		pool:      pool,
		apiKey:    apiKey,
		model:     model,
		batchSize: batchSize,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

type companyForEmbedding struct {
	CNPJ         string
	UF           string
	RazaoSocial  string
	CNAEDesc     string
	MunicipioNome string
	Situacao     int16
	Capital      float64
}

// RunAll generates embeddings for all companies that don't have one yet.
// It runs in a loop until no more pending companies remain.
func (e *Embedder) RunAll(ctx context.Context) error {
	total := 0
	for {
		companies, err := e.fetchPending(ctx)
		if err != nil {
			return fmt.Errorf("fetch pending: %w", err)
		}
		if len(companies) == 0 {
			break
		}

		texts := make([]string, len(companies))
		for i, c := range companies {
			texts[i] = buildEmbeddingText(c)
		}

		vectors, err := e.callAPI(ctx, texts)
		if err != nil {
			return fmt.Errorf("openai embeddings: %w", err)
		}

		if err := e.saveEmbeddings(ctx, companies, vectors); err != nil {
			return fmt.Errorf("save embeddings: %w", err)
		}

		total += len(companies)
		log.Printf("embeddings: %d done", total)
	}
	log.Printf("embeddings: all done (%d total)", total)
	return nil
}

func (e *Embedder) fetchPending(ctx context.Context) ([]companyForEmbedding, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT
			c.cnpj,
			c.uf,
			c.razao_social,
			COALESCE((
				SELECT cc2.description
				FROM tb_company_cnaes j
				JOIN tb_cnaes cc2 ON cc2.code = j.cnae_code
				WHERE j.cnpj = c.cnpj AND j.is_primary = true
				LIMIT 1
			), '') AS cnae_desc,
			COALESCE(c.municipio_nome, '') AS municipio_nome,
			c.situacao_cadastral,
			COALESCE(c.capital_social, 0)
		FROM tb_companies c
		WHERE c.embedding IS NULL
		LIMIT $1
	`, e.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []companyForEmbedding
	for rows.Next() {
		var c companyForEmbedding
		if err := rows.Scan(&c.CNPJ, &c.UF, &c.RazaoSocial, &c.CNAEDesc,
			&c.MunicipioNome, &c.Situacao, &c.Capital); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// buildEmbeddingText creates the text representation used for vector embedding.
// Format: "RAZAO SOCIAL | CNAE DESC | MUNICIPIO UF | situacao:N capital:N.NN"
func buildEmbeddingText(c companyForEmbedding) string {
	var sb strings.Builder
	sb.WriteString(c.RazaoSocial)
	if c.CNAEDesc != "" {
		sb.WriteString(" | ")
		sb.WriteString(c.CNAEDesc)
	}
	if c.MunicipioNome != "" {
		sb.WriteString(" | ")
		sb.WriteString(c.MunicipioNome)
		sb.WriteString(" ")
		sb.WriteString(c.UF)
	}
	sb.WriteString(fmt.Sprintf(" | situacao:%d capital:%.2f", c.Situacao, c.Capital))
	return sb.String()
}

// ---- OpenAI API ----

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

func (e *Embedder) callAPI(ctx context.Context, texts []string) ([][]float32, error) {
	payload, err := json.Marshal(embeddingRequest{Input: texts, Model: e.model})
	if err != nil {
		return nil, err
	}

	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		vectors, err := e.doRequest(ctx, payload)
		if err == nil {
			return vectors, nil
		}
		// Exponential backoff for rate limit or server errors.
		wait := time.Duration(1<<uint(attempt)) * time.Second
		log.Printf("embedding API error (attempt %d/%d): %v — retrying in %s", attempt+1, maxRetries, err, wait)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, fmt.Errorf("embedding API failed after %d attempts", maxRetries)
}

func (e *Embedder) doRequest(ctx context.Context, payload []byte) ([][]float32, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result embeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai error (%s): %s", result.Error.Type, result.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	vectors := make([][]float32, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(vectors) {
			vectors[d.Index] = d.Embedding
		}
	}
	return vectors, nil
}

// saveEmbeddings writes embedding vectors back to tb_companies.
func (e *Embedder) saveEmbeddings(ctx context.Context, companies []companyForEmbedding, vectors [][]float32) error {
	now := nullTime(time.Now())
	batch := &pgx.Batch{}
	for i, c := range companies {
		if i >= len(vectors) || len(vectors[i]) == 0 {
			continue
		}
		// Format vector as PostgreSQL array literal: '[0.1,0.2,...]'
		batch.Queue(`
			UPDATE tb_companies
			SET embedding = $1::vector, embedding_updated_at = $2
			WHERE cnpj = $3 AND uf = $4
		`, vectorLiteral(vectors[i]), now, c.CNPJ, c.UF)
	}
	return sendBatch(ctx, e.pool, batch)
}

// vectorLiteral converts a float32 slice to the pgvector text format "[v1,v2,...]".
func vectorLiteral(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteByte(']')
	return sb.String()
}
