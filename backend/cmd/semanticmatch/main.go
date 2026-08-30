package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/gob"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver for -write-edges

	"github.com/hondyman/uisce/backend/internal/semanticmatch"
)

func main() {
	var (
		bbPath     = flag.String("bb", "BB_Fields.csv", "Bloomberg fields CSV")
		glossPath  = flag.String("glossary", "Semantic_Definitions.csv", "glossary CSV")
		colsPath   = flag.String("columns", "columns.csv", "scanned columns CSV (ignored in -eval)")
		reportPath = flag.String("report", "match_report.csv", "review report output")
		model      = flag.String("model", "gemini-2.5-flash", "Gemini model for verification")
		embedModel = flag.String("embed-model", "text-embedding-004", "embedding model ('' disables)")
		rpm        = flag.Int("rpm", 60, "Gemini requests per minute")
		cacheDir   = flag.String("cache-dir", ".gemini-cache", "LLM response cache dir")
		vecCache   = flag.String("vec-cache", ".vector-index.gob", "term embedding cache file")

		evalMode   = flag.Bool("eval", false, "hold-out evaluation over the glossary")
		holdout    = flag.Float64("holdout-frac", 0.3, "fraction of glossary rows to hold out")
		evalReport = flag.String("eval-report", "eval_misses.txt", "hold-out miss list output")

		dsn        = flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres DSN (for -write-edges)")
		tenantID   = flag.String("tenant", "", "tenant ID for catalog nodes/edges")
		writeEdges = flag.Bool("write-edges", false, "persist matches as catalog edges")
	)
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// ---- Build registry + column set (mode-dependent) ----
	reg := semanticmatch.NewTermRegistry()
	if err := semanticmatch.LoadBloombergFields(reg, *bbPath); err != nil {
		log.Error("load bloomberg fields", "err", err)
		os.Exit(1)
	}

	var cols []semanticmatch.ColumnMeta
	var expected map[string]string

	if *evalMode {
		rows, err := semanticmatch.LoadGlossaryRows(*glossPath)
		if err != nil {
			log.Error("load glossary rows", "err", err)
			os.Exit(1)
		}
		seeds, heldOut := semanticmatch.SplitGlossary(rows, *holdout)
		semanticmatch.RegisterGlossaryRows(reg, seeds, false)
		semanticmatch.RegisterGlossaryRows(reg, heldOut, true) // aliases hidden
		cols = semanticmatch.ColumnsFromRows(heldOut)
		expected = semanticmatch.ExpectedMap(heldOut)
		log.Info("eval mode", "glossary_rows", len(rows), "seeds", len(seeds),
			"held_out_terms", len(heldOut), "eval_columns", len(cols))
	} else {
		if err := semanticmatch.LoadGlossary(reg, *glossPath); err != nil {
			log.Error("load glossary", "err", err)
			os.Exit(1)
		}
		var err error
		if cols, err = semanticmatch.LoadColumnsCSV(*colsPath); err != nil {
			log.Error("load columns", "err", err)
			os.Exit(1)
		}
	}
	reg.BuildIDF()
	log.Info("registry built", "terms", len(reg.Terms), "columns", len(cols))

	// ---- LLM + vectors ----
	var llm *semanticmatch.LLMVerifier
	var vecs *semanticmatch.VectorIndex
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey == "" {
		log.Warn("GEMINI_API_KEY not set; running in lexical-only mode")
	} else {
		client := semanticmatch.NewGeminiClient(apiKey, *model, *rpm, *cacheDir)
		llm = &semanticmatch.LLMVerifier{Client: client}
		if *embedModel != "" {
			vecs = buildOrLoadVectors(ctx, client, *embedModel, reg, *vecCache, log)
		}
	}

	cfg := semanticmatch.DefaultConfig()
	cfg.EmbedModel = *embedModel
	pipe := semanticmatch.NewPipeline(cfg, reg, llm)
	pipe.Vectors = vecs
	pipe.Log = log

	outcomes, err := pipe.Run(ctx, cols)
	if err != nil {
		log.Error("pipeline", "err", err)
		os.Exit(1)
	}

	// ---- Report / score ----
	if err := writeReport(*reportPath, outcomes, *model); err != nil {
		log.Error("write report", "err", err)
		os.Exit(1)
	}
	log.Info("report written", "path", *reportPath)

	if *evalMode {
		sum, err := semanticmatch.RunHoldoutEval(ctx, pipe, cols, expected)
		if err != nil {
			log.Error("eval", "err", err)
			os.Exit(1)
		}
		log.Info("hold-out eval",
			"total", sum.Total, "correct", sum.Correct,
			"accuracy", fmt.Sprintf("%.1f%%", sum.Accuracy*100),
			"by_method", fmt.Sprint(sum.ByMethod),
			"by_status", fmt.Sprint(sum.ByStatus))
		if err := writeMisses(*evalReport, sum, *model, *holdout); err != nil {
			log.Error("write eval report", "err", err)
			os.Exit(1)
		}
		log.Info("misses written", "path", *evalReport, "count", len(sum.Misses))
	}

	// ---- Persistence (dry run by default) ----
	if *writeEdges {
		if *evalMode {
			log.Error("-write-edges is disabled in -eval mode (never persist hold-out artifacts)")
			os.Exit(1)
		}
		if *dsn == "" || *tenantID == "" {
			log.Error("-write-edges requires -dsn and -tenant")
			os.Exit(1)
		}
		db, err := sql.Open("pgx", *dsn) // or "postgres" with lib/pq
		if err != nil {
			log.Error("open db", "err", err)
			os.Exit(1)
		}
		defer db.Close()

		resolver := semanticmatch.NewNodeResolver(db, *tenantID)
		n, err := semanticmatch.WriteEdges(ctx, db, *tenantID, *model, outcomes,
			func(c semanticmatch.ColumnMeta, t *semanticmatch.SemanticTerm) (string, string, bool) {
				return resolver.Resolve(ctx, c, t)
			})
		if err != nil {
			log.Error("write edges", "err", err)
			os.Exit(1)
		}
		log.Info("edges written", "count", n)
	}
}

func buildOrLoadVectors(ctx context.Context, c *semanticmatch.GeminiClient, model string,
	reg *semanticmatch.TermRegistry, cacheFile string, log *slog.Logger) *semanticmatch.VectorIndex {

	if f, err := os.Open(cacheFile); err == nil {
		var vi semanticmatch.VectorIndex
		if err := gob.NewDecoder(f).Decode(&vi); err == nil {
			f.Close()
			log.Info("vector index loaded from cache", "vectors", len(vi.IDs))
			return &vi
		}
		f.Close()
	}
	// Aliases are deliberately excluded from TermEmbeddingText, so this cache
	// is valid for both normal and -eval modes.
	texts := make([]string, len(reg.Terms))
	ids := make([]string, len(reg.Terms))
	for i, t := range reg.Terms {
		texts[i], ids[i] = semanticmatch.TermEmbeddingText(t), t.ID
	}
	log.Info("embedding registry (one-time)", "terms", len(texts))
	vecs, err := c.EmbedBatch(ctx, model, "RETRIEVAL_DOCUMENT", texts)
	if err != nil {
		log.Warn("registry embedding failed; continuing without vectors", "err", err)
		return nil
	}
	vi := semanticmatch.NewVectorIndex(ids, vecs)
	if dir := filepath.Dir(cacheFile); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	if f, err := os.Create(cacheFile); err == nil {
		_ = gob.NewEncoder(f).Encode(vi)
		f.Close()
	}
	return vi
}

func writeReport(path string, outcomes []semanticmatch.MatchOutcome, model string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"status", "method", "confidence", "table", "column", "data_type",
		"term_id", "term_name", "term_source", "reasoning", "suggested_new_term", "model"})
	for _, o := range outcomes {
		row := []string{o.Status, o.Method, fmt.Sprintf("%.3f", o.Confidence),
			o.Column.Table, o.Column.Column, o.Column.DataType, "", "", "", o.Reasoning, o.SuggestedNewTerm, model}
		if o.Term != nil {
			row[6], row[7], row[8] = o.Term.ID, o.Term.Name, o.Term.Source
		}
		_ = w.Write(row)
	}
	return nil
}

func writeMisses(path string, s *semanticmatch.EvalSummary, model string, holdout float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "# hold-out eval  model=%s  holdout_frac=%.2f  accuracy=%d/%d (%.1f%%)\n",
		model, holdout, s.Correct, s.Total, s.Accuracy*100)
	for _, m := range s.Misses {
		fmt.Fprintln(f, m)
	}
	return nil
}
