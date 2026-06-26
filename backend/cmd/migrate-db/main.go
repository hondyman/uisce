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
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Read migration file
	file, err := os.Open("migrations/000003_database_optimizations.sql")
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
			sql := strings.TrimSpace(statement.String())

			// Skip empty statements
			if sql == ";" {
				statement.Reset()
				continue
			}

			log.Printf("Executing: %s", sql[:min(100, len(sql))])

			_, err := db.Exec(sql)
			if err != nil {
				log.Printf("Warning: Failed to execute statement: %v", err)
				log.Printf("Statement was: %s", sql)
				// Continue with next statement instead of failing
			} else {
				executedCount++
			}

			statement.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading migration file:", err)
	}

	log.Printf("Migration completed! Successfully executed %d statements.", executedCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
