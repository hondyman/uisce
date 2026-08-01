package mdm

import (
	"context"
	"math"
	"strings"
	"time"
)

// SourcePayload represents a single source's contribution to a Golden Record.
// Common sources include Bloomberg, Refinitiv, CRIMS, and internal systems.
type SourcePayload struct {
	SourceID  string         `json:"source_id"`  // e.g. "BLOOMBERG", "REFINITIV"
	Timestamp time.Time      `json:"timestamp"`  // Ingestion timestamp
	Data      map[string]any `json:"data"`       // Raw record fields
}

// FieldRule declares the survivorship strategy for a single field.
// Strategies are looked up by field name from the survivorship_rules table.
type FieldRule struct {
	Strategy        string   `json:"strategy"`
	PriorityOrder   []string `json:"priority_order"`
	MaxStaleSeconds int      `json:"max_stale_seconds"`
}

// SurvivorshipEngine merges multiple SourcePayload records into a single
// Golden Record map[string]any that can be projected into a FastRecord
// for zero-allocation VM evaluation.
//
// The engine runs pre-VM and is stateless. Rules can be supplied per call
// or cached in a higher-level service keyed by (tenant_id, bo_id).
type SurvivorshipEngine struct{}

func NewSurvivorshipEngine() *SurvivorshipEngine {
	return &SurvivorshipEngine{}
}

// MergeToGoldenRecord resolves multiple source payloads into a single
// Golden Record map. Fields with no explicit rule default to MOST_RECENT.
//
// Returns an empty map (not nil) if sources is empty, so callers can
// safely pass the result into vm.Project().
func (e *SurvivorshipEngine) MergeToGoldenRecord(
	ctx context.Context,
	sources []SourcePayload,
	rules map[string]FieldRule,
	now time.Time,
) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	golden := make(map[string]any)
	if len(sources) == 0 {
		return golden, nil
	}

	allFields := make(map[string]bool)
	for _, src := range sources {
		for k := range src.Data {
			allFields[k] = true
		}
	}

	for field := range allFields {
		rule, hasRule := rules[field]
		if !hasRule {
			rule = FieldRule{Strategy: "MOST_RECENT"}
		}

		val := e.resolveField(field, sources, rule, now)
		if val != nil {
			golden[field] = val
		}
	}

	return golden, nil
}

// resolveField picks the winning value for a single field across all sources
// using the declared strategy. Sources whose Data does not contain the
// field, or which exceed the staleness window, are filtered out before
// strategy application.
func (e *SurvivorshipEngine) resolveField(
	field string,
	sources []SourcePayload,
	rule FieldRule,
	now time.Time,
) any {
	validSources := make([]SourcePayload, 0, len(sources))
	for _, src := range sources {
		if _, exists := src.Data[field]; !exists {
			continue
		}
		if rule.MaxStaleSeconds > 0 && !src.Timestamp.IsZero() {
			if now.Sub(src.Timestamp).Seconds() > float64(rule.MaxStaleSeconds) {
				continue
			}
		}
		validSources = append(validSources, src)
	}

	if len(validSources) == 0 {
		return nil
	}

	switch rule.Strategy {
	case "SOURCE_PRIORITY":
		for _, targetSrc := range rule.PriorityOrder {
			for _, src := range validSources {
				if strings.EqualFold(src.SourceID, targetSrc) {
					return src.Data[field]
				}
			}
		}
		return validSources[0].Data[field]

	case "MOST_RECENT":
		var latestSource SourcePayload
		var latestTime time.Time
		for _, src := range validSources {
			if src.Timestamp.After(latestTime) || latestTime.IsZero() {
				latestTime = src.Timestamp
				latestSource = src
			}
		}
		return latestSource.Data[field]

	case "CONSERVATIVE_MIN":
		var minVal float64 = math.MaxFloat64
		var found bool
		for _, src := range validSources {
			if v, ok := toFloat64(src.Data[field]); ok {
				if v < minVal {
					minVal = v
					found = true
				}
			}
		}
		if found {
			return minVal
		}
		return validSources[0].Data[field]

	case "CONSERVATIVE_MAX":
		var maxVal float64 = -math.MaxFloat64
		var found bool
		for _, src := range validSources {
			if v, ok := toFloat64(src.Data[field]); ok {
				if v > maxVal {
					maxVal = v
					found = true
				}
			}
		}
		if found {
			return maxVal
		}
		return validSources[0].Data[field]

	default:
		return validSources[0].Data[field]
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
