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
	err = db.QueryRow(`
		SELECT count(*)
		FROM catalog_node cn
		LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
		WHERE (
			(cn.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND cn.tenant_datasource_id = '25b5dce3-27d9-4773-933e-6ee29a42871f')
			OR EXISTS (SELECT 1 FROM tenants WHERE id = cn.tenant_id AND gold_copy = true)
		)`).Scan(&count)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Query result count for Northwinds: %d\n", count)
	}
}
