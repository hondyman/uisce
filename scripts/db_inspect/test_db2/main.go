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
	db.QueryRow("SELECT count(*) FROM entity_attribute WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'").Scan(&count)
	fmt.Printf("EntityAttributes for Northwinds: %d\n", count)

	db.QueryRow("SELECT count(*) FROM entity_attribute WHERE is_core = true").Scan(&count)
	fmt.Printf("Core EntityAttributes: %d\n", count)
}
