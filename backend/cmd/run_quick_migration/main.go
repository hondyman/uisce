package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	// Using the connection string found in cmd/migrate-db/main.go
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	migrationPath := "migrations/20251231_align_workday_schema.sql"
	log.Printf("Reading migration file: %s", migrationPath)

	file, err := os.Open(migrationPath)
	if err != nil {
		log.Fatal("Failed to open migration file:", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var statement strings.Builder
	executedCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "--") || line == "" {
			continue
		}

		statement.WriteString(line)
		statement.WriteString(" ")

		// If line ends with semicolon, execute the statement
		if strings.HasSuffix(line, ";") {
			sqlCmd := strings.TrimSpace(statement.String())

			// Skip empty statements
			if sqlCmd == ";" {
				statement.Reset()
				continue
			}

			log.Printf("Executing: %s", sqlCmd[:min(100, len(sqlCmd))])

			_, err := db.Exec(sqlCmd)
			if err != nil {
				// Don't error out on "already exists" but log it
				log.Printf("Warning: Failed to execute statement: %v", err)
			} else {
				executedCount++
			}

			statement.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading migration file:", err)
	}

	log.Printf("Migration completed! Executed %d statements.", executedCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
