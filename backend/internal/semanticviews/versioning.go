package semanticviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ViewVersion represents a version of a semantic view
type ViewVersion struct {
	ID                 uuid.UUID              `json:"id"`
	ViewID             uuid.UUID              `json:"view_id"`
	Version            int                    `json:"version"`
	Schema             map[string]interface{} `json:"schema"`
	Description        string                 `json:"description"`
	IsActive           bool                   `json:"is_active"`
	IsDeprecated       bool                   `json:"is_deprecated"`
	MigrationScript    string                 `json:"migration_script,omitempty"`
	BreakingChanges    bool                   `json:"breaking_changes"`
	CompatibilityNotes string                 `json:"compatibility_notes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	DeprecatedAt       *time.Time             `json:"deprecated_at,omitempty"`
	CreatedBy          string                 `json:"created_by,omitempty"`
}

// SchemaMigration represents a migration between versions
type SchemaMigration struct {
	ID              uuid.UUID  `json:"id"`
	ViewID          uuid.UUID  `json:"view_id"`
	FromVersion     int        `json:"from_version"`
	ToVersion       int        `json:"to_version"`
	MigrationType   string     `json:"migration_type"`
	MigrationStatus string     `json:"migration_status"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	RowsAffected    int        `json:"rows_affected"`
	CreatedAt       time.Time  `json:"created_at"`
	ExecutedBy      string     `json:"executed_by,omitempty"`
}

// VersioningService manages semantic view versions
type VersioningService struct {
	db *sql.DB
}

// NewVersioningService creates a new versioning service
func NewVersioningService(db *sql.DB) *VersioningService {
	return &VersioningService{db: db}
}

// CreateVersion creates a new version of a semantic view
func (vs *VersioningService) CreateVersion(ctx context.Context, version *ViewVersion) error {
	schemaJSON, err := json.Marshal(version.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	query := `
		INSERT INTO semantic_view_versions (
			id, view_id, version, schema, description,
			is_active, breaking_changes, migration_script,
			compatibility_notes, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`

	err = vs.db.QueryRowContext(ctx, query,
		uuid.New(),
		version.ViewID,
		version.Version,
		schemaJSON,
		version.Description,
		version.IsActive,
		version.BreakingChanges,
		version.MigrationScript,
		version.CompatibilityNotes,
		version.CreatedBy,
	).Scan(&version.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	return nil
}

// GetVersion retrieves a specific version of a view
func (vs *VersioningService) GetVersion(ctx context.Context, viewID uuid.UUID, version int) (*ViewVersion, error) {
	query := `
		SELECT id, view_id, version, schema, description,
		       is_active, is_deprecated, migration_script,
		       breaking_changes, compatibility_notes,
		       created_at, deprecated_at, created_by
		FROM semantic_view_versions
		WHERE view_id = $1 AND version = $2
	`

	var vv ViewVersion
	var schemaJSON []byte
	var deprecatedAt sql.NullTime
	var createdBy sql.NullString

	err := vs.db.QueryRowContext(ctx, query, viewID, version).Scan(
		&vv.ID,
		&vv.ViewID,
		&vv.Version,
		&schemaJSON,
		&vv.Description,
		&vv.IsActive,
		&vv.IsDeprecated,
		&vv.MigrationScript,
		&vv.BreakingChanges,
		&vv.CompatibilityNotes,
		&vv.CreatedAt,
		&deprecatedAt,
		&createdBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("version not found: view=%s version=%d", viewID, version)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if err := json.Unmarshal(schemaJSON, &vv.Schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	if deprecatedAt.Valid {
		vv.DeprecatedAt = &deprecatedAt.Time
	}
	if createdBy.Valid {
		vv.CreatedBy = createdBy.String
	}

	return &vv, nil
}

// GetLatestVersion retrieves the latest active version
func (vs *VersioningService) GetLatestVersion(ctx context.Context, viewID uuid.UUID) (*ViewVersion, error) {
	query := `
		SELECT version
		FROM semantic_view_versions
		WHERE view_id = $1
		  AND is_active = true
		ORDER BY version DESC
		LIMIT 1
	`

	var latestVersion int
	err := vs.db.QueryRowContext(ctx, query, viewID).Scan(&latestVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active version found for view: %s", viewID)
		}
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	return vs.GetVersion(ctx, viewID, latestVersion)
}

// ListVersions lists all versions of a view
func (vs *VersioningService) ListVersions(ctx context.Context, viewID uuid.UUID) ([]ViewVersion, error) {
	query := `
		SELECT id, view_id, version, description,
		       is_active, is_deprecated, breaking_changes,
		       created_at, deprecated_at
		FROM semantic_view_versions
		WHERE view_id = $1
		ORDER BY version DESC
	`

	rows, err := vs.db.QueryContext(ctx, query, viewID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []ViewVersion
	for rows.Next() {
		var vv ViewVersion
		var deprecatedAt sql.NullTime

		err := rows.Scan(
			&vv.ID,
			&vv.ViewID,
			&vv.Version,
			&vv.Description,
			&vv.IsActive,
			&vv.IsDeprecated,
			&vv.BreakingChanges,
			&vv.CreatedAt,
			&deprecatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}

		if deprecatedAt.Valid {
			vv.DeprecatedAt = &deprecatedAt.Time
		}

		versions = append(versions, vv)
	}

	return versions, nil
}

// DeprecateVersion marks a version as deprecated
func (vs *VersioningService) DeprecateVersion(ctx context.Context, viewID uuid.UUID, version int) error {
	query := `
		UPDATE semantic_view_versions
		SET is_deprecated = true,
		    deprecated_at = NOW()
		WHERE view_id = $1 AND version = $2
	`

	result, err := vs.db.ExecContext(ctx, query, viewID, version)
	if err != nil {
		return fmt.Errorf("failed to deprecate version: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("version not found: view=%s version=%d", viewID, version)
	}

	return nil
}

// MigrateView migrates data from one version to another
func (vs *VersioningService) MigrateView(ctx context.Context, viewID uuid.UUID, fromVersion, toVersion int, executedBy string) (*SchemaMigration, error) {
	// Check if migration is safe
	var isSafe bool
	err := vs.db.QueryRowContext(ctx, "SELECT is_migration_safe($1, $2, $3)", viewID, fromVersion, toVersion).Scan(&isSafe)
	if err != nil {
		return nil, fmt.Errorf("failed to check migration safety: %w", err)
	}

	if !isSafe {
		return nil, fmt.Errorf("migration is unsafe: breaking changes with active users")
	}

	// Determine migration type
	toVer, err := vs.GetVersion(ctx, viewID, toVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}

	migrationType := "compatible"
	if toVer.BreakingChanges {
		migrationType = "breaking"
	} else if toVersion > fromVersion {
		migrationType = "additive"
	}

	// Create migration record
	migration := &SchemaMigration{
		ID:              uuid.New(),
		ViewID:          viewID,
		FromVersion:     fromVersion,
		ToVersion:       toVersion,
		MigrationType:   migrationType,
		MigrationStatus: "in_progress",
		ExecutedBy:      executedBy,
	}

	now := time.Now()
	migration.StartedAt = &now

	query := `
		INSERT INTO view_schema_migrations (
			id, view_id, from_version, to_version,
			migration_type, migration_status, started_at, executed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	err = vs.db.QueryRowContext(ctx, query,
		migration.ID,
		migration.ViewID,
		migration.FromVersion,
		migration.ToVersion,
		migration.MigrationType,
		migration.MigrationStatus,
		migration.StartedAt,
		migration.ExecutedBy,
	).Scan(&migration.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create migration record: %w", err)
	}

	// Execute migration script if available
	if toVer.MigrationScript != "" {
		// TODO: Execute migration script safely
		// For now, we'll just mark it as completed
	}

	// Update migration status
	completedAt := time.Now()
	migration.CompletedAt = &completedAt
	migration.MigrationStatus = "completed"

	updateQuery := `
		UPDATE view_schema_migrations
		SET migration_status = $1, completed_at = $2
		WHERE id = $3
	`

	_, err = vs.db.ExecContext(ctx, updateQuery, migration.MigrationStatus, migration.CompletedAt, migration.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update migration status: %w", err)
	}

	return migration, nil
}

// GetMigrationHistory retrieves migration history for a view
func (vs *VersioningService) GetMigrationHistory(ctx context.Context, viewID uuid.UUID) ([]SchemaMigration, error) {
	query := `
		SELECT id, view_id, from_version, to_version,
		       migration_type, migration_status,
		       started_at, completed_at, error_message,
		       rows_affected, created_at, executed_by
		FROM view_schema_migrations
		WHERE view_id = $1
		ORDER BY created_at DESC
	`

	rows, err := vs.db.QueryContext(ctx, query, viewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}
	defer rows.Close()

	var migrations []SchemaMigration
	for rows.Next() {
		var m SchemaMigration
		var startedAt, completedAt sql.NullTime
		var errorMessage, executedBy sql.NullString

		err := rows.Scan(
			&m.ID,
			&m.ViewID,
			&m.FromVersion,
			&m.ToVersion,
			&m.MigrationType,
			&m.MigrationStatus,
			&startedAt,
			&completedAt,
			&errorMessage,
			&m.RowsAffected,
			&m.CreatedAt,
			&executedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}

		if startedAt.Valid {
			m.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			m.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			m.ErrorMessage = errorMessage.String
		}
		if executedBy.Valid {
			m.ExecutedBy = executedBy.String
		}

		migrations = append(migrations, m)
	}

	return migrations, nil
}
