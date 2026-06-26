package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Try alpha first
	connStr := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Printf("Failed to connect to alpha: %v. Trying semlayer...\n", err)
		connStr = "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable"
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal("Failed to connect to semlayer too:", err)
		}
	}
	fmt.Println("Connected to database.")

	// Read SQL file
	content, err := ioutil.ReadFile("backend/migrations/20260126_create_missing_relationship_tables.sql")
	if err != nil {
		log.Fatal("Failed to read SQL file:", err)
	}

	query := string(content)
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to execute SQL:", err)
	}

	fmt.Println("Successfully created missing relationship tables!")
}
