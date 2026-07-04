//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"time"
	"encoding/json"
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
				SELECT cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties, COALESCE(cnt.catalog_type_name, 'table') as catalog_type
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

	nodes := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, nodeName, description, tID, catalogTypeName string
		var dsID sql.NullString
		var createdAt, updatedAt time.Time
		var propsJSON []byte

		err := rows.Scan(&id, &nodeName, &description, &tID, &dsID, &createdAt, &updatedAt, &propsJSON, &catalogTypeName)
		if err != nil {
			log.Printf("Scan error: %v\n", err)
			continue
		}

		var props map[string]interface{}
		if err := json.Unmarshal(propsJSON, &props); err != nil {
			props = make(map[string]interface{})
		}

		node := map[string]interface{}{
			"id":                   id,
			"node_id":              id,
			"node_name":            nodeName,
			"qualified_path":       nodeName,
			"catalog_type":         catalogTypeName,
			"node_type":            catalogTypeName,
			"tenant_id":            tID,
			"tenant_datasource_id": dsID.String,
			"created_at":           createdAt,
			"updated_at":           updatedAt,
			"description":          description,
			"properties":           props,
		}

		nodes = append(nodes, node)
	}

	out, _ := json.MarshalIndent(nodes, "", "  ")
	fmt.Printf("Length: %d\n", len(nodes))
	if len(nodes) > 0 {
		fmt.Printf("First node JSON: %s\n", string(out[:500]))
	}
}
