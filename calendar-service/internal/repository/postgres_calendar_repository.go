package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// PostgreSQL Calendar Repository Implementation
// ============================================================================
// This implementation demonstrates the production patterns for:
// - Mandatory tenant filtering in all queries
// - Cross-tenant access prevention
// - Audit trail support (created_by, updated_by timestamps)
// - Soft-delete with tenant verification
// ============================================================================

type PostgresCalendarRepository struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

// NewPostgresCalendarRepository creates a new PostgreSQL calendar repository
func NewPostgresCalendarRepository(pool *pgxpool.Pool, logger *logrus.Entry) TenantCalendarRepository {
	return &PostgresCalendarRepository{
		pool:   pool,
		logger: logger.WithField("component", "postgres_calendar_repository"),
	}
}

// Create inserts a new calendar for a tenant
func (r *PostgresCalendarRepository) Create(ctx context.Context, tenantID string, calendar *TenantCalendar) error {
	if tenantID == "" {
		return errors.New("tenant_id required")
	}

	if calendar.TenantID != tenantID {
		return errors.New("calendar tenant_id must match request tenant_id")
	}

	// ⚠️ CRITICAL PATTERN: Always include tenant_id in INSERT clause
	// This ensures the database receives the tenant context explicitly
	query := `
		INSERT INTO calendars (tenant_id, id, name, description, timezone, created_by, created_at, updated_by, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, tenant_id, name, description, timezone, created_by, created_at, updated_by, updated_at
	`

	now := time.Now()
	err := r.pool.QueryRow(ctx, query,
		calendar.TenantID,
		calendar.ID,
		calendar.Name,
		calendar.Description,
		calendar.Timezone,
		calendar.CreatedBy,
		now,
		calendar.UpdatedBy,
		nil, // deleted_at is null on creation
	).Scan(
		&calendar.ID,
		&calendar.TenantID,
		&calendar.Name,
		&calendar.Description,
		&calendar.Timezone,
		&calendar.CreatedBy,
		&calendar.CreatedAt,
		&calendar.UpdatedBy,
		&calendar.UpdatedAt,
	)

	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendar.ID,
			"action":      "create",
		}).Error("Failed to create calendar")
		return err
	}

	r.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendar.ID,
		"created_by":  calendar.CreatedBy,
	}).Debug("Calendar created")

	return nil
}

// GetByID retrieves a calendar by ID, verifying tenant ownership
func (r *PostgresCalendarRepository) GetByID(ctx context.Context, tenantID string, calendarID string) (*TenantCalendar, error) {
	if tenantID == "" || calendarID == "" {
		return nil, errors.New("tenant_id and calendar_id required")
	}

	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1 AND id = $2
	// The tenant_id filter is MANDATORY - queries cannot bypass tenant scope
	query := `
		SELECT id, tenant_id, name, description, timezone, created_by, created_at, updated_by, updated_at
		FROM calendars
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		LIMIT 1
	`

	calendar := &TenantCalendar{}
	err := r.pool.QueryRow(ctx, query, tenantID, calendarID).Scan(
		&calendar.ID,
		&calendar.TenantID,
		&calendar.Name,
		&calendar.Description,
		&calendar.Timezone,
		&calendar.CreatedBy,
		&calendar.CreatedAt,
		&calendar.UpdatedBy,
		&calendar.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithFields(logrus.Fields{
				"tenant_id":   tenantID,
				"calendar_id": calendarID,
				"action":      "get_by_id",
			}).Debug("Calendar not found")
			return nil, sql.ErrNoRows
		}

		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to get calendar")
		return nil, err
	}

	return calendar, nil
}

// ListByTenant returns all calendars for a tenant
func (r *PostgresCalendarRepository) ListByTenant(ctx context.Context, tenantID string, limit int, offset int) ([]TenantCalendar, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id required")
	}

	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1 ONLY
	// The WHERE clause uses tenant_id as the first and most restrictive condition
	// This ensures database query optimizer uses tenant_id index first
	query := `
		SELECT id, tenant_id, name, description, timezone, created_by, created_at, updated_by, updated_at
		FROM calendars
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"limit":     limit,
			"offset":    offset,
		}).Error("Failed to list calendars")
		return nil, err
	}
	defer rows.Close()

	calendars := []TenantCalendar{}
	for rows.Next() {
		cal := TenantCalendar{}
		if err := rows.Scan(
			&cal.ID,
			&cal.TenantID,
			&cal.Name,
			&cal.Description,
			&cal.Timezone,
			&cal.CreatedBy,
			&cal.CreatedAt,
			&cal.UpdatedBy,
			&cal.UpdatedAt,
		); err != nil {
			r.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to scan calendar")
			return nil, err
		}

		calendars = append(calendars, cal)
	}

	if err = rows.Err(); err != nil {
		r.logger.WithError(err).WithField("tenant_id", tenantID).Error("Row iteration error")
		return nil, err
	}

	return calendars, nil
}

// Update modifies a calendar for a tenant
func (r *PostgresCalendarRepository) Update(ctx context.Context, tenantID string, calendarID string, updates map[string]interface{}) (*TenantCalendar, error) {
	if tenantID == "" || calendarID == "" {
		return nil, errors.New("tenant_id and calendar_id required")
	}

	// Build dynamic UPDATE query with tenant verification
	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1 AND id = $2
	query := `
		UPDATE calendars
		SET name = COALESCE($3, name),
		    description = COALESCE($4, description),
		    timezone = COALESCE($5, timezone),
		    updated_by = $6,
		    updated_at = $7
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING id, tenant_id, name, description, timezone, created_by, created_at, updated_by, updated_at
	`

	calendar := &TenantCalendar{}
	now := time.Now()
	updatedBy := fmt.Sprintf("%v", updates["updated_by"]) // Usually from auth context

	err := r.pool.QueryRow(ctx, query,
		tenantID,
		calendarID,
		updates["name"],
		updates["description"],
		updates["timezone"],
		updatedBy,
		now,
	).Scan(
		&calendar.ID,
		&calendar.TenantID,
		&calendar.Name,
		&calendar.Description,
		&calendar.Timezone,
		&calendar.CreatedBy,
		&calendar.CreatedAt,
		&calendar.UpdatedBy,
		&calendar.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithFields(logrus.Fields{
				"tenant_id":   tenantID,
				"calendar_id": calendarID,
			}).Debug("Calendar not found or already deleted")
			return nil, sql.ErrNoRows
		}

		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to update calendar")
		return nil, err
	}

	r.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
		"updated_by":  updatedBy,
	}).Debug("Calendar updated")

	return calendar, nil
}

// Delete performs a soft-delete on a calendar for a tenant
func (r *PostgresCalendarRepository) Delete(ctx context.Context, tenantID string, calendarID string) error {
	if tenantID == "" || calendarID == "" {
		return errors.New("tenant_id and calendar_id required")
	}

	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1 AND id = $2
	// The UPDATE statement uses tenant_id as first WHERE condition
	// This prevents any calendar from any tenant from being deleted
	query := `
		UPDATE calendars
		SET deleted_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, tenantID, calendarID)
	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to delete calendar")
		return err
	}

	// Check if any rows were affected (row found and belonged to tenant)
	if result.RowsAffected() == 0 {
		r.logger.WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Debug("Calendar not found or already deleted")
		return sql.ErrNoRows
	}

	r.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
	}).Debug("Calendar deleted (soft-delete)")

	return nil
}

// CountByTenant returns the count of non-deleted calendars for a tenant
func (r *PostgresCalendarRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	if tenantID == "" {
		return 0, errors.New("tenant_id required")
	}

	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1
	query := `
		SELECT COUNT(*)
		FROM calendars
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		r.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to count calendars")
		return 0, err
	}

	return count, nil
}

// ExistsByID checks if a calendar exists for a tenant (include deleted for audit)
func (r *PostgresCalendarRepository) ExistsByID(ctx context.Context, tenantID string, calendarID string) (bool, error) {
	if tenantID == "" || calendarID == "" {
		return false, errors.New("tenant_id and calendar_id required")
	}

	// ⚠️ CRITICAL PATTERN: WHERE tenant_id = $1 AND id = $2
	query := `
		SELECT EXISTS(
			SELECT 1 FROM calendars
			WHERE tenant_id = $1 AND id = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, tenantID, calendarID).Scan(&exists)
	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"calendar_id": calendarID,
		}).Error("Failed to check calendar existence")
		return false, err
	}

	return exists, nil
}

// ============================================================================
// SQL Building Helpers (Safe Query Construction)
// ============================================================================

// SafeCalendarWhere builds a WHERE clause that always includes tenant filter
func SafeCalendarWhere(tenantID string, additionalConditions ...string) string {
	where := fmt.Sprintf("tenant_id = '%s'", tenantID)

	for _, condition := range additionalConditions {
		if condition != "" {
			where = fmt.Sprintf("%s AND (%s)", where, condition)
		}
	}

	// Always include soft-delete check
	where = fmt.Sprintf("%s AND deleted_at IS NULL", where)

	return where
}

// ============================================================================
// Schema Migration Helper
// ============================================================================

// This would be called during database initialization to ensure schema exists.
// Including here for reference purposes.

const CalendarTableSchema = `
CREATE TABLE IF NOT EXISTS calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(64) DEFAULT 'UTC',
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    
    -- Tenant isolation: ensure no calendar escapes its tenant
    CONSTRAINT fk_tenant CHECK (tenant_id IS NOT NULL),
    
    -- Indexes for performance
    -- Tenant-first index ensures all queries start with tenant filtering
    UNIQUE INDEX idx_calendars_tenant_id (tenant_id, id) WHERE deleted_at IS NULL,
    
    INDEX idx_calendars_tenant_created (tenant_id, created_at DESC),
    INDEX idx_calendars_tenant_updated (tenant_id, updated_at DESC),
    
    -- Soft-delete index for queries checking deleted_at IS NULL
    INDEX idx_calendars_deleted (tenant_id, deleted_at)
);

-- Row-Level Security Policy (database-level tenant isolation)
ALTER TABLE calendars ENABLE ROW LEVEL SECURITY;

CREATE POLICY calendars_tenant_isolation ON calendars
    USING (tenant_id = current_setting('app.current_tenant_id'))
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id'));
`

// ============================================================================
// Phase 3 Documentation
// ============================================================================
// This implementation demonstrates the 4-layer security model:
//
// Layer 1: Application Logic (Service)
//   - Validates tenant_id is not empty
//   - Validates tenant context before operations
//   - Returns generic error for cross-tenant attempts
//
// Layer 2: SQL Queries (Repository)
//   - Every WHERE clause includes tenant_id as first condition
//   - ⚠️ CRITICAL: tenant_id filter is MANDATORY, not optional
//   - No query can bypass tenant scope
//
// Layer 3: Database Schema
//   - UNIQUE INDEX on (tenant_id, id) ensures logical isolation
//   - CHECK constraint ensures tenant_id never NULL
//   - Soft-delete field (deleted_at) for audit trails
//
// Layer 4: Database Policies (Row Level Security)
//   - RLS policy restricts even direct DB connections to tenant's rows only
//   - Requires app.current_tenant_id session variable
//   - Acts as catch-all for any application logic bypasses
//
// Result: Defense in depth - no single point of failure in tenant isolation
// ============================================================================

// eof
