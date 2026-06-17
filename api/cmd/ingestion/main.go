package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vantaggio/prospect-api/pkg/db"
)

func main() {
	csvDir := flag.String("csv-dir", "./DADOS", "directory containing the Receita Federal CSV subdirectories")
	batchSize := flag.Int("batch-size", 1000, "rows per INSERT batch")
	embeddingBatch := flag.Int("embedding-batch", 100, "companies per OpenAI embeddings call")
	skipEmbeddings := flag.Bool("skip-embeddings", false, "skip embedding generation (useful for first-pass import)")
	skipHNSW := flag.Bool("skip-hnsw", false, "skip HNSW index creation")
	progressFile := flag.String("progress-file", ".ingestion_progress", "file tracking completed CSV files for resumability")
	flag.Parse()

	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	imp := NewImporter(pool, *batchSize)
	progress := loadProgress(*progressFile)

	log.Println("creating staging tables...")
	if err := imp.CreateStagingTables(ctx); err != nil {
		log.Fatalf("create staging tables: %v", err)
	}

	// 1. CNAEs
	if err := ingestCNAEs(ctx, imp, *csvDir, progress, *progressFile); err != nil {
		log.Fatalf("ingest cnaes: %v", err)
	}

	// 2. Municípios — loaded into memory (only ~5600 rows)
	munMap, err := loadMunicipios(*csvDir)
	if err != nil {
		log.Fatalf("load municipios: %v", err)
	}
	log.Printf("municípios: %d loaded", len(munMap))

	// 3. Empresas → staging_empresas
	if err := ingestEmpresas(ctx, imp, *csvDir, *batchSize, progress, *progressFile); err != nil {
		log.Fatalf("ingest empresas: %v", err)
	}

	// 4. Simples → staging_simples
	if err := ingestSimples(ctx, imp, *csvDir, *batchSize, progress, *progressFile); err != nil {
		log.Fatalf("ingest simples: %v", err)
	}

	// 5. Estabelecimentos → tb_companies + tb_company_cnaes
	if err := ingestEstabelecimentos(ctx, imp, *csvDir, *batchSize, munMap, progress, *progressFile); err != nil {
		log.Fatalf("ingest estabelecimentos: %v", err)
	}

	// 6. Sócios → tb_partners
	if err := ingestSocios(ctx, imp, *csvDir, *batchSize, progress, *progressFile); err != nil {
		log.Fatalf("ingest socios: %v", err)
	}

	log.Println("dropping staging tables...")
	if err := imp.DropStagingTables(ctx); err != nil {
		log.Printf("warn: drop staging tables: %v", err)
	}

	// 7. Embeddings
	if !*skipEmbeddings {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("OPENAI_API_KEY not set; use --skip-embeddings to skip")
		}
		model := os.Getenv("AI_EMBEDDING_MODEL")
		if model == "" {
			model = "text-embedding-3-small"
		}
		embedder := NewEmbedder(pool, apiKey, model, *embeddingBatch)
		log.Println("generating embeddings...")
		if err := embedder.RunAll(ctx); err != nil {
			log.Fatalf("embeddings: %v", err)
		}
	}

	// 8. HNSW index (only after embeddings exist)
	if !*skipHNSW && !*skipEmbeddings {
		log.Println("creating HNSW index (this may take several minutes)...")
		if err := imp.CreateHNSWIndex(ctx); err != nil {
			log.Fatalf("create HNSW index: %v", err)
		}
		log.Println("HNSW index created")
	}

	log.Println("ingestion complete")
}

// ---- ingestion phases ----

func ingestCNAEs(ctx context.Context, imp *Importer, csvDir string, progress map[string]bool, progressFile string) error {
	files, err := findFiles(csvDir, "Cnaes")
	if err != nil {
		return err
	}
	for _, path := range files {
		key := relKey(csvDir, path)
		if progress[key] {
			log.Printf("skip (done): %s", key)
			continue
		}
		log.Printf("cnaes: %s", filepath.Base(path))
		rows, err := ParseCNAEs(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := imp.UpsertCNAEs(ctx, rows); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("cnaes: %d rows", len(rows))
		markDone(progressFile, key)
		progress[key] = true
	}
	return nil
}

func loadMunicipios(csvDir string) (map[int]string, error) {
	files, err := findFiles(csvDir, "Municipios")
	if err != nil {
		return nil, err
	}
	munMap := make(map[int]string)
	for _, path := range files {
		rows, err := ParseMunicipios(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for _, r := range rows {
			munMap[r.Code] = r.Name
		}
	}
	return munMap, nil
}

func ingestEmpresas(ctx context.Context, imp *Importer, csvDir string, batchSize int, progress map[string]bool, progressFile string) error {
	files, err := findFiles(csvDir, "Empresas")
	if err != nil {
		return err
	}
	for _, path := range files {
		key := relKey(csvDir, path)
		if progress[key] {
			log.Printf("skip (done): %s", key)
			continue
		}
		log.Printf("empresas: %s", filepath.Base(path))
		count, err := streamAndBatch(batchSize, func(flush func([]EmpresaRow) error) error {
			return StreamEmpresas(path, func(r EmpresaRow) error {
				return flush([]EmpresaRow{r})
			})
		}, func(batch []EmpresaRow) error {
			return imp.UpsertEmpresas(ctx, batch)
		})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("empresas: %d rows", count)
		markDone(progressFile, key)
		progress[key] = true
	}
	return nil
}

func ingestSimples(ctx context.Context, imp *Importer, csvDir string, batchSize int, progress map[string]bool, progressFile string) error {
	files, err := findFiles(csvDir, "Simples")
	if err != nil {
		return err
	}
	for _, path := range files {
		key := relKey(csvDir, path)
		if progress[key] {
			log.Printf("skip (done): %s", key)
			continue
		}
		log.Printf("simples: %s", filepath.Base(path))
		count, err := streamAndBatch(batchSize, func(flush func([]SimplesRow) error) error {
			return StreamSimples(path, func(r SimplesRow) error {
				return flush([]SimplesRow{r})
			})
		}, func(batch []SimplesRow) error {
			return imp.UpsertSimples(ctx, batch)
		})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("simples: %d rows", count)
		markDone(progressFile, key)
		progress[key] = true
	}
	return nil
}

func ingestEstabelecimentos(ctx context.Context, imp *Importer, csvDir string, batchSize int, munMap map[int]string, progress map[string]bool, progressFile string) error {
	files, err := findFiles(csvDir, "Estabelecimentos")
	if err != nil {
		return err
	}
	for _, path := range files {
		key := relKey(csvDir, path)
		if progress[key] {
			log.Printf("skip (done): %s", key)
			continue
		}
		log.Printf("estabelecimentos: %s", filepath.Base(path))
		count, err := streamAndBatch(batchSize, func(flush func([]EstabelecimentoRow) error) error {
			return StreamEstabelecimentos(path, func(r EstabelecimentoRow) error {
				return flush([]EstabelecimentoRow{r})
			})
		}, func(batch []EstabelecimentoRow) error {
			return imp.UpsertEstabelecimentos(ctx, batch, munMap)
		})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("estabelecimentos: %d rows", count)
		markDone(progressFile, key)
		progress[key] = true
	}
	return nil
}

func ingestSocios(ctx context.Context, imp *Importer, csvDir string, batchSize int, progress map[string]bool, progressFile string) error {
	files, err := findFiles(csvDir, "Socios")
	if err != nil {
		return err
	}
	for _, path := range files {
		key := relKey(csvDir, path)
		if progress[key] {
			log.Printf("skip (done): %s", key)
			continue
		}
		log.Printf("socios: %s", filepath.Base(path))
		count, err := streamAndBatch(batchSize, func(flush func([]SocioRow) error) error {
			return StreamSocios(path, func(r SocioRow) error {
				return flush([]SocioRow{r})
			})
		}, func(batch []SocioRow) error {
			return imp.UpsertSocios(ctx, batch)
		})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("socios: %d rows", count)
		markDone(progressFile, key)
		progress[key] = true
	}
	return nil
}

// ---- generic batch helper ----

// streamAndBatch collects rows into batches of batchSize and calls flush for each.
// It returns the total number of rows processed.
//
// producer calls flush with each row individually; streamAndBatch accumulates them
// and calls the real flush function when the batch is full (or at the end).
func streamAndBatch[T any](batchSize int, producer func(flush func([]T) error) error, flush func([]T) error) (int, error) {
	var buf []T
	total := 0

	err := producer(func(rows []T) error {
		buf = append(buf, rows...)
		if len(buf) >= batchSize {
			if err := flush(buf); err != nil {
				return err
			}
			total += len(buf)
			buf = buf[:0]
		}
		return nil
	})
	if err != nil {
		return total, err
	}
	// Flush remainder
	if len(buf) > 0 {
		if err := flush(buf); err != nil {
			return total, err
		}
		total += len(buf)
	}
	return total, nil
}

// ---- progress tracking ----

func loadProgress(path string) map[string]bool {
	f, err := os.Open(path)
	if err != nil {
		return make(map[string]bool)
	}
	defer f.Close()
	done := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			done[line] = true
		}
	}
	return done
}

func markDone(path, key string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("warn: progress file: %v", err)
		return
	}
	defer f.Close()
	fmt.Fprintln(f, key)
}

// ---- file helpers ----

// findFiles returns all regular files inside subdirectories of csvDir whose
// names start with subDirPrefix (case-sensitive).
func findFiles(csvDir, subDirPrefix string) ([]string, error) {
	entries, err := os.ReadDir(csvDir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", csvDir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), subDirPrefix) {
			continue
		}
		subDir := filepath.Join(csvDir, e.Name())
		inner, err := os.ReadDir(subDir)
		if err != nil {
			return nil, err
		}
		for _, f := range inner {
			if !f.IsDir() {
				files = append(files, filepath.Join(subDir, f.Name()))
			}
		}
	}
	return files, nil
}

// relKey returns a stable, relative key for the progress file.
func relKey(csvDir, path string) string {
	rel, err := filepath.Rel(csvDir, path)
	if err != nil {
		return path
	}
	return rel
}
