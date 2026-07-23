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

type SeedItem struct {
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Children []SeedItem `json:"children"`
}

type SeedEntry struct {
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Children []SeedItem `json:"children"`
}

func ensureLookup(ctx context.Context, db *sqlx.DB, tenantID, name, description string) (string, error) {
	var id string
	err := db.GetContext(ctx, &id, `SELECT id FROM lookups WHERE tenant_id = $1 AND name = $2 LIMIT 1`, tenantID, name)
	if err == nil && id != "" {
		return id, nil
	}

	err = db.QueryRowContext(ctx, `INSERT INTO lookups (tenant_id, name, description) VALUES ($1,$2,$3) RETURNING id`, tenantID, name, description).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func ensureValue(ctx context.Context, db *sqlx.DB, lookupID, tenantID, code, label, parentID string) (string, error) {
	var id string
	_ = db.GetContext(ctx, &id, `SELECT id FROM lookup_values WHERE lookup_id = $1 AND value = $2 LIMIT 1`, lookupID, code)
	if id != "" {
		return id, nil
	}

	if parentID == "" {
		_, err := db.ExecContext(ctx, `INSERT INTO lookup_values (lookup_id, tenant_id, value, label) VALUES ($1,$2,$3,$4)`, lookupID, tenantID, code, label)
		if err != nil {
			return "", err
		}
	} else {
		_, err := db.ExecContext(ctx, `INSERT INTO lookup_values (lookup_id, tenant_id, value, label, parent_id) VALUES ($1,$2,$3,$4,$5)`, lookupID, tenantID, code, label, parentID)
		if err != nil {
			return "", err
		}
	}

	_ = db.GetContext(ctx, &id, `SELECT id FROM lookup_values WHERE lookup_id = $1 AND value = $2 LIMIT 1`, lookupID, code)

	return id, nil
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

	// Get all tenants
	var tenantIDs []string
	err = db.SelectContext(ctx, &tenantIDs, `SELECT id FROM tenants`)
	if err != nil {
		log.Fatalf("failed to fetch tenants: %v", err)
	}

	if len(tenantIDs) == 0 {
		log.Fatalf("no tenants found to seed")
	}

	bs, _ := ioutil.ReadFile("backend/seeds/data_values.json")

	var seeds []SeedEntry
	if err := json.Unmarshal(bs, &seeds); err != nil {
		log.Fatalf("failed to parse data_values.json: %v", err)
	}

	// Seed for each tenant
	for _, tenantID := range tenantIDs {
		lkupID, err := ensureLookup(ctx, db, tenantID, "data_values", "Data value and type categories")
		if err != nil {
			log.Printf("failed to create data_values lookup for tenant %s: %v", tenantID, err)
			continue
		}

		parentIDs := map[string]string{}

		for _, s := range seeds {
			id, err := ensureValue(ctx, db, lkupID, tenantID, s.Code, s.Name, "")
			if err != nil {
				log.Printf("failed to insert top-level %s for tenant %s: %v", s.Code, tenantID, err)
				continue
			}
			parentIDs[s.Code] = id

			for _, c := range s.Children {
				parentID := parentIDs[s.Code]
				_, err := ensureValue(ctx, db, lkupID, tenantID, c.Code, c.Name, parentID)
				if err != nil {
					log.Printf("warning: failed to insert child %s.%s for tenant %s: %v", s.Code, c.Code, tenantID, err)
				}
			}
		}

		fmt.Printf("seeded data_values for tenant %s\n", tenantID)
	}

	fmt.Printf("data_values seeding complete for %d tenants\n", len(tenantIDs))
}
