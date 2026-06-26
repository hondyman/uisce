package engine

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/models"
)

func TestValidationEngine_LoadRuleSet(t *testing.T) {
	engine := NewValidationEngine("../../policy")

	tests := []struct {
		name        string
		version     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "load 2025 rules successfully",
			version: "2025",
			wantErr: false,
		},
		{
			name:    "load 2021 rules successfully",
			version: "2021",
			wantErr: false,
		},
		{
			name:        "non-existent version",
			version:     "2099",
			wantErr:     true,
			errContains: "policy file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := engine.LoadRuleSet(tt.version)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !val.Exists() {
				t.Error("Loaded value does not exist")
			}
		})
	}
}

func TestValidationEngine_Validate(t *testing.T) {
	engine := NewValidationEngine("../../policy")
	ctx := context.Background()

	tests := []struct {
		name      string
		trade     models.TradeRequest
		version   string
		checkType string
		wantErr   bool
	}{
		{
			name: "valid limit order within limits",
			trade: models.TradeRequest{
				ID:         "TXN-001",
				TradeDate:  "2025-12-29",
				Amount:     500000,
				Currency:   "USD",
				OrderType:  "LIMIT",
				LimitPrice: ptrFloat64(150.0),
			},
			version:   "2025",
			checkType: "PreTrade",
			wantErr:   false,
		},
		{
			name: "invalid high value market order",
			trade: models.TradeRequest{
				ID:        "TXN-002",
				TradeDate: "2025-12-29",
				Amount:    2000000, // > 1M triggers limit order requirement
				Currency:  "USD",
				OrderType: "MARKET",
			},
			version:   "2025",
			checkType: "PreTrade",
			wantErr:   true, // Should fail - high value must be LIMIT
		},
		{
			name: "limit order missing limit price",
			trade: models.TradeRequest{
				ID:        "TXN-003",
				TradeDate: "2025-12-29",
				Amount:    100000,
				Currency:  "USD",
				OrderType: "LIMIT",
				// LimitPrice is missing - should fail
			},
			version:   "2025",
			checkType: "PreTrade",
			wantErr:   true,
		},
		{
			name: "2021 rules more lenient",
			trade: models.TradeRequest{
				ID:        "TXN-004",
				TradeDate: "2021-06-15",
				Amount:    1500000, // Between 1M and 2M
				Currency:  "USD",
				OrderType: "MARKET", // Allowed in 2021 (<2M threshold)
			},
			version:   "2021",
			checkType: "PreTrade",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(ctx, tt.trade, tt.version, tt.checkType)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationEngine_CacheWorks(t *testing.T) {
	engine := NewValidationEngine("../../policy")

	// First load
	_, err := engine.LoadRuleSet("2025")
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Second load should hit cache
	_, err = engine.LoadRuleSet("2025")
	if err != nil {
		t.Fatal("Second load (cached) failed: %v", err)
	}

	// Verify cache has entry
	engine.cacheMutex.RLock()
	_, exists := engine.cache["2025"]
	engine.cacheMutex.RUnlock()

	if !exists {
		t.Error("Expected cache entry for version 2025")
	}

	// Clear cache
	engine.ClearCache()

	engine.cacheMutex.RLock()
	cacheLen := len(engine.cache)
	engine.cacheMutex.RUnlock()

	if cacheLen != 0 {
		t.Errorf("Expected empty cache after clear, got %v entries", cacheLen)
	}
}

func ptrFloat64(f float64) *float64 {
	return &f
}
