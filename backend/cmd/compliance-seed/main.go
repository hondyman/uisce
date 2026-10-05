package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/compliance"
)

func main() {
	log.Println("Starting Core Compliance Rule Library Catalog Graph Seeder...")

	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
		keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

		if _, err := os.Stat(caPath); err == nil {
			dsn = fmt.Sprintf("host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
		} else {
			dsn = "host=localhost port=5432 user=postgres password=postgres dbname=alpha sslmode=disable"
		}
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenantID, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		log.Fatalf("Failed to resolve gold copy tenant ID: %v", err)
	}

	rules, err := loader.LoadTenantActiveRules(ctx, goldTenantID)
	if err != nil {
		log.Fatalf("Failed to load gold copy rules: %v", err)
	}

	log.Printf("Loaded %d gold-copy compliance rules from compliance_rule table", len(rules))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Seed COMPLIANCE_RULESET Nodes
	rulesets := []struct {
		Code        string
		Name        string
		Description string
	}{
		{
			Code:        "CORE_REGULATORY",
			Name:        "Core Regulatory Pack",
			Description: "Mandatory statutory compliance rules covering UCITS, 1940-Act, 144A/Reg S, Sanctions, CSDR, and MiFID II reporting",
		},
		{
			Code:        "MARKET_CONDUCT",
			Name:        "Market Conduct Pack",
			Description: "Market abuse and trading conduct controls covering Reg M, Wash Sales, Front Running, Reg T, Spoofing, and RTS 27/28",
		},
		{
			Code:        "INSTITUTIONAL_CONTROLS",
			Name:        "Institutional Controls Pack",
			Description: "Enterprise risk and operational safeguards covering Fat Finger, Price Collars, STP, Margin, Personal Trading, and EMIR",
		},
	}

	ruleTypeID := uuid.MustParse("e39856ec-e9e2-4151-836a-cc93b801fe6c")   // validation_rule
	bundleTypeID := uuid.MustParse("5ed74b46-137a-424c-8aec-e84db8dcfcdf") // rule_bundle

	for _, rs := range rulesets {
		qPath := fmt.Sprintf("compliance.ruleset/%s", rs.Code)
		propsJSON, _ := json.Marshal(map[string]interface{}{
			"ruleset_code": rs.Code,
			"is_active":    true,
			"gold_copy":    true,
		})

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (
				id, node_type_id, node_type, tenant_id, node_name, qualified_path, description, properties, is_active, created_at, updated_at
			) VALUES (
				gen_random_uuid(), $1, 'COMPLIANCE_RULESET', $2, $3, $4, $5, $6, true, now(), now()
			)
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				updated_at = now()
		`, bundleTypeID, goldTenantID, rs.Name, qPath, rs.Description, propsJSON)
		if err != nil {
			log.Printf("Warning: ruleset catalog node upsert for %s: %v", rs.Code, err)
		}
	}

	// 2. Seed COMPLIANCE_RULE Nodes
	for _, r := range rules {
		node := r.ToCatalogNode()
		propsJSON, _ := json.Marshal(node.Properties)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (
				id, node_type_id, node_type, tenant_id, node_name, qualified_path, description, properties, is_active, created_at, updated_at
			) VALUES (
				$1, $2, 'COMPLIANCE_RULE', $3, $4, $5, $6, $7, true, now(), now()
			)
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				updated_at = now()
		`, r.ID, ruleTypeID, goldTenantID, r.Name, node.QualifiedPath, r.Description, propsJSON)
		if err != nil {
			log.Printf("Warning: rule catalog node upsert for %s: %v", r.RuleCode, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit catalog sync: %v", err)
	}

	log.Printf("Successfully synchronized %d compliance rules and %d rulesets into catalog graph!", len(rules), len(rulesets))
}
