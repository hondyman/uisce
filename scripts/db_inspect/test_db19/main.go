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

	rows, err := db.Query("SELECT id, name, gold_copy FROM tenants")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		var goldCopy bool
		rows.Scan(&id, &name, &goldCopy)
		fmt.Printf("id: %s, name: %s, gold_copy: %v\n", id, name, goldCopy)
	}
}
