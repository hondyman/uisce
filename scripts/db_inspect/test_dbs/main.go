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

	rows, err := db.Query("SELECT datname FROM pg_database")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println(name)
	}
}
