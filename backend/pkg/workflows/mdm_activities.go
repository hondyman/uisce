package workflows

import (
	"context"
	"fmt"
)

// MDMActivities handles validation against Master Data Management systems
type MDMActivities struct {
	// In a real system, this would hold clients to external MDM (Informatica, Tibco, etc.)
}

func NewMDMActivities() *MDMActivities {
	return &MDMActivities{}
}

type MDMValidationRequest struct {
	EntityType string                 `json:"entityType"` // e.g. "Counterparty"
	EntityID   string                 `json:"entityId"`
	Attributes map[string]interface{} `json:"attributes"` // The data we want to write
}

// ActivityValidateGoldenRecord checks if the proposed attributes match the Trusted Source.
// Use Case: A user tries to update a Risk Rating to "Low", but the MDM says it is "High".
func (a *MDMActivities) ActivityValidateGoldenRecord(ctx context.Context, req MDMValidationRequest) (map[string]interface{}, error) {
	// activity.RecordHeartbeat(ctx, "Checking MDM...")

	// 1. Fetch Golden Record (Mocked)
	goldenRecord, err := a.fetchGoldenRecord(req.EntityType, req.EntityID)
	if err != nil {
		return nil, fmt.Errorf("MDM lookup failed: %w", err)
	}

	// 2. Compare Attributes
	mismatches := []string{}
	for key, proposedVal := range req.Attributes {
		goldenVal, exists := goldenRecord[key]
		if !exists {
			// If field doesn't exist in MDM, maybe we allow or warn?
			// For strict governance, we might error.
			continue
		}

		if proposedVal != goldenVal {
			mismatches = append(mismatches, fmt.Sprintf("%s: Proposed='%v', Golden='%v'", key, proposedVal, goldenVal))
		}
	}

	if len(mismatches) > 0 {
		return nil, fmt.Errorf("GOLDEN_RECORD_VIOLATION: Data does not match source of truth: %v", mismatches)
	}

	return map[string]interface{}{
		"validation_status": "MATCH",
		"mdm_version":       "v1.5",
	}, nil
}

// Mock MDM Data Store
func (a *MDMActivities) fetchGoldenRecord(entityType, id string) (map[string]interface{}, error) {
	// Simulating a "Counterparty" database
	if entityType == "Counterparty" {
		if id == "CP-123" {
			return map[string]interface{}{
				"risk_rating": "HIGH",
				"kyc_status":  "APPROVED",
				"country":     "US",
			}, nil
		}
		if id == "CP-999" {
			return nil, fmt.Errorf("entity not found in MDM")
		}
	}
	return map[string]interface{}{}, nil
}
