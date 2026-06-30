package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a single database migration
type Migration struct {
	ID        string
	Name      string
	UpSQL     string
	DownSQL   string
	CreatedAt time.Time
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db           *sql.DB
	tableName    string
	migrationDir string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB, migrationDir string) *MigrationRunner {
	return &MigrationRunner{
		db:           db,
		tableName:    "schema_migrations",
		migrationDir: migrationDir,
	}
}

// Init creates the migrations table if it doesn't exist
func (mr *MigrationRunner) Init() error {
	_, err := mr.db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			checksum VARCHAR(64)
		)
	`, mr.tableName))
	return err
}

// LoadMigrations loads all migration files from the migration directory
func (mr *MigrationRunner) LoadMigrations() ([]Migration, error) {
	files, err := filepath.Glob(filepath.Join(mr.migrationDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to list migration files: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		// Skip down migration files, they are handled by their up counterpart
		if strings.HasSuffix(file, ".down.sql") {
			continue
		}
		migration, err := mr.parseMigrationFile(file)
		if err != nil {
			log.Printf("Warning: failed to parse migration file %s: %v", file, err)
			continue
		}
		migrations = append(migrations, migration)
	}

	// Sort migrations by leading numeric prefix when possible, otherwise lexicographically
	sort.Slice(migrations, func(i, j int) bool {
		leadingNum := func(s string) (int, bool) {
			var buf strings.Builder
			for _, r := range s {
				if r >= '0' && r <= '9' {
					buf.WriteRune(r)
				} else {
					break
				}
			}
			if buf.Len() == 0 {
				return 0, false
			}
			n, err := strconv.Atoi(buf.String())
			if err != nil {
				return 0, false
			}
			return n, true
		}

		iNum, iOK := leadingNum(migrations[i].ID)
		jNum, jOK := leadingNum(migrations[j].ID)

		if iOK && jOK {
			if iNum != jNum {
				return iNum < jNum
			}
			// same numeric prefix, fall back to full string compare
			return migrations[i].ID < migrations[j].ID
		}

		if iOK && !jOK {
			// numeric-prefixed IDs come before non-numeric-prefixed
			return true
		}
		if !iOK && jOK {
			return false
		}

		// Fallback to string comparison
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

// parseMigrationFile parses a single migration file
func (mr *MigrationRunner) parseMigrationFile(filePath string) (Migration, error) {
	filename := filepath.Base(filePath)
	
	var id, name, upSQL, downSQL string
	
	if strings.HasSuffix(filename, ".up.sql") {
		id = strings.TrimSuffix(filename, ".up.sql")
		name = id
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return Migration{}, fmt.Errorf("failed to read up migration file: %w", err)
		}
		contentStr := string(content)
		if parts := strings.Split(contentStr, "-- +migrate Down"); len(parts) == 2 {
			upSQL = strings.TrimSpace(parts[0])
			downSQL = strings.TrimSpace(parts[1])
		} else if parts := strings.Split(contentStr, "-- +goose Down"); len(parts) == 2 {
			upSQL = strings.TrimSpace(parts[0])
			downSQL = strings.TrimSpace(parts[1])
		} else {
			upSQL = contentStr
			// Look for corresponding .down.sql file
			downPath := strings.TrimSuffix(filePath, ".up.sql") + ".down.sql"
			if _, err := os.Stat(downPath); err == nil {
				downContent, err := os.ReadFile(downPath)
				if err == nil {
					downSQL = string(downContent)
				}
			}
		}
	} else {
		id = strings.TrimSuffix(filename, ".sql")
		name = id
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return Migration{}, fmt.Errorf("failed to read migration file: %w", err)
		}
		contentStr := string(content)
		
		// Try "-- +migrate Down" first
		if parts := strings.Split(contentStr, "-- +migrate Down"); len(parts) == 2 {
			upSQL = strings.TrimSpace(parts[0])
			downSQL = strings.TrimSpace(parts[1])
		} else if parts := strings.Split(contentStr, "-- +goose Down"); len(parts) == 2 {
			upSQL = strings.TrimSpace(parts[0])
			downSQL = strings.TrimSpace(parts[1])
		} else {
			upSQL = contentStr
			downSQL = ""
		}
	}

	return Migration{
		ID:        id,
		Name:      name,
		UpSQL:     upSQL,
		DownSQL:   downSQL,
		CreatedAt: time.Now(),
	}, nil
}

// GetAppliedMigrations returns all applied migration versions
func (mr *MigrationRunner) GetAppliedMigrations() (map[string]bool, error) {
	rows, err := mr.db.Query(fmt.Sprintf("SELECT version FROM %s", mr.tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, nil
}

// ApplyMigration applies a single migration
func (mr *MigrationRunner) ApplyMigration(migration Migration) error {
	log.Printf("Applying migration: %s - %s", migration.ID, migration.Name)

	// Start transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	checksum := mr.calculateChecksum(migration.UpSQL)
	_, err = tx.Exec(fmt.Sprintf(`
		INSERT INTO %s (version, name, checksum)
		VALUES ($1, $2, $3)
	`, mr.tableName), migration.ID, migration.Name, checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	log.Printf("Successfully applied migration: %s", migration.ID)
	return nil
}

// RollbackMigration rolls back a single migration
func (mr *MigrationRunner) RollbackMigration(migration Migration) error {
	if migration.DownSQL == "" {
		return fmt.Errorf("no down migration available for: %s", migration.ID)
	}

	log.Printf("Rolling back migration: %s - %s", migration.ID, migration.Name)

	// Start transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if _, err := tx.Exec(migration.DownSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration record
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE version = $1", mr.tableName), migration.ID)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	log.Printf("Successfully rolled back migration: %s", migration.ID)
	return nil
}

// Up applies all pending migrations
func (mr *MigrationRunner) Up() error {
	if err := mr.Init(); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	migrations, err := mr.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, migration := range migrations {
		if applied[migration.ID] {
			log.Printf("Migration already applied: %s", migration.ID)
			continue
		}

		if err := mr.ApplyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	log.Println("All migrations applied successfully")
	return nil
}

// Down rolls back the last applied migration
func (mr *MigrationRunner) Down() error {
	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		log.Println("No migrations to rollback")
		return nil
	}

	migrations, err := mr.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Find the last applied migration
	var lastMigration *Migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if applied[migrations[i].ID] {
			lastMigration = &migrations[i]
			break
		}
	}

	if lastMigration == nil {
		log.Println("No applied migrations found to rollback")
		return nil
	}

	return mr.RollbackMigration(*lastMigration)
}

// Status shows the current migration status
func (mr *MigrationRunner) Status() error {
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	log.Println("Migration Status:")
	log.Println("=================")

	for _, migration := range migrations {
		status := "Pending"
		if applied[migration.ID] {
			status = "Applied"
		}
		log.Printf("%s: %s - %s", migration.ID, migration.Name, status)
	}

	return nil
}

// calculateChecksum calculates a simple checksum for migration content
func (mr *MigrationRunner) calculateChecksum(content string) string {
	// Simple checksum implementation - in production, use a proper hash
	sum := 0
	for _, char := range content {
		sum += int(char)
	}
	return fmt.Sprintf("%d", sum%1000000)
}
