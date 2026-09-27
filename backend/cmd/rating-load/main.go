// Command rating-load runs one batch of staging.rating_incoming → mdm.rating.
//
// Vertical-slice CLI for the Rating domain. Mirrors cmd/account-load and
// cmd/security-load: thin shell over datapipeline.TwoPoolLoader.LoadRatingBatch.
//
//	ALPHA_DSN / DATABASE_URL  — control plane (attribute_def; unused for rating)
//	CRIMS_DSN                 — tenant data plane (crims)
//	TENANT_ID                 — required
//	BATCH_SIZE                — default 100
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
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
	batchSize := 100
	if s := os.Getenv("BATCH_SIZE"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			batchSize = n
		}
	}

	// alpha connection is required by TwoPoolLoader (it has AlphaPool +
	// DataPool fields). Rating does not read from alpha, but the loader
	// type requires the connection be present.
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

	loader := &datapipeline.TwoPoolLoader{
		AlphaPool:            alpha,
		DataPool:             data,
		RequireSemanticTerms: false, // rating facts do not need semantic-term validation
	}
	loaded, rejected, err := loader.LoadRatingBatch(context.Background(), tenantID, batchSize)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	fmt.Printf("rating-load complete: loaded=%d rejected=%d\n", loaded, rejected)
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
	path := strings.TrimPrefix(u.Path, "/")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[:i]
	}
	u.Path = "/" + db
	return u.String()
}
