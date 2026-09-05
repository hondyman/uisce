package migrations

import (
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

// Package migrations owns the database schema versioning machinery for the
// Uisce platform. The authoritative bookkeeping lives in `oms.migration_log`:
//
//   - Each row records a single applied migration file (the .up.sql form).
//   - The `sha256` column stores the SHA-256 of the file content at apply time.
//   - On every ApplyMigrations invocation, the runner re-computes the SHA of
//     each .up.sql in `db/migrations/` and skips files whose SHA matches the
//     stored value. If the SHA has changed since apply, the file is
//     **skipped with a warning** — this runner does not re-execute changed
//     migrations. (Editing an applied file is treated as a defect, not as an
//     opportunity to replay history.)
//   - public.schema_migrations (golang-migrate's table) is NOT authoritative
//     for this database and may be empty or stale. Do not use it for
//     migration-status queries — always query `oms.migration_log`.
func ApplyMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS oms`)
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
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			// 3rd fallback: MIGRATIONS_DIR env var (absolute path, set by deploy)
			migrationsDir = os.Getenv("MIGRATIONS_DIR")
			if migrationsDir == "" {
				log.Printf("⚠️  Migrations directory not found (tried db/migrations, ../db/migrations, MIGRATIONS_DIR); skipping auto-migration.")
				return nil
			}
		}
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Printf("⚠️  Migrations directory %s unreadable; skipping auto-migration: %v", migrationsDir, err)
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

		content = stripTransactionStatements(content)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin tx for migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed executing migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(`INSERT INTO oms.migration_log (filename, sha256) VALUES ($1, $2)`, filename, fileHash); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed logging migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			if isUnexpectedTxStatusIdle(err) {
				log.Printf("⚠️  Migration %s committed internally (contains BEGIN/COMMIT), recording as applied", filename)
				if _, logErr := db.Exec(`INSERT INTO oms.migration_log (filename, sha256) VALUES ($1, $2)`, filename, fileHash); logErr != nil {
					log.Printf("⚠️  Failed to record migration %s: %v", filename, logErr)
				}
				continue
			}
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

func isUnexpectedTxStatusIdle(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unexpected transaction status")
}
