package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/tenantauto"
)

func main() {
	var (
		dsn              = flag.String("dsn", "", "Postgres DSN (defaults to ALPHA_DB_URL or DATABASE_URL)")
		schemaRoot       = flag.String("schema-root", "", "Path to cube tenant schemas root")
		generatedDir     = flag.String("generated-dir", "", "Directory for generated Cube artifacts")
		tenantFilter     = flag.String("tenants", "", "Comma-separated tenant IDs to include")
		datasourceFilter = flag.String("datasources", "", "Comma-separated datasource IDs to include")
		dryRun           = flag.Bool("dry-run", false, "Preview changes without touching DB or filesystem")
		timeout          = flag.Duration("timeout", 2*time.Minute, "Overall timeout for the run")
		triggeredBy      = flag.String("triggered-by", "tenant-auto-cli", "Identifier recorded in tenant_provision_jobs")
	)

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg := tenantauto.Config{
		DSN:              resolveDSN(*dsn),
		SchemaRoot:       *schemaRoot,
		GeneratedDir:     *generatedDir,
		TenantFilter:     splitCSV(*tenantFilter),
		DatasourceFilter: splitCSV(*datasourceFilter),
		DryRun:           *dryRun,
		TriggeredBy:      strings.TrimSpace(*triggeredBy),
		Logger:           slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	result, err := tenantauto.Execute(ctx, cfg)
	if err != nil {
		log.Fatalf("tenant automation failed: %v", err)
	}

	slog.InfoContext(ctx, "tenant automation finished",
		"total", result.TotalRows,
		"filtered", result.FilteredOut,
		"written", result.Written,
		"failures", len(result.Failures),
		"dryRun", cfg.DryRun,
	)

	if len(result.Failures) > 0 {
		for _, failure := range result.Failures {
			slog.ErrorContext(ctx, "tenant provisioning failure",
				"tenant", failure.TenantID,
				"datasource", failure.DatasourceID,
				"err", failure.Err,
			)
		}
		os.Exit(1)
	}
}

func resolveDSN(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	if env := strings.TrimSpace(os.Getenv("ALPHA_DB_URL")); env != "" {
		return env
	}
	if env := strings.TrimSpace(os.Getenv("DATABASE_URL")); env != "" {
		return env
	}
	return "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	pieces := strings.Split(raw, ",")
	var out []string
	for _, piece := range pieces {
		trimmed := strings.TrimSpace(piece)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
