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

type SeedEntry struct {
  Code     string     `json:"code"`
  Name     string     `json:"name"`
  Children []SeedItem `json:"children"`
}

type SeedItem struct {
  Code string `json:"code"`
  Name string `json:"name"`
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
    _ = db.GetContext(ctx, &id, `SELECT id FROM lookup_values WHERE lookup_id = $1 AND value = $2 LIMIT 1`, lookupID, code)
    return id, nil
  }
  _, err := db.ExecContext(ctx, `INSERT INTO lookup_values (lookup_id, tenant_id, value, label, parent_id) VALUES ($1,$2,$3,$4,$5)`, lookupID, tenantID, code, label, parentID)
  if err != nil {
    return "", err
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

  var tenantID string
  err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants LIMIT 1`)
  if err != nil {
    log.Fatalf("no tenants found to seed: %v", err)
  }

  bs, _ := ioutil.ReadFile("backend/seeds/data_values.json")
  var seeds []SeedEntry
  if err := json.Unmarshal(bs, &seeds); err != nil {
    log.Fatalf("failed to parse data_values.json: %v", err)
  }

  lkupID, err := ensureLookup(ctx, db, tenantID, "data_values", "Data value and type categories")
  if err != nil {
    log.Fatalf("failed to create data_values lookup: %v", err)
  }

  parentIDs := map[string]string{}
  for _, s := range seeds {
    id, err := ensureValue(ctx, db, lkupID, tenantID, s.Code, s.Name, "")
    if err != nil {
      log.Fatalf("failed to insert top-level %s: %v", s.Code, err)
    }
    parentIDs[s.Code] = id
  }

  for _, s := range seeds {
    parentID := parentIDs[s.Code]
    for _, c := range s.Children {
      _, err := ensureValue(ctx, db, lkupID, tenantID, c.Code, c.Name, parentID)
      if err != nil {
        log.Printf("warning: failed to insert child %s.%s: %v", s.Code, c.Code, err)
      }
    }
  }

  fmt.Println("data_values seeding complete")
}