package preference

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ImpactAnalysis summarises the expected effect of a source preference change
type ImpactAnalysis struct {
	AffectedDates    int           `json:"affected_dates"`
	ConfidenceDelta  int           `json:"confidence_delta"`
	BusinessImpact   string        `json:"business_impact"` // none, low, moderate, high
	ConfidenceBefore int           `json:"confidence_before"`
	ConfidenceAfter  int           `json:"confidence_after"`
	ChangedDates     []ChangedDate `json:"changed_dates"`
}

// ChangedDate describes one date whose source changes under the proposed preference
type ChangedDate struct {
	Date          string `json:"date"`
	OldSource     string `json:"old_source"`
	NewSource     string `json:"new_source"`
	OldConfidence int    `json:"old_confidence"`
	NewConfidence int    `json:"new_confidence"`
}

// SourcePreference is the canonical record for a tenant's preferred data source
// for a given BusinessObject / SemanticTerm / Region combination.
type SourcePreference struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	BusinessObject string         `json:"business_object"`
	SemanticTerm   string         `json:"semantic_term"`
	Region         string         `json:"region"`
	Priority       int            `json:"priority"` // 1=first, 2=second, 3=third
	SourceSystem   string         `json:"source_system"`
	Confidence     int            `json:"confidence"` // 0-100
	Status         string         `json:"status"`     // draft | testing | staging | production
	Version        int            `json:"version"`
	CoreID         *uuid.UUID     `json:"core_id,omitempty"`         // nil => this IS the core record
	OverrideReason *string        `json:"override_reason,omitempty"` // present on tenant overrides
	ValidFrom      time.Time      `json:"valid_from"`
	ValidTo        *time.Time     `json:"valid_to,omitempty"`
	ImpactAnalysis ImpactAnalysis `json:"impact_analysis"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	CreatedBy      uuid.UUID      `json:"created_by"`
	UpdatedBy      *uuid.UUID     `json:"updated_by,omitempty"`
}

// CanOverride returns true when a core production preference may be overridden
func (p *SourcePreference) CanOverride() bool {
	return p.Status == "production"
}

// CreateOverrideRequest derives a new draft preference that overrides this record
func (p *SourcePreference) CreateOverrideRequest(managerID uuid.UUID, reason string, validTo time.Time) (*SourcePreference, error) {
	if !p.CanOverride() {
		return nil, errors.New("preference must be in production status before it can be overridden")
	}
	override := *p
	override.ID = uuid.New()
	override.CoreID = &p.ID
	override.Status = "draft"
	override.Version = 1
	override.CreatedBy = managerID
	override.UpdatedBy = nil
	override.OverrideReason = &reason
	override.ValidTo = &validTo
	override.ValidFrom = time.Now()
	override.CreatedAt = time.Now()
	override.UpdatedAt = time.Now()
	override.ImpactAnalysis = ImpactAnalysis{}
	return &override, nil
}

// ImpactAnalysisJSON marshals ImpactAnalysis to []byte for DB storage
func (ia ImpactAnalysis) JSON() ([]byte, error) {
	return json.Marshal(ia)
}

// determineBusinessImpact categorises delta into a human-readable level
func DetermineBusinessImpact(delta int) string {
	switch {
	case delta == 0:
		return "none"
	case delta <= 5:
		return "low"
	case delta <= 15:
		return "moderate"
	default:
		return "high"
	}
}

// SourceException represents a data-source conflict or quality issue
type SourceException struct {
	ID                uuid.UUID                `json:"id"`
	TenantID          uuid.UUID                `json:"tenant_id"`
	BusinessObject    string                   `json:"business_object"`
	SemanticTerm      string                   `json:"semantic_term"`
	Region            string                   `json:"region"`
	SourceSystem      string                   `json:"source_system"`
	ExceptionType     string                   `json:"exception_type"` // SOURCE_CONFLICT, DATA_QUALITY, SYSTEM_ERROR, COMPLIANCE_VIOLATION
	Description       string                   `json:"description"`
	ImpactLevel       int                      `json:"impact_level"` // 1-5
	CriticalPath      bool                     `json:"critical_path"`
	Status            string                   `json:"status"` // open | in_progress | resolved
	Metadata          map[string]interface{}   `json:"metadata"`
	ResolutionHistory []map[string]interface{} `json:"resolution_history"`
	CreatedAt         time.Time                `json:"created_at"`
	ResolvedAt        *time.Time               `json:"resolved_at,omitempty"`
	ResolvedBy        *uuid.UUID               `json:"resolved_by,omitempty"`
}

// IsCritical reports whether this exception must be handled synchronously
func (e *SourceException) IsCritical() bool {
	switch e.ExceptionType {
	case "SOURCE_CONFLICT":
		return e.ImpactLevel >= 3
	case "DATA_QUALITY":
		return e.ImpactLevel >= 4
	case "SYSTEM_ERROR", "COMPLIANCE_VIOLATION":
		return true
	default:
		return false
	}
}
