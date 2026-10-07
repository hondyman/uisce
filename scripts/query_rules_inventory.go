package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

func getAlphaDSN() string {
	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
		keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

		if _, err := os.Stat(caPath); err == nil {
			dsn = fmt.Sprintf("host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
		}
	}
	return dsn
}

func main() {
	dsn := getAlphaDSN()
	if dsn == "" {
		log.Fatal("ALPHA_DSN not configured and certs not found")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT 
			r.rule_code,
			COALESCE(m.ruleset_code, 'NO_PACK') as pack_name,
			r.data_provenance,
			COALESCE(v.content_hash, '') as content_hash,
			COALESCE(r.citation, '') as citation
		FROM compliance.compliance_rule r
		LEFT JOIN compliance.compliance_ruleset_membership m ON r.id = m.rule_id
		LEFT JOIN compliance.compliance_rule_version v ON r.id = v.rule_id
		WHERE r.valid_to IS NULL AND r.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
		ORDER BY r.rule_code;
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	type ruleRec struct {
		code, pack, prov, hash, citation string
	}
	var rules []ruleRec
	packCounts := make(map[string]int)
	for rows.Next() {
		var r ruleRec
		if err := rows.Scan(&r.code, &r.pack, &r.prov, &r.hash, &r.citation); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		rules = append(rules, r)
		packCounts[r.pack]++
	}

	fmt.Println("======================================================================")
	fmt.Printf("CANONICAL COMPLIANCE RULE INVENTORY RECEIPT (LIVE ALPHA MASTER):\n")
	fmt.Printf("  Total Active Gold Copy Rules: %d\n", len(rules))
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("PACK MEMBERSHIP DISTRIBUTION:")
	for _, p := range []string{"CORE_REGULATORY", "MARKET_CONDUCT", "INSTITUTIONAL_CONTROLS", "POST_TRADE_MONITORING", "ESG_AND_SUSTAINABILITY"} {
		fmt.Printf("  %-25s: %2d rules\n", p, packCounts[p])
	}
	if packCounts["NO_PACK"] > 0 {
		fmt.Printf("  %-25s: %2d rules\n", "UNASSIGNED / NO_PACK", packCounts["NO_PACK"])
	}
	fmt.Println("======================================================================")
}
