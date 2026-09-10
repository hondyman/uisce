package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// withPublicSearchPath forces every physical connection opened from this DSN to
// start with search_path=public. The shared "alpha" database has a
// database-level `ALTER DATABASE alpha SET search_path = 'vend, public'`
// (another service's schema, unrelated to Uisce) - without this, unqualified
// table names like `tenants` silently resolve to that other schema's
// same-named-but-differently-shaped tables instead of public.tenants.
func withPublicSearchPath(dsn string) string {
	sep := "&"
	if !strings.Contains(dsn, "?") {
		sep = "?"
	}
	return dsn + sep + "options=" + url.QueryEscape("-c search_path=public")
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = os.Getenv("POSTGRES_DSN")
	}
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}
	dbURL = withPublicSearchPath(dbURL)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	sqlxDB := sqlx.NewDb(db, "postgres")
	_ = sqlxDB

	router := api.SetupRouter(db, nil, nil, nil, nil, nil, nil, nil, nil)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting main Uisce Unified API server on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
