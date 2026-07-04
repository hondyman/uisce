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
	err = db.QueryRow("SELECT count(*) FROM catalog_node").Scan(&count)
	if err != nil {
		fmt.Printf("catalog_node err: %v\n", err)
	} else {
		fmt.Printf("Total catalog_nodes in alpha: %d\n", count)
	}
}
