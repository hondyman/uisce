package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check columns for 'categories' table
	query := `
		SELECT id, node_name, qualified_path, properties 
		FROM catalog_node 
		WHERE tenant_datasource_id = '25b5dce3-27d9-4773-933e-6ee29a42871f'
		AND node_name IN ('category_id', 'category_name', 'description', 'picture')
		ORDER BY node_name, qualified_path
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, path string
		var props []byte
		rows.Scan(&id, &name, &path, &props)

		var propsMap map[string]interface{}
		json.Unmarshal(props, &propsMap)

		fmt.Printf("ID: %s\nName: %s\nPath: %s\n", id, name, path)
		fmt.Printf("Stats: total_count=%v, unique_count=%v\n", propsMap["total_count"], propsMap["unique_count"])
		fmt.Println("------------------------------------------------")
	}
}
