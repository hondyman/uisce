package api

import (
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// PlanRequest is the planner request contract. Region and TenantID are required.
type PlanRequest struct {
	Snapshot       *audit.SemanticSnapshot `json:"snapshot,omitempty"`
	TenantID       string                  `json:"tenant_id"`
	Region         string                  `json:"region"`
	BusinessObject string                  `json:"business_object"`
	Dimensions     []string                `json:"dimensions,omitempty"`
	Measures       []string                `json:"measures,omitempty"`
	Filters        []string                `json:"filters,omitempty"`
}

// Validate ensures required fields are present and returns a descriptive error.
func (pr *PlanRequest) Validate() error {
	if pr.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if pr.Region == "" {
		return fmt.Errorf("region is required for all semantic operations.")
	}
	if pr.BusinessObject == "" {
		return fmt.Errorf("business_object is required")
	}
	return nil
}
