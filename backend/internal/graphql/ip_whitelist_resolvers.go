package graphql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Input validation helper
func validateIPEntryInput(input IPWhitelistEntryInput) error {
	if input.IPAddress == "" {
		return fmt.Errorf("ipAddress is required")
	}
	// Validate IP format
	if parsed := net.ParseIP(input.IPAddress); parsed == nil {
		return fmt.Errorf("ipAddress is not a valid IP: %s", input.IPAddress)
	}
	return nil
}

// Query: get all entries
func (r *Resolver) IpWhitelistEntries(ctx context.Context) ([]*IPWhitelistEntry, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, tenant_id, ip_address, description, created_at, updated_at FROM ip_whitelist_entries`)
	if err != nil {
		return nil, fmt.Errorf("query ip_whitelist_entries: %w", err)
	}
	defer rows.Close()
	var entries []*IPWhitelistEntry
	for rows.Next() {
		var e IPWhitelistEntry
		var tid sql.NullString
		var desc sql.NullString
		if err := rows.Scan(&e.ID, &tid, &e.IPAddress, &desc, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ip_whitelist_entries row: %w", err)
		}
		if tid.Valid {
			if tidVal, err := uuid.Parse(tid.String); err == nil {
				e.TenantID = &tidVal
			}
		}
		if desc.Valid {
			e.Description = &desc.String
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return entries, nil
}

// Query: get entry by id
func (r *Resolver) IpWhitelistEntry(ctx context.Context, id uuid.UUID) (*IPWhitelistEntry, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, tenant_id, ip_address, description, created_at, updated_at FROM ip_whitelist_entries WHERE id = $1`, id)
	var e IPWhitelistEntry
	var tid sql.NullString
	var desc sql.NullString
	if err := row.Scan(&e.ID, &tid, &e.IPAddress, &desc, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("ip whitelist entry not found: %w", err)
		}
		return nil, fmt.Errorf("scan ip_whitelist_entry: %w", err)
	}
	if tid.Valid {
		if tidVal, err := uuid.Parse(tid.String); err == nil {
			e.TenantID = &tidVal
		}
	}
	if desc.Valid {
		e.Description = &desc.String
	}
	return &e, nil
}

// Query: get entries by tenant
func (r *Resolver) IpWhitelistEntriesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*IPWhitelistEntry, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, tenant_id, ip_address, description, created_at, updated_at FROM ip_whitelist_entries WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query ip_whitelist_entries by tenant: %w", err)
	}
	defer rows.Close()
	var entries []*IPWhitelistEntry
	for rows.Next() {
		var e IPWhitelistEntry
		var tid sql.NullString
		var desc sql.NullString
		if err := rows.Scan(&e.ID, &tid, &e.IPAddress, &desc, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ip_whitelist_entries by tenant row: %w", err)
		}
		if tid.Valid {
			if tidVal, err := uuid.Parse(tid.String); err == nil {
				e.TenantID = &tidVal
			}
		}
		if desc.Valid {
			e.Description = &desc.String
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return entries, nil
}

// Mutation: create entry (uses transaction)
func (r *Resolver) CreateIpWhitelistEntry(ctx context.Context, input IPWhitelistEntryInput) (*IPWhitelistEntry, error) {
	if err := validateIPEntryInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Use a single INSERT ... SELECT ... WHERE NOT EXISTS to avoid a separate pre-check
	// and reduce race windows. We still rely on DB unique indexes as ultimate enforcement.
	id := uuid.New()
	now := time.Now().UTC()

	// Build a tenant-aware NOT EXISTS condition: if tenantId is NULL we check tenant_id IS NULL,
	// otherwise we check equality.
	var res sql.Result
	var err error
	if input.TenantID != nil {
		res, err = r.DB.ExecContext(ctx,
			`
			INSERT INTO ip_whitelist_entries (id, tenant_id, ip_address, description, created_at, updated_at)
			SELECT $1, $2, $3, $4, $5, $6
			WHERE NOT EXISTS (
				SELECT 1 FROM ip_whitelist_entries WHERE ip_address = $3 AND tenant_id = $2
			)
			`,
			id, input.TenantID, input.IPAddress, input.Description, now, now)
	} else {
		res, err = r.DB.ExecContext(ctx,
			`
			INSERT INTO ip_whitelist_entries (id, tenant_id, ip_address, description, created_at, updated_at)
			SELECT $1, NULL, $2, $3, $4, $5
			WHERE NOT EXISTS (
				SELECT 1 FROM ip_whitelist_entries WHERE ip_address = $2 AND tenant_id IS NULL
			)
			`,
			id, input.IPAddress, input.Description, now, now)
	}
	if err != nil {
		return nil, fmt.Errorf("insert ip_whitelist_entry: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("ipAddress already exists")
	}

	return r.IpWhitelistEntry(ctx, id)
}

// Mutation: update entry (uses transaction)
func (r *Resolver) UpdateIpWhitelistEntry(ctx context.Context, id uuid.UUID, input IPWhitelistEntryInput) (*IPWhitelistEntry, error) {
	if err := validateIPEntryInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Uniqueness check for update: ensure if ip_address is changing it does not collide with others
	var existingID uuid.UUID
	var scanErr error
	if input.TenantID != nil {
		scanErr = r.DB.QueryRowContext(ctx, `SELECT id FROM ip_whitelist_entries WHERE ip_address = $1 AND tenant_id = $2`, input.IPAddress, input.TenantID).Scan(&existingID)
	} else {
		scanErr = r.DB.QueryRowContext(ctx, `SELECT id FROM ip_whitelist_entries WHERE ip_address = $1`, input.IPAddress).Scan(&existingID)
	}
	if scanErr == nil && existingID != id {
		return nil, fmt.Errorf("ipAddress already exists")
	}
	if scanErr != nil && scanErr != sql.ErrNoRows {
		return nil, fmt.Errorf("uniqueness check failed: %w", scanErr)
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		`UPDATE ip_whitelist_entries SET tenant_id = $2, ip_address = $3, description = $4, updated_at = $5 WHERE id = $1`,
		id, input.TenantID, input.IPAddress, input.Description, now)
	if err != nil {
		// Try to detect unique constraint violation and return a friendlier message
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("ipAddress already exists")
		}
		return nil, fmt.Errorf("update ip_whitelist_entry: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("ip whitelist entry not found")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.IpWhitelistEntry(ctx, id)
}

// Mutation: delete entry
func (r *Resolver) DeleteIpWhitelistEntry(ctx context.Context, id uuid.UUID) (bool, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `DELETE FROM ip_whitelist_entries WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete ip_whitelist_entry: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return rowsAffected > 0, nil
}
