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

	// What happens when $2 is "" ?
	var count int
	query := `
				SELECT count(*) FROM catalog_node cn
				WHERE (
					(cn.tenant_id = $1::uuid AND (cn.tenant_datasource_id = $2::uuid OR cn.tenant_datasource_id IS NULL))
				)
			`
	err = db.QueryRow(query, "99e99e99-99e9-49e9-89e9-99e99e99e999", "").Scan(&count)
	if err != nil {
		fmt.Printf("DB Error: %v\n", err)
	} else {
		fmt.Printf("Count: %d\n", count)
	}
}
