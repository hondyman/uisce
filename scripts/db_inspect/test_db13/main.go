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
	nodeType := "business_term"

	query := `
				SELECT cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties
				FROM catalog_node cn
				LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
				WHERE (
					(cn.tenant_id = $1::uuid AND cn.tenant_datasource_id = $2::uuid)
					OR EXISTS (SELECT 1 FROM tenants WHERE id = cn.tenant_id AND gold_copy = true)
				)
			`
	args := []interface{}{tenantID, tenantDatasourceID}
	argIndex := 3

	if nodeType != "" {
		query += fmt.Sprintf(" AND cnt.catalog_type_name = $%d", argIndex)
		args = append(args, nodeType)
		argIndex++
	}

	query += " ORDER BY cn.node_name LIMIT 50"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	fmt.Printf("Fetched %d nodes\n", count)
}
