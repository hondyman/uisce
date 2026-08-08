// +build ignore

// UTC Compliance Migration Checker
// Validates that migrations do not introduce timestamp columns without timezone
// Run: go run scripts/check_migration_utc.go migrations/*.sql
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check_migration_utc.go <migration_files...>")
		fmt.Println("Example: go run check_migration_utc.go migrations/20260101_*.sql")
		os.Exit(1)
	}

	violations := 0

	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ Error reading %s: %v\n", path, err)
			violations++
			continue
		}

		content := string(data)

		// Skip down migrations
		if strings.Contains(content, "-- +migrate Down") {
			continue
		}

		// Patterns that indicate bare timestamp columns (not timestamptz)
		bareTimestampRegex := regexp.MustCompile(`(?i)^\s*[a-z_]+\s+timestamp\s+(NOT\s+NULL|NULL|DEFAULT|,\s*\w|\))`)

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			// Skip comments
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}

			if bareTimestampRegex.MatchString(line) {
				// Check it's not timestamptz
				if strings.Contains(strings.ToLower(line), "timestamptz") {
					continue
				}
				fmt.Printf("❌ %s:%d - Bare TIMESTAMP column (use TIMESTAMPTZ):\n    %s\n", path, i+1, strings.TrimSpace(line))
				violations++
			}
		}

		// Also check for ALTER TABLE ... TYPE TIMESTAMP (not timestamptz)
		alterRegex := regexp.MustCompile(`(?i)ALTER\s+TABLE.*ALTER\s+COLUMN.*TYPE\s+TIMESTAMP[^a-z]`)
		for i, line := range lines {
			lower := strings.ToLower(line)
			if alterRegex.MatchString(line) && !strings.Contains(lower, "timestamptz") {
				fmt.Printf("❌ %s:%d - Converting to bare TIMESTAMP (use TIMESTAMPTZ):\n    %s\n", path, i+1, strings.TrimSpace(line))
				violations++
			}
		}
	}

	fmt.Println()
	if violations > 0 {
		fmt.Printf("❌ Found %d UTC compliance violation(s)\n", violations)
		os.Exit(1)
	}
	fmt.Println("✅ All migrations are UTC compliant")
}
