// Command security-load runs one two-pool Security staging→master batch.
//
// OPS / DEBUG ONLY. Prefer Data Pipelines master_sink with entity_type=SECURITY.
//
//	ALPHA_DSN / DATABASE_URL  — control plane (attribute_def)
//	CRIMS_DSN                 — tenant data plane
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
	"github.com/hondyman/uisce/backend/internal/survivorship"
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

	alpha, err := sqlx.Open("pgx", alphaURL)
	if err != nil {
		log.Fatal(err)
	}
	defer alpha.Close()
	crims, err := sqlx.Open("pgx", crimsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer crims.Close()

	loader := &datapipeline.TwoPoolLoader{
		AlphaPool:            alpha,
		DataPool:             crims,
		Surv:                 survivorship.NewService(alpha),
		RequireSemanticTerms: true,
	}
	loaded, warnings, err := loader.LoadSecurityBatch(context.Background(), tenantID, 100)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("security-load complete: loaded=%d warnings=%d\n", loaded, warnings)
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func swapDB(dsn, db string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	u.Path = "/" + db
	return u.String()
}
