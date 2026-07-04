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

	var count1, count2 int
	db.QueryRow("SELECT count(*) FROM catalog_node_type").Scan(&count1)
	db.QueryRow("SELECT count(*) FROM catalog_node_types").Scan(&count2)
	
	fmt.Printf("catalog_node_type: %d\n", count1)
	fmt.Printf("catalog_node_types: %d\n", count2)
}
