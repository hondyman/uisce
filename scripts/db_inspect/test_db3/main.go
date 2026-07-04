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
	db, err := sql.Open("postgres", "postgres://postgres:postgres@100.84.50.65:5432/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT count(*) FROM catalog_node").Scan(&count)
	fmt.Printf("Total catalog_nodes: %d\n", count)

	var name string
	err = db.QueryRow("SELECT node_name FROM catalog_node LIMIT 1").Scan(&name)
	if err == nil {
		fmt.Printf("Example node: %s\n", name)
	}
}
