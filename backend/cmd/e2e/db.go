package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

// e2eDB opens a short-lived Postgres connection for reading state directly,
// replacing the Hasura GraphQL queries these E2E tests previously used.
func e2eDB() (*sql.DB, error) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}
	return sql.Open("postgres", url)
}

// queryUMATaxSaved returns the same row the old Hasura query implicitly used
// (the first row of the table, unfiltered) so E2E assertions keep their prior meaning.
func queryUMATaxSaved(db *sql.DB) (float64, error) {
	var saved float64
	err := db.QueryRow("SELECT tax_saved FROM uma_accounts LIMIT 1").Scan(&saved)
	return saved, err
}

func queryPortfolioAlpha(db *sql.DB) (float64, error) {
	var alpha float64
	err := db.QueryRow("SELECT alpha FROM portfolios LIMIT 1").Scan(&alpha)
	return alpha, err
}
