package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/semlayer/backend/internal/utils/ip"
)

// Whitelist represents the structure for IP whitelist data, matching the frontend.
type Whitelist struct {
	Global  []string            `json:"global"`
	Tenants map[string][]string `json:"tenants"`
}

// IpWhitelistEntry corresponds to a row in the tenant_ip_whitelist_entries table.
type IpWhitelistEntry struct {
	ID        string         `db:"id"`
	TenantID  sql.NullString `db:"tenant_id"`
	IPAddress string         `db:"ip_address"`
}

// IPWhitelistService provides methods for managing IP whitelists.
type IPWhitelistService struct {
	db *sqlx.DB
}

// NewIPWhitelistService creates a new IPWhitelistService.
func NewIPWhitelistService(db *sqlx.DB) *IPWhitelistService {
	return &IPWhitelistService{db: db}
}

// GetWhitelist retrieves the entire IP whitelist configuration from the database.
func (s *IPWhitelistService) GetWhitelist(ctx context.Context) (*Whitelist, error) {
	var entries []IpWhitelistEntry
	// Join with assignments to get tenant_id. Global entries have no assignment.
	query := `
		SELECT e.ip_address, a.tenant_id
		FROM tenant_ip_whitelist_entries e
		LEFT JOIN tenant_ip_whitelist_assignments a ON e.id = a.whitelist_id
	`
	err := s.db.SelectContext(ctx, &entries, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query whitelist entries: %w", err)
	}

	whitelist := &Whitelist{
		Global:  []string{},
		Tenants: make(map[string][]string),
	}

	for _, entry := range entries {
		if entry.TenantID.Valid {
			// Tenant-specific entry
			tenantID := entry.TenantID.String
			whitelist.Tenants[tenantID] = append(whitelist.Tenants[tenantID], entry.IPAddress)
		} else {
			// Global entry
			whitelist.Global = append(whitelist.Global, entry.IPAddress)
		}
	}

	return whitelist, nil
}

// UpdateWhitelist replaces the current IP whitelist configuration with the provided one.
// Note: This implementation is complex due to the normalized schema.
// It assumes a full replacement of the state.
func (s *IPWhitelistService) UpdateWhitelist(ctx context.Context, whitelist Whitelist) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error

	// Clear existing assignments and entries.
	// Note: Deleting entries will cascade delete assignments due to foreign key constraints.
	if _, err := tx.ExecContext(ctx, "DELETE FROM tenant_ip_whitelist_entries"); err != nil {
		return fmt.Errorf("failed to clear existing whitelist: %w", err)
	}

	// Helper to insert entry and assignment
	insertEntry := func(ipAddr string, tenantID *string) error {
		var entryID string
		// Insert entry if not exists (global/shared IPs might be repeated in input but should be unique in DB)
		// We use ON CONFLICT to handle duplicates if the input list has same IP for multiple tenants (though structure suggests separation)
		err := tx.QueryRowContext(ctx, `
			INSERT INTO tenant_ip_whitelist_entries (ip_address) VALUES ($1)
			ON CONFLICT (ip_address) DO UPDATE SET updated_at = now()
			RETURNING id
		`, ipAddr).Scan(&entryID)
		if err != nil {
			return fmt.Errorf("failed to insert IP %s: %w", ipAddr, err)
		}

		if tenantID != nil {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, entryID, *tenantID)
			if err != nil {
				return fmt.Errorf("failed to assign IP %s to tenant %s: %w", ipAddr, *tenantID, err)
			}
		}
		return nil
	}

	// Insert global IPs
	for _, ipAddr := range whitelist.Global {
		if err := insertEntry(ipAddr, nil); err != nil {
			return err
		}
	}

	// Insert tenant-specific IPs
	for tenantID, ips := range whitelist.Tenants {
		for _, ipAddr := range ips {
			tID := tenantID
			if err := insertEntry(ipAddr, &tID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// IsIpAllowed checks if a given IP address is permitted for a tenant.
// It checks against both global rules and rules specific to the tenant.
func (s *IPWhitelistService) IsIpAllowed(ctx context.Context, tenantID, requestIP string) (bool, error) {
	parsedRequestIP := net.ParseIP(requestIP)
	if parsedRequestIP == nil {
		return false, fmt.Errorf("invalid request IP address: %s", requestIP)
	}

	// Fetch applicable whitelist entries in a single query
	query := `
		SELECT e.ip_address
		FROM tenant_ip_whitelist_entries e
		LEFT JOIN tenant_ip_whitelist_assignments a ON e.id = a.whitelist_id
		WHERE a.tenant_id = $1 OR a.tenant_id IS NULL
	`
	var allowedRanges []string
	if err := s.db.SelectContext(ctx, &allowedRanges, query, tenantID); err != nil {
		return false, fmt.Errorf("could not fetch IP whitelist for tenant %s: %w", tenantID, err)
	}

	return ip.IsIpAllowed(allowedRanges, requestIP), nil
}

// WhitelistInput is used for unmarshalling the GraphQL mutation input.
type WhitelistInput struct {
	Global  []string        `json:"global"`
	Tenants json.RawMessage `json:"tenants"`
}
