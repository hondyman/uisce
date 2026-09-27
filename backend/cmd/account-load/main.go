// Command account-load runs one two-pool Account staging→master batch.
//
// OPS / DEBUG ONLY. The product path is Data Pipelines:
//
//	Build → Data → Pipelines → "Account Master ← staging (survivorship)"
//
// Configure source hierarchy under Build → Data → Survivorship.
// Prefer master_sink over this CLI in steward workflows.
//
//	ALPHA_DSN / DATABASE_URL  — control plane (attribute_def)
//	CRIMS_DSN                 — tenant data plane (optional; derived from DATABASE_URL by swapping /alpha → /crims)
//	TENANT_ID                 — required
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
)

func main() {
	alphaURL := firstEnv("ALPHA_DSN", "DATABASE_URL")
	if alphaURL == "" {
		log.Fatal("ALPHA_DSN or DATABASE_URL required")
	}
	crimsURL := os.Getenv("CRIMS_DSN")
	if crimsURL == "" {
		crimsURL = swapDB(alphaURL, "crims")
	}
	tenantStr := os.Getenv("TENANT_ID")
	if tenantStr == "" {
		log.Fatal("TENANT_ID required")
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		log.Fatalf("TENANT_ID: %v", err)
	}

	alpha, err := sqlx.Connect("pgx", alphaURL)
	if err != nil {
		log.Fatalf("alpha: %v", err)
	}
	defer alpha.Close()
	data, err := sqlx.Connect("pgx", crimsURL)
	if err != nil {
		log.Fatalf("crims: %v", err)
	}
	defer data.Close()

	loader := &datapipeline.TwoPoolLoader{AlphaPool: alpha, DataPool: data}
	loaded, warnings, err := loader.LoadAccountBatch(context.Background(), tenantID, 100)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	fmt.Printf("account-load complete: loaded=%d warnings=%d\n", loaded, warnings)
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func swapDB(raw, db string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	// Path is "/dbname"
	path := strings.TrimPrefix(u.Path, "/")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[:i]
	}
	u.Path = "/" + db
	return u.String()
}
