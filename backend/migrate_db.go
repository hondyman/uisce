package backend

import (
	"bufio"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func MigrateDB() error {
	// Connect to database
	db, err := GetAppDBConnection("alpha")
	if err != nil {
		return err
	}
	defer db.Close()

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
		return err
	}

	log.Printf("Migration completed! Successfully executed %d statements.", executedCount)
	return nil
}
