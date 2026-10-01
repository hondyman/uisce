package mdm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
)

// SurvivorshipEngineMode represents the execution mode of the survivorship engine
type SurvivorshipEngineMode string

const (
	EngineModeLegacy SurvivorshipEngineMode = "legacy"
	EngineModeShadow SurvivorshipEngineMode = "shadow"
	EngineModeNew    SurvivorshipEngineMode = "new"
)

// AttributeDiffRecord records a divergence between legacy oracle and new CTE engine
type AttributeDiffRecord struct {
	EntityKey     string `json:"entity_key"`
	AttributeName string `json:"attribute_name"`
	LegacyValue   any    `json:"legacy_value"`
	NewValue      any    `json:"new_value"`
	Reason        string `json:"reason,omitempty"`
}

// ShadowComparisonSummary contains the comparison metrics from a shadow run
type ShadowComparisonSummary struct {
	TotalEntities   int                   `json:"total_entities"`
	MatchedEntities int                   `json:"matched_entities"`
	TotalDiffs      int                   `json:"total_diffs"`
	AttributeDiffs  []AttributeDiffRecord `json:"attribute_diffs,omitempty"`
	IsParityClean   bool                  `json:"is_parity_clean"`
}

// CompareGoldenRecords compares records produced by legacy oracle vs new CTE engine
func CompareGoldenRecords(
	legacyRecords map[string]map[string]any,
	newRecords map[string]map[string]any,
) ShadowComparisonSummary {
	summary := ShadowComparisonSummary{
		TotalEntities: len(legacyRecords),
		IsParityClean: true,
	}

	for entityKey, legacyAttrs := range legacyRecords {
		newAttrs, exists := newRecords[entityKey]
		if !exists {
			summary.TotalDiffs++
			summary.AttributeDiffs = append(summary.AttributeDiffs, AttributeDiffRecord{
				EntityKey: entityKey,
				Reason:    "Entity missing in new engine output",
			})
			summary.IsParityClean = false
			continue
		}

		entityMatched := true

		// Check all legacy attributes
		for attrName, legVal := range legacyAttrs {
			newVal, newAttrExists := newAttrs[attrName]
			if !newAttrExists && legVal != nil {
				summary.TotalDiffs++
				summary.AttributeDiffs = append(summary.AttributeDiffs, AttributeDiffRecord{
					EntityKey:     entityKey,
					AttributeName: attrName,
					LegacyValue:   legVal,
					NewValue:      nil,
					Reason:        "Attribute missing in new engine",
				})
				entityMatched = false
				continue
			}

			if !isValuesEqual(legVal, newVal) {
				summary.TotalDiffs++
				summary.AttributeDiffs = append(summary.AttributeDiffs, AttributeDiffRecord{
					EntityKey:     entityKey,
					AttributeName: attrName,
					LegacyValue:   legVal,
					NewValue:      newVal,
					Reason:        "Value mismatch",
				})
				entityMatched = false
			}
		}

		if entityMatched {
			summary.MatchedEntities++
		} else {
			summary.IsParityClean = false
		}
	}

	return summary
}

// ComputePayloadHash calculates a deterministic SHA-256 hash for record payload
func ComputePayloadHash(payload any) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload for hash: %w", err)
	}
	h := sha256.Sum256(bytes)
	return hex.EncodeToString(h[:8]), nil // 16-character hex hash prefix
}

func isValuesEqual(v1, v2 any) bool {
	if v1 == nil && v2 == nil {
		return true
	}
	if v1 == nil || v2 == nil {
		return false
	}

	// Normalize numeric types
	switch n1 := v1.(type) {
	case float64:
		if n2, ok := v2.(float64); ok {
			return n1 == n2
		}
		if n2, ok := v2.(int); ok {
			return n1 == float64(n2)
		}
	case int:
		if n2, ok := v2.(int); ok {
			return n1 == n2
		}
		if n2, ok := v2.(float64); ok {
			return float64(n1) == n2
		}
	}

	return reflect.DeepEqual(v1, v2)
}
