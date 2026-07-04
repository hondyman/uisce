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

	query := `
				SELECT cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties, COALESCE(cnt.catalog_type_name, 'table') as catalog_type
				FROM catalog_node cn
				LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
				WHERE (
					(cn.tenant_id = $1::uuid AND cn.tenant_datasource_id = $2::uuid)
					OR EXISTS (SELECT 1 FROM tenants WHERE id = cn.tenant_id AND gold_copy = true)
				)
				AND cnt.catalog_type_name = 'business_term'
				ORDER BY cn.node_name LIMIT 1
			`
	var id, nodeName, description, tID, catalogTypeName string
	var dsID sql.NullString
	var createdAt, updatedAt string
	var propsJSON []byte

	err = db.QueryRow(query, tenantID, tenantDatasourceID).Scan(&id, &nodeName, &description, &tID, &dsID, &createdAt, &updatedAt, &propsJSON, &catalogTypeName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("id: %s\n", id)
	fmt.Printf("node_name: %s\n", nodeName)
	fmt.Printf("catalog_type: %s\n", catalogTypeName)
	fmt.Printf("tenant_datasource_id: %s\n", dsID.String)
}
