package mdm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSurvivorshipEngine_ResolveField(t *testing.T) {
	engine := NewSurvivorshipEngine(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	now := time.Now().UTC()

	// 1. Test Most Recent strategy (default in-memory)
	sources := []FieldSourceRecord{
		{
			SourceProvider: "CRIMS",
			Value:          100.0,
			Timestamp:      now.Add(-2 * time.Hour),
			Confidence:     0.95,
		},
		{
			SourceProvider: "BLOOMBERG",
			Value:          102.5,
			Timestamp:      now.Add(-5 * time.Minute),
			Confidence:     0.99,
		},
		{
			SourceProvider: "REFINITIV",
			Value:          101.0,
			Timestamp:      now.Add(-1 * time.Hour),
			Confidence:     0.98,
		},
	}

	res, err := engine.ResolveField(ctx, tenantID, "SECURITY_PRICE", "px_last", sources)
	if err != nil {
		t.Fatalf("ResolveField failed: %v", err)
	}

	if res.WinningSource != "BLOOMBERG" {
		t.Errorf("expected winning source BLOOMBERG, got %s", res.WinningSource)
	}
	if res.ResolvedValue != 102.5 {
		t.Errorf("expected resolved value 102.5, got %v", res.ResolvedValue)
	}
}
