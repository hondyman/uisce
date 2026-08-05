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
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Try alpha first
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Printf("Failed to connect to alpha: %v. Trying semlayer...\n", err)
		// Fallback to semlayer if alpha fails
		dbURL = os.Getenv("SEMLAYER_DATABASE_URL")
		if dbURL == "" {
			log.Fatal("SEMLAYER_DATABASE_URL environment variable is required as fallback")
		}
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal("Failed to connect to semlayer too:", err)
		}
	}
	fmt.Println("Connected to database.")

	// Read SQL file
	content, err := ioutil.ReadFile("backend/migrations/20251231_align_workday_schema.sql")
	if err != nil {
		log.Fatal("Failed to read SQL file:", err)
	}

	query := string(content)
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to execute SQL:", err)
	}

	fmt.Println("Successfully applied driver table migration!")
}
