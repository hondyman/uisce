package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOutbox_CalculatePayloadHash(t *testing.T) {
	payload := map[string]interface{}{
		"bo_key":            "customer_profile",
		"active_version_id": 1,
		"field_count":       12,
	}
	bytes, _ := json.Marshal(payload)
	hash1 := CalculatePayloadHash(bytes)
	hash2 := CalculatePayloadHash(bytes)

	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash, got %s != %s", hash1, hash2)
	}
	if len(hash1) != 64 {
		t.Fatalf("expected 64-char SHA256 hex string, got length %d (%s)", len(hash1), hash1)
	}
}

func TestOutbox_CalculateChainSeal(t *testing.T) {
	key := []byte("sec-rule-17a4-secret")
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	payloadHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	seal1 := CalculateChainSeal(key, tenantID, "BO_PUBLISHED", payloadHash, now)
	seal2 := CalculateChainSeal(key, tenantID, "BO_PUBLISHED", payloadHash, now)

	if seal1 != seal2 {
		t.Fatalf("expected deterministic chain seal, got %s != %s", seal1, seal2)
	}

	// Tampered payload hash should produce completely different seal
	tamperedHash := "f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	tamperedSeal := CalculateChainSeal(key, tenantID, "BO_PUBLISHED", tamperedHash, now)
	if seal1 == tamperedSeal {
		t.Fatalf("tampered seal should not match original")
	}
}

func TestOutbox_MemoryPublisherRelay(t *testing.T) {
	pub := NewMemoryEventPublisher()
	ctx := context.Background()

	err := pub.Publish(ctx, "uisce.audit.governance.v1", "tenant-100", []byte(`{"action":"BO_PUBLISHED"}`))
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if len(pub.PublishedMessages) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.PublishedMessages))
	}
	if pub.PublishedMessages[0].Topic != "uisce.audit.governance.v1" {
		t.Errorf("expected topic uisce.audit.governance.v1, got %s", pub.PublishedMessages[0].Topic)
	}
}
