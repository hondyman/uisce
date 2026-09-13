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
	"time"
)

// migrationsDir resolves the migrations directory relative to either the
// repo root or backend/ (mirrors the two working directories ApplyMigrations
// has historically been invoked from).
func migrationsDir() string {
	dir := "db/migrations"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = "../db/migrations"
	}
	return dir
}

// listUpFiles returns the sorted list of *.up.sql filenames (not full paths)
// in the migrations directory. Sort order is the apply order ApplyMigrations
// uses, so Status and Verify must use the same listing to describe the same
// sequence a real run would see.
func listUpFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" && len(entry.Name()) > 7 && entry.Name()[len(entry.Name())-7:] == ".up.sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func fileSHA256(dir, filename string) (string, error) {
	contentBytes, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return "", err
	}
	hashBytes := sha256.Sum256(contentBytes)
	return hex.EncodeToString(hashBytes[:]), nil
}

// AppliedMigration is one row of oms.migration_log.
type AppliedMigration struct {
	Filename  string
	SHA256    string
	AppliedAt time.Time
}

func loadAppliedMigrations(db *sql.DB) (map[string]AppliedMigration, error) {
	rows, err := db.Query(`SELECT filename, sha256, applied_at FROM oms.migration_log`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]AppliedMigration)
	for rows.Next() {
		var m AppliedMigration
		if err := rows.Scan(&m.Filename, &m.SHA256, &m.AppliedAt); err != nil {
			return nil, err
		}
		applied[m.Filename] = m
	}
	return applied, rows.Err()
}

// StatusEntry describes one migration file's state relative to oms.migration_log.
type StatusEntry struct {
	Filename  string
	Applied   bool
	AppliedAt time.Time
}

// Status reports, for every *.up.sql file on disk, whether it has been
// applied according to oms.migration_log. It does not touch the database
// beyond a read of the log table, and does not require the table to already
// exist — a missing table means every file reports as pending.
func Status(db *sql.DB) ([]StatusEntry, error) {
	dir := migrationsDir()
	files, err := listUpFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list migration files: %w", err)
	}

	applied, err := loadAppliedMigrations(db)
	if err != nil {
		// oms.migration_log may not exist yet (never applied anything) —
		// every file is simply pending rather than an error.
		applied = map[string]AppliedMigration{}
	}

	entries := make([]StatusEntry, 0, len(files))
	for _, f := range files {
		if m, ok := applied[f]; ok {
			entries = append(entries, StatusEntry{Filename: f, Applied: true, AppliedAt: m.AppliedAt})
		} else {
			entries = append(entries, StatusEntry{Filename: f})
		}
	}
	return entries, nil
}

// DriftEntry describes one migration file whose on-disk content no longer
// matches the checksum recorded when it was applied.
type DriftEntry struct {
	Filename    string
	RecordedSHA string
	CurrentSHA  string
	AppliedAt   time.Time
}

// Verify recomputes the SHA-256 of every applied migration file still
// present on disk and compares it against oms.migration_log. It reports
// every mismatch rather than the runner's own warn-and-skip behavior, which
// is silent unless someone reads the boot log. A file recorded as applied
// but missing from disk is also reported as drift (RecordedSHA set,
// CurrentSHA empty) since that's exactly the "how do we know this ever
// really ran" gap this command exists to close.
func Verify(db *sql.DB) ([]DriftEntry, error) {
	dir := migrationsDir()
	applied, err := loadAppliedMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read oms.migration_log: %w", err)
	}

	var drift []DriftEntry
	for filename, m := range applied {
		currentSHA, err := fileSHA256(dir, filename)
		if err != nil {
			if os.IsNotExist(err) {
				drift = append(drift, DriftEntry{Filename: filename, RecordedSHA: m.SHA256, AppliedAt: m.AppliedAt})
				continue
			}
			return nil, fmt.Errorf("failed to hash %s: %w", filename, err)
		}
		if currentSHA != m.SHA256 {
			drift = append(drift, DriftEntry{Filename: filename, RecordedSHA: m.SHA256, CurrentSHA: currentSHA, AppliedAt: m.AppliedAt})
		}
	}
	sort.Slice(drift, func(i, j int) bool { return drift[i].Filename < drift[j].Filename })
	return drift, nil
}

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

	dir := migrationsDir()

	files, err := listUpFiles(dir)
	if err != nil {
		log.Printf("⚠️  Migrations directory %s not found; skipping auto-migration.", dir)
		return nil
	}

	for _, filename := range files {
		fileHash, err := fileSHA256(dir, filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		contentBytes, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

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
