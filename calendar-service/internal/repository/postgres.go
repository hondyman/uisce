package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// PostgresRepositories holds all repository implementations
type PostgresRepositories struct {
	Tenant   TenantRepository
	Profile  CalendarProfileRepository
	Holiday  HolidayRepository
	Blackout BlackoutWindowRepository
	Metadata MetadataRepository
	AuditLog AuditLogRepository
	pool     *pgxpool.Pool
	logger   *logrus.Entry
}

// NewPostgresRepositories creates new repository instances
func NewPostgresRepositories(pool *pgxpool.Pool, logger *logrus.Entry) *PostgresRepositories {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	repos := &PostgresRepositories{
		pool:   pool,
		logger: logger.WithField("component", "repository"),
	}

	// Initialize individual repositories
	repos.Tenant = &tenantRepo{pool: pool, logger: repos.logger}
	repos.Profile = &profileRepo{pool: pool, logger: repos.logger}
	repos.Holiday = &holidayRepo{pool: pool, logger: repos.logger}
	repos.Blackout = &blackoutRepo{pool: pool, logger: repos.logger}
	repos.Metadata = &metadataRepo{pool: pool, logger: repos.logger}
	repos.AuditLog = &auditLogRepo{pool: pool, logger: repos.logger}

	return repos
}

// ============================================================================
// Tenant Repository Implementation
// ============================================================================

type tenantRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *tenantRepo) Create(ctx context.Context, tenant *Tenant) error {
	query := `
		INSERT INTO tenants (id, name, region, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	metadata, _ := json.Marshal(tenant.Metadata)
	err := r.pool.QueryRow(ctx, query,
		uuid.New(),
		tenant.Name,
		tenant.Region,
		metadata,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create tenant")
	}
	return err
}

func (r *tenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	query := `SELECT id, name, region, created_at, updated_at, metadata FROM tenants WHERE id = $1 AND deleted_at IS NULL`
	tenant := &Tenant{}
	var metadata json.RawMessage

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Region, &tenant.CreatedAt, &tenant.UpdatedAt, &metadata,
	)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get tenant")
		return nil, err
	}

	_ = json.Unmarshal(metadata, &tenant.Metadata)
	return tenant, nil
}

func (r *tenantRepo) GetByName(ctx context.Context, name string) (*Tenant, error) {
	query := `SELECT id, name, region, created_at, updated_at FROM tenants WHERE name = $1 AND deleted_at IS NULL`
	tenant := &Tenant{}
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&tenant.ID, &tenant.Name, &tenant.Region, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	return tenant, err
}

func (r *tenantRepo) List(ctx context.Context, limit, offset int) ([]*Tenant, error) {
	query := `
		SELECT id, name, region, created_at, updated_at 
		FROM tenants 
		WHERE deleted_at IS NULL 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	tenants := []*Tenant{}
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tenant := &Tenant{}
		if err := rows.Scan(&tenant.ID, &tenant.Name, &tenant.Region, &tenant.CreatedAt, &tenant.UpdatedAt); err != nil {
			r.logger.WithError(err).Error("Failed to scan tenant")
			continue
		}
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func (r *tenantRepo) Update(ctx context.Context, tenant *Tenant) error {
	query := `UPDATE tenants SET name = $1, region = $2 WHERE id = $3 RETURNING updated_at`
	err := r.pool.QueryRow(ctx, query, tenant.Name, tenant.Region, tenant.ID).Scan(&tenant.UpdatedAt)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update tenant")
	}
	return err
}

func (r *tenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete tenant")
	}
	return err
}

// ============================================================================
// Calendar Profile Repository Implementation
// ============================================================================

type profileRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *profileRepo) Create(ctx context.Context, profile *CalendarProfile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	query := `
		INSERT INTO calendar_profiles (id, tenant_id, name, description, timezone, region, is_active, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		profile.ID, profile.TenantID, profile.Name, profile.Description,
		profile.Timezone, profile.Region, profile.IsActive, profile.Version,
	).Scan(&profile.CreatedAt, &profile.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create calendar profile")
	}
	return err
}

func (r *profileRepo) GetByID(ctx context.Context, id uuid.UUID) (*CalendarProfile, error) {
	query := `
		SELECT id, tenant_id, name, description, timezone, region, is_active, version, created_at, updated_at 
		FROM calendar_profiles 
		WHERE id = $1 AND deleted_at IS NULL
	`
	profile := &CalendarProfile{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.TenantID, &profile.Name, &profile.Description,
		&profile.Timezone, &profile.Region, &profile.IsActive, &profile.Version,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	return profile, err
}

func (r *profileRepo) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*CalendarProfile, error) {
	query := `
		SELECT id, tenant_id, name, description, timezone, region, is_active, version, created_at, updated_at 
		FROM calendar_profiles 
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL
		LIMIT 1
	`
	profile := &CalendarProfile{}
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(
		&profile.ID, &profile.TenantID, &profile.Name, &profile.Description,
		&profile.Timezone, &profile.Region, &profile.IsActive, &profile.Version,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	return profile, err
}

func (r *profileRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*CalendarProfile, error) {
	query := `
		SELECT id, tenant_id, name, description, timezone, region, is_active, version, created_at, updated_at 
		FROM calendar_profiles 
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	profiles := []*CalendarProfile{}
	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		profile := &CalendarProfile{}
		if err := rows.Scan(
			&profile.ID, &profile.TenantID, &profile.Name, &profile.Description,
			&profile.Timezone, &profile.Region, &profile.IsActive, &profile.Version,
			&profile.CreatedAt, &profile.UpdatedAt,
		); err != nil {
			r.logger.WithError(err).Error("Failed to scan profile")
			continue
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func (r *profileRepo) Update(ctx context.Context, profile *CalendarProfile) error {
	query := `
		UPDATE calendar_profiles 
		SET name = $1, description = $2, timezone = $3, region = $4, is_active = $5, version = $6
		WHERE id = $7 
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		profile.Name, profile.Description, profile.Timezone, profile.Region,
		profile.IsActive, profile.Version, profile.ID,
	).Scan(&profile.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to update profile")
	}
	return err
}

func (r *profileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE calendar_profiles SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete profile")
	}
	return err
}

func (r *profileRepo) InvalidateCache(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE resolved_calendar_metadata SET content_hash = NULL, resolved_at = NULL WHERE profile_id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ============================================================================
// Holiday Repository Implementation
// ============================================================================

type holidayRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *holidayRepo) Create(ctx context.Context, holiday *Holiday) error {
	if holiday.ID == uuid.Nil {
		holiday.ID = uuid.New()
	}

	query := `
		INSERT INTO holidays (id, profile_id, tenant_id, holiday_date, name, region)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		holiday.ID, holiday.ProfileID, holiday.TenantID, holiday.HolidayDate, holiday.Name, holiday.Region,
	).Scan(&holiday.CreatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create holiday")
	}
	return err
}

func (r *holidayRepo) GetByID(ctx context.Context, id uuid.UUID) (*Holiday, error) {
	query := `SELECT id, profile_id, tenant_id, holiday_date, name, region, created_at FROM holidays WHERE id = $1`
	holiday := &Holiday{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&holiday.ID, &holiday.ProfileID, &holiday.TenantID, &holiday.HolidayDate, &holiday.Name, &holiday.Region, &holiday.CreatedAt,
	)
	return holiday, err
}

func (r *holidayRepo) ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*Holiday, error) {
	query := `SELECT id, profile_id, tenant_id, holiday_date, name, region, created_at FROM holidays WHERE profile_id = $1 ORDER BY holiday_date ASC`
	holidays := []*Holiday{}
	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		holiday := &Holiday{}
		if err := rows.Scan(
			&holiday.ID, &holiday.ProfileID, &holiday.TenantID, &holiday.HolidayDate, &holiday.Name, &holiday.Region, &holiday.CreatedAt,
		); err != nil {
			r.logger.WithError(err).Error("Failed to scan holiday")
			continue
		}
		holidays = append(holidays, holiday)
	}
	return holidays, nil
}

func (r *holidayRepo) ListByDateRange(ctx context.Context, profileID uuid.UUID, start, end time.Time) ([]*Holiday, error) {
	query := `
		SELECT id, profile_id, tenant_id, holiday_date, name, region, created_at 
		FROM holidays 
		WHERE profile_id = $1 AND holiday_date >= $2 AND holiday_date <= $3 
		ORDER BY holiday_date ASC
	`
	holidays := []*Holiday{}
	rows, err := r.pool.Query(ctx, query, profileID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		holiday := &Holiday{}
		if err := rows.Scan(
			&holiday.ID, &holiday.ProfileID, &holiday.TenantID, &holiday.HolidayDate, &holiday.Name, &holiday.Region, &holiday.CreatedAt,
		); err != nil {
			continue
		}
		holidays = append(holidays, holiday)
	}
	return holidays, nil
}

func (r *holidayRepo) Update(ctx context.Context, holiday *Holiday) error {
	query := `UPDATE holidays SET name = $1, region = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, holiday.Name, holiday.Region, holiday.ID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to update holiday")
	}
	return err
}

func (r *holidayRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM holidays WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete holiday")
	}
	return err
}

func (r *holidayRepo) DeleteByProfile(ctx context.Context, profileID uuid.UUID) error {
	query := `DELETE FROM holidays WHERE profile_id = $1`
	_, err := r.pool.Exec(ctx, query, profileID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete holidays by profile")
	}
	return err
}

func (r *holidayRepo) Upsert(ctx context.Context, holiday *Holiday) error {
	if holiday.ID == uuid.Nil {
		holiday.ID = uuid.New()
	}

	query := `
		INSERT INTO holidays (id, profile_id, tenant_id, holiday_date, name, region)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (profile_id, holiday_date) DO UPDATE
		SET name = EXCLUDED.name, region = EXCLUDED.region
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		holiday.ID, holiday.ProfileID, holiday.TenantID, holiday.HolidayDate, holiday.Name, holiday.Region,
	).Scan(&holiday.CreatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to upsert holiday")
	}
	return err
}

// ============================================================================
// Blackout Window Repository Implementation
// ============================================================================

type blackoutRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *blackoutRepo) Create(ctx context.Context, blackout *BlackoutWindow) error {
	if blackout.ID == uuid.Nil {
		blackout.ID = uuid.New()
	}

	query := `
		INSERT INTO blackout_windows (id, profile_id, tenant_id, start_time, end_time, title, reason, rrule, is_recurring, recurrence_start, recurrence_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		blackout.ID, blackout.ProfileID, blackout.TenantID, blackout.StartTime, blackout.EndTime,
		blackout.Title, blackout.Reason, blackout.RRULE, blackout.IsRecurring,
		blackout.RecurrenceStart, blackout.RecurrenceEnd,
	).Scan(&blackout.CreatedAt, &blackout.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to create blackout")
	}
	return err
}

func (r *blackoutRepo) GetByID(ctx context.Context, id uuid.UUID) (*BlackoutWindow, error) {
	query := `
		SELECT id, profile_id, tenant_id, start_time, end_time, title, reason, rrule, is_recurring, recurrence_start, recurrence_end, created_at, updated_at 
		FROM blackout_windows WHERE id = $1
	`
	blackout := &BlackoutWindow{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&blackout.ID, &blackout.ProfileID, &blackout.TenantID, &blackout.StartTime, &blackout.EndTime,
		&blackout.Title, &blackout.Reason, &blackout.RRULE, &blackout.IsRecurring,
		&blackout.RecurrenceStart, &blackout.RecurrenceEnd, &blackout.CreatedAt, &blackout.UpdatedAt,
	)
	return blackout, err
}

func (r *blackoutRepo) ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*BlackoutWindow, error) {
	query := `
		SELECT id, profile_id, tenant_id, start_time, end_time, title, reason, rrule, is_recurring, recurrence_start, recurrence_end, created_at, updated_at 
		FROM blackout_windows WHERE profile_id = $1 ORDER BY start_time ASC
	`
	blackouts := []*BlackoutWindow{}
	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		blackout := &BlackoutWindow{}
		if err := rows.Scan(
			&blackout.ID, &blackout.ProfileID, &blackout.TenantID, &blackout.StartTime, &blackout.EndTime,
			&blackout.Title, &blackout.Reason, &blackout.RRULE, &blackout.IsRecurring,
			&blackout.RecurrenceStart, &blackout.RecurrenceEnd, &blackout.CreatedAt, &blackout.UpdatedAt,
		); err != nil {
			r.logger.WithError(err).Error("Failed to scan blackout")
			continue
		}
		blackouts = append(blackouts, blackout)
	}
	return blackouts, nil
}

func (r *blackoutRepo) ListByDateRange(ctx context.Context, profileID uuid.UUID, start, end time.Time) ([]*BlackoutWindow, error) {
	query := `
		SELECT id, profile_id, tenant_id, start_time, end_time, title, reason, rrule, is_recurring, recurrence_start, recurrence_end, created_at, updated_at 
		FROM blackout_windows 
		WHERE profile_id = $1 AND start_time < $3 AND end_time > $2 
		ORDER BY start_time ASC
	`
	blackouts := []*BlackoutWindow{}
	rows, err := r.pool.Query(ctx, query, profileID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		blackout := &BlackoutWindow{}
		if err := rows.Scan(
			&blackout.ID, &blackout.ProfileID, &blackout.TenantID, &blackout.StartTime, &blackout.EndTime,
			&blackout.Title, &blackout.Reason, &blackout.RRULE, &blackout.IsRecurring,
			&blackout.RecurrenceStart, &blackout.RecurrenceEnd, &blackout.CreatedAt, &blackout.UpdatedAt,
		); err != nil {
			continue
		}
		blackouts = append(blackouts, blackout)
	}
	return blackouts, nil
}

func (r *blackoutRepo) Update(ctx context.Context, blackout *BlackoutWindow) error {
	query := `
		UPDATE blackout_windows 
		SET title = $1, reason = $2, start_time = $3, end_time = $4, rrule = $5, is_recurring = $6, recurrence_start = $7, recurrence_end = $8
		WHERE id = $9 
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		blackout.Title, blackout.Reason, blackout.StartTime, blackout.EndTime,
		blackout.RRULE, blackout.IsRecurring, blackout.RecurrenceStart, blackout.RecurrenceEnd,
		blackout.ID,
	).Scan(&blackout.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to update blackout")
	}
	return err
}

func (r *blackoutRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM blackout_windows WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete blackout")
	}
	return err
}

func (r *blackoutRepo) DeleteByProfile(ctx context.Context, profileID uuid.UUID) error {
	query := `DELETE FROM blackout_windows WHERE profile_id = $1`
	_, err := r.pool.Exec(ctx, query, profileID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete blackouts by profile")
	}
	return err
}

// ============================================================================
// Metadata Repository Implementation
// ============================================================================

type metadataRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *metadataRepo) Upsert(ctx context.Context, metadata *ResolvedCalendarMetadata) error {
	if metadata.ID == uuid.Nil {
		metadata.ID = uuid.New()
	}

	query := `
		INSERT INTO resolved_calendar_metadata (id, tenant_id, profile_id, region, resolved_at, version, holidays_count, blackouts_count, content_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, profile_id, region) DO UPDATE
		SET resolved_at = EXCLUDED.resolved_at, version = EXCLUDED.version, holidays_count = EXCLUDED.holidays_count, blackouts_count = EXCLUDED.blackouts_count, content_hash = EXCLUDED.content_hash
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		metadata.ID, metadata.TenantID, metadata.ProfileID, metadata.Region,
		metadata.ResolvedAt, metadata.Version, metadata.HolidaysCount, metadata.BlackoutsCount, metadata.ContentHash,
	).Scan(&metadata.ID, &metadata.CreatedAt, &metadata.UpdatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to upsert metadata")
	}
	return err
}

func (r *metadataRepo) GetByProfile(ctx context.Context, tenantID, profileID uuid.UUID, region string) (*ResolvedCalendarMetadata, error) {
	query := `
		SELECT id, tenant_id, profile_id, region, resolved_at, version, holidays_count, blackouts_count, content_hash, created_at, updated_at 
		FROM resolved_calendar_metadata 
		WHERE tenant_id = $1 AND profile_id = $2 AND region = $3
	`
	metadata := &ResolvedCalendarMetadata{}
	err := r.pool.QueryRow(ctx, query, tenantID, profileID, region).Scan(
		&metadata.ID, &metadata.TenantID, &metadata.ProfileID, &metadata.Region,
		&metadata.ResolvedAt, &metadata.Version, &metadata.HolidaysCount, &metadata.BlackoutsCount,
		&metadata.ContentHash, &metadata.CreatedAt, &metadata.UpdatedAt,
	)
	return metadata, err
}

func (r *metadataRepo) InvalidateByProfile(ctx context.Context, profileID uuid.UUID) error {
	query := `UPDATE resolved_calendar_metadata SET resolved_at = NULL, content_hash = NULL WHERE profile_id = $1`
	_, err := r.pool.Exec(ctx, query, profileID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to invalidate metadata")
	}
	return err
}

// ============================================================================
// Audit Log Repository Implementation
// ============================================================================

type auditLogRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
}

func (r *auditLogRepo) Log(ctx context.Context, log *AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	changes, _ := json.Marshal(log.Changes)
	query := `
		INSERT INTO audit_logs (id, tenant_id, entity_type, entity_id, action, changes, performed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		log.ID, log.TenantID, log.EntityType, log.EntityID, log.Action, changes, log.PerformedBy,
	).Scan(&log.CreatedAt)

	if err != nil {
		r.logger.WithError(err).Error("Failed to log audit entry")
	}
	return err
}

func (r *auditLogRepo) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, tenant_id, entity_type, entity_id, action, changes, performed_by, created_at 
		FROM audit_logs 
		WHERE entity_type = $1 AND entity_id = $2 
		ORDER BY created_at DESC 
		LIMIT $3
	`
	logs := []*AuditLog{}
	rows, err := r.pool.Query(ctx, query, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		log := &AuditLog{}
		var changes json.RawMessage
		if err := rows.Scan(
			&log.ID, &log.TenantID, &log.EntityType, &log.EntityID, &log.Action, &changes, &log.PerformedBy, &log.CreatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(changes, &log.Changes)
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *auditLogRepo) GetByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*AuditLog, error) {
	query := `
		SELECT id, tenant_id, entity_type, entity_id, action, changes, performed_by, created_at 
		FROM audit_logs 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`
	logs := []*AuditLog{}
	rows, err := r.pool.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		log := &AuditLog{}
		var changes json.RawMessage
		if err := rows.Scan(
			&log.ID, &log.TenantID, &log.EntityType, &log.EntityID, &log.Action, &changes, &log.PerformedBy, &log.CreatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(changes, &log.Changes)
		logs = append(logs, log)
	}
	return logs, nil
}
