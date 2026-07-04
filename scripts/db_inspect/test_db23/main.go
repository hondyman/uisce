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

	customTenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"
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
					AND cnt.catalog_type_name = 'business_term'
					ORDER BY cn.node_name, (cn.tenant_id = $1::uuid) DESC
				) as subq
			`
	db.QueryRow(query, customTenantID, tenantDatasourceID).Scan(&count)
	fmt.Printf("Results with DISTINCT ON SQL query for custom tenant (business_term): %d\n", count)
}
