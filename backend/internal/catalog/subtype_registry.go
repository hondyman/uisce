package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SubtypeRow struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenantId"`
	RootObject        string    `json:"rootObject"`
	SubtypeCode       string    `json:"subtypeCode"`
	DisplayName       string    `json:"displayName"`
	ParentSubtypeCode *string   `json:"parentSubtypeCode,omitempty"`
	FieldAllowlist    []string  `json:"fieldAllowlist"`
	IsActive          bool      `json:"isActive"`
	CreatedAt         time.Time `json:"createdAt"`
}

type SubtypeRegistryLoader interface {
	LoadAllForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) ([]SubtypeRow, error)
}

type CachedSubtypeRegistryLoader struct {
	mu    sync.Mutex
	cache map[uuid.UUID]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	rows      []SubtypeRow
	expiresAt time.Time
}

func NewSubtypeRegistryLoader(ttl time.Duration) *CachedSubtypeRegistryLoader {
	return &CachedSubtypeRegistryLoader{
		cache: make(map[uuid.UUID]cacheEntry),
		ttl:   ttl,
	}
}

func (l *CachedSubtypeRegistryLoader) LoadAllForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) ([]SubtypeRow, error) {
	l.mu.Lock()
	entry, found := l.cache[tenantID]
	if found && time.Now().Before(entry.expiresAt) {
		l.mu.Unlock()
		return entry.rows, nil
	}
	l.mu.Unlock()

	query := `
		SELECT id, tenant_id, root_object, subtype_code, display_name, parent_subtype_code, field_allowlist, is_active, created_at
		FROM oms.subtype_registry
		WHERE tenant_id = $1 AND is_active = true
	`
	rows, err := db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load subtype registry: %w", err)
	}
	defer rows.Close()

	var result []SubtypeRow
	for rows.Next() {
		var r SubtypeRow
		var allowlistRaw []byte
		if err := rows.Scan(&r.ID, &r.TenantID, &r.RootObject, &r.SubtypeCode, &r.DisplayName, &r.ParentSubtypeCode, &allowlistRaw, &r.IsActive, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan subtype row: %w", err)
		}
		if len(allowlistRaw) > 0 {
			_ = json.Unmarshal(allowlistRaw, &r.FieldAllowlist)
		}
		result = append(result, r)
	}

	l.mu.Lock()
	l.cache[tenantID] = cacheEntry{
		rows:      result,
		expiresAt: time.Now().Add(l.ttl),
	}
	l.mu.Unlock()

	return result, nil
}
