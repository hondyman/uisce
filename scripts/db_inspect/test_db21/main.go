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
	db.QueryRow(`SELECT count(*) FROM catalog_node cn LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id WHERE cn.tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6' AND cnt.catalog_type_name = 'business_term'`).Scan(&count)
	fmt.Printf("Northwinds custom specific terms: %d\n", count)
}
