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

	tenantID := "99e99e99-99e9-49e9-89e9-99e99e99e999"
	tenantDatasourceID := "25b5dce3-27d9-4773-933e-6ee29a42871f"

	var count int
	query := `
				SELECT count(*) FROM (
					SELECT DISTINCT ON (cn.node_name) cn.id
					FROM catalog_node cn
					LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
					WHERE (
						(cn.tenant_id = $1::uuid AND (cn.tenant_datasource_id = $2::uuid OR cn.tenant_datasource_id IS NULL))
						OR EXISTS (SELECT 1 FROM tenants WHERE id = cn.tenant_id AND gold_copy = true)
					)
					ORDER BY cn.node_name, (cn.tenant_id = $1::uuid) DESC
				) as subq
			`
	db.QueryRow(query, tenantID, tenantDatasourceID).Scan(&count)
	fmt.Printf("Total nodes across entire tenant: %d\n", count)
}
