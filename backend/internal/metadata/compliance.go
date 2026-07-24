package metadata

import (
	"time"

	"github.com/lib/pq"
)

// Compliance constants
const (
	ResidencyUS     = "US"
	ResidencyEU     = "EU"
	ResidencyGlobal = "GLOBAL"

	SensitivityLow    = "LOW"
	SensitivityMedium = "MEDIUM"
	SensitivityHigh   = "HIGH"
)

// BusinessTerm represents a governed business term
type BusinessTerm struct {
	ID               string         `json:"id" db:"id"`
	TenantID         string         `json:"tenant_id" db:"tenant_id"`
	Name             string         `json:"name" db:"name"`
	Description      string         `json:"description" db:"description"`
	PIIFlag          bool           `json:"pii_flag" db:"pii_flag"`
	Residency        string         `json:"residency" db:"residency"`                 // US, EU, GLOBAL
	SensitivityLevel string         `json:"sensitivity_level" db:"sensitivity_level"` // LOW, MEDIUM, HIGH
	SemanticTermIDs  pq.StringArray `json:"semantic_term_ids" db:"semantic_term_ids"` // Linked semantic terms (ids in catalog_node)
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	CreatedBy        string         `json:"created_by" db:"created_by"`
	UpdatedBy        string         `json:"updated_by" db:"updated_by"`
}

// SemanticTermCompliance represents the compliance properties inherited by a semantic term
type SemanticTermCompliance struct {
	InheritedPIIFlag     bool   `json:"inherited_pii_flag"`
	InheritedResidency   string `json:"inherited_residency"`
	InheritedSensitivity string `json:"inherited_sensitivity"`
	BusinessTermID       string `json:"business_term_id"`
}

// UpdateBusinessTermRequest defines the payload for updating a business term
type UpdateBusinessTermRequest struct {
	Name             *string   `json:"name"`
	Description      *string   `json:"description"`
	PIIFlag          *bool     `json:"pii_flag"`
	Residency        *string   `json:"residency"`
	SensitivityLevel *string   `json:"sensitivity_level"`
	SemanticTermIDs  *[]string `json:"semantic_term_ids"`
}
