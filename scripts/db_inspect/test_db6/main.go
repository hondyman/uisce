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
	db, err := sql.Open("postgres", "postgres://postgres:postgres@100.84.50.65:5432/uisce?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT count(*) FROM entity_attribute").Scan(&count)
	fmt.Printf("Total entity_attributes: %d\n", count)
	
	db.QueryRow("SELECT count(*) FROM semantic_terms").Scan(&count)
	fmt.Printf("Total semantic_terms: %d\n", count)
}
