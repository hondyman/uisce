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
	db.QueryRow(`SELECT count(*) FROM catalog_node cn LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id WHERE cn.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND cn.tenant_datasource_id IS NOT NULL AND cnt.catalog_type_name = 'business_term'`).Scan(&count)
	fmt.Printf("Northwinds business terms with datasource: %d\n", count)

	db.QueryRow(`SELECT count(*) FROM catalog_node cn LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id WHERE cn.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND cn.tenant_datasource_id IS NULL AND cnt.catalog_type_name = 'business_term'`).Scan(&count)
	fmt.Printf("Northwinds business terms WITHOUT datasource: %d\n", count)

	db.QueryRow(`SELECT count(*) FROM catalog_node cn LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id WHERE cn.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true) AND cnt.catalog_type_name = 'business_term'`).Scan(&count)
	fmt.Printf("Gold copy business terms: %d\n", count)
}
