package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type IsoEntry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func ensureLookup(ctx context.Context, db *sqlx.DB, tenantID, name, description string) (string, error) {
	var id string
	err := db.GetContext(ctx, &id, `SELECT id FROM lookups WHERE tenant_id = $1 AND name = $2 LIMIT 1`, tenantID, name)
	if err == nil && id != "" {
		return id, nil
	}

	// Insert new lookup and return id
	err = db.QueryRowContext(ctx, `INSERT INTO lookups (tenant_id, name, description) VALUES ($1,$2,$3) RETURNING id`, tenantID, name, description).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func ensureValue(ctx context.Context, db *sqlx.DB, lookupID, tenantID, code, label string) error {
	var exists string
	_ = db.GetContext(ctx, &exists, `SELECT id FROM lookup_values WHERE lookup_id = $1 AND value = $2 LIMIT 1`, lookupID, code)
	if exists != "" {
		return nil
	}
	_, err := db.ExecContext(ctx, `INSERT INTO lookup_values (lookup_id, tenant_id, value, label) VALUES ($1,$2,$3,$4)`, lookupID, tenantID, code, label)
	return err
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Find first tenant
	var tenantID string
	err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants LIMIT 1`)
	if err != nil {
		log.Fatalf("no tenants found to seed: %v", err)
	}

	// Load datasets from backend/seeds
	// Countries
	bs, _ := ioutil.ReadFile("backend/seeds/iso_countries.json")
	var countries []IsoEntry
	if err := json.Unmarshal(bs, &countries); err != nil {
		log.Fatalf("failed to parse iso_countries.json: %v", err)
	}

	lkupID, err := ensureLookup(ctx, db, tenantID, "iso_countries", "ISO 3166 Country Codes")
	if err != nil {
		log.Fatalf("failed to create iso_countries lookup: %v", err)
	}

	for _, c := range countries {
		if err := ensureValue(ctx, db, lkupID, tenantID, c.Code, c.Name); err != nil {
			log.Printf("warning: failed to insert country %s: %v", c.Code, err)
		}
	}

	// Currencies
	bs, _ = ioutil.ReadFile("backend/seeds/iso_currencies.json")
	var currencies []IsoEntry
	if err := json.Unmarshal(bs, &currencies); err != nil {
		log.Fatalf("failed to parse iso_currencies.json: %v", err)
	}

	curID, err := ensureLookup(ctx, db, tenantID, "iso_currencies", "ISO 4217 Currency Codes")
	if err != nil {
		log.Fatalf("failed to create iso_currencies lookup: %v", err)
	}

	for _, c := range currencies {
		if err := ensureValue(ctx, db, curID, tenantID, c.Code, c.Name); err != nil {
			log.Printf("warning: failed to insert currency %s: %v", c.Code, err)
		}
	}

	fmt.Println("ISO code seeding complete")
}
