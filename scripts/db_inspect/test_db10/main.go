//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT count(*) FROM catalog_node WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'").Scan(&count)
	fmt.Printf("Nodes for Northwinds: %d\n", count)

	db.QueryRow("SELECT count(*) FROM catalog_node WHERE tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)").Scan(&count)
	fmt.Printf("Nodes for Gold Copy: %d\n", count)
}
