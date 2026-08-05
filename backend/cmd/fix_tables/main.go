package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "alpha"
	}
	connStr := fmt.Sprintf("postgres://postgres@localhost:5432/%s?sslmode=disable", dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", dbName, err)
		os.Exit(1)
	}
	fmt.Printf("Connected to %s.\n", dbName)

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
