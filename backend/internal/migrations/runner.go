package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func ApplyMigrations(db *sql.DB) error {
	// Hold a single dedicated connection for the entire run so that
	// search_path (set once per connection) stays pinned throughout.
	// Go's database/sql pools connections; without this, later migrations
	// in the loop may run on pooled connections with the DB-level
	// search_path ('vend, public') still active, causing unqualified
	// CREATE TABLE to land in vend instead of public.
	conn, err := db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get dedicated connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(context.Background(), `SET search_path TO public, oms`); err != nil {
		return fmt.Errorf("failed to pin search_path: %w", err)
	}

	_, err = db.Exec(`CREATE SCHEMA IF NOT EXISTS oms`)
	if err != nil {
		return fmt.Errorf("failed to create oms schema: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS oms.migration_log (
			filename TEXT PRIMARY KEY,
			sha256   TEXT NOT NULL,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migration_log table: %w", err)
	}

	migrationsDir := "db/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "../db/migrations"
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Printf("⚠️  Migrations directory %s not found; skipping auto-migration.", migrationsDir)
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" && len(entry.Name()) > 7 && entry.Name()[len(entry.Name())-7:] == ".up.sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		fullPath := filepath.Join(migrationsDir, filename)
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		hashBytes := sha256.Sum256(contentBytes)
		fileHash := hex.EncodeToString(hashBytes[:])

		var existingHash string
		err = db.QueryRow(`SELECT sha256 FROM oms.migration_log WHERE filename = $1`, filename).Scan(&existingHash)
		if err == nil {
			if existingHash != fileHash {
				log.Printf("⚠️  WARNING: migration file %s content has changed since it was applied; skipping", filename)
			}
			continue
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("failed checking migration log for %s: %w", filename, err)
		}

		content := string(contentBytes)

		if hasTransactionControl(content) {
			return fmt.Errorf("migration %s contains transaction-control statements (BEGIN/COMMIT/ROLLBACK); remove them before running via the runner", filename)
		}

		content = stripTransactionStatements(content)

		tx, err := conn.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("failed to begin tx for migration %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(context.Background(), content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed executing migration %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(context.Background(), `INSERT INTO oms.migration_log (filename, sha256) VALUES ($1, $2)`, filename, fileHash); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed logging migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed committing migration %s: %w", filename, err)
		}

		log.Printf("✅  Applied migration: %s", filename)
	}

	return nil
}

func stripTransactionStatements(content string) string {
	content = regexp.MustCompile(`(?i)^\s*BEGIN\s*;?\s*$`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)^\s*COMMIT\s*;?\s*$`).ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func hasTransactionControl(content string) bool {
	content = regexp.MustCompile(`(?i)--.*$`).ReplaceAllString(content, "")          // strip single-line comments
	content = regexp.MustCompile(`(?i)'[^']*'`).ReplaceAllString(content, "")        // strip single-quoted string literals
	content = regexp.MustCompile(`(?i)"[^"]*"`).ReplaceAllString(content, "")        // strip double-quoted identifiers
	matched, _ := regexp.MatchString(`(?i)\b(COMMIT|ROLLBACK)\b`, content)
	return matched
}
