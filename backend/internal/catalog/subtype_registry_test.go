package catalog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSubtypeRegistryLoader_LoadAllForTenant(t *testing.T) {
	loader := NewSubtypeRegistryLoader(time.Minute)

	if loader == nil {
		t.Fatal("expected non-nil loader")
	}

	if loader.ttl != time.Minute {
		t.Errorf("expected ttl of 1 minute, got %v", loader.ttl)
	}
}

func TestCachedSubtypeRegistryLoader_CacheHit(t *testing.T) {
	tenantID := uuid.New()
	loader := NewSubtypeRegistryLoader(time.Hour)

	ctx := context.Background()

	rows := []SubtypeRow{
		{
			ID:                uuid.New(),
			TenantID:          tenantID,
			RootObject:        "account",
			SubtypeCode:       "institutional",
			DisplayName:       "Institutional Account",
			FieldAllowlist:    []string{"account_number", "sponsor_id"},
			IsActive:          true,
			CreatedAt:         time.Now(),
		},
	}

	loader.mu.Lock()
	loader.cache[tenantID] = cacheEntry{
		rows:      rows,
		expiresAt: time.Now().Add(time.Hour),
	}
	loader.mu.Unlock()

	got, err := loader.LoadAllForTenant(ctx, nil, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != len(rows) {
		t.Errorf("expected %d rows, got %d", len(rows), len(got))
	}

	if got[0].SubtypeCode != "institutional" {
		t.Errorf("expected subtype 'institutional', got %q", got[0].SubtypeCode)
	}
}

func TestSubtypeRow_FieldAllowlistJSON(t *testing.T) {
	raw := []byte(`["account_number","sponsor_id","account_name"]`)
	var allowlist []string
	if err := json.Unmarshal(raw, &allowlist); err != nil {
		t.Fatalf("failed to unmarshal allowlist: %v", err)
	}

	if len(allowlist) != 3 {
		t.Errorf("expected 3 items, got %d", len(allowlist))
	}

	if allowlist[0] != "account_number" {
		t.Errorf("expected first item 'account_number', got %q", allowlist[0])
	}
}
