package calculation

import "github.com/google/uuid"

// BOField represents a column or calculated attribute inside a resolved Business Object.
type BOField struct {
	ID                uuid.UUID  `json:"id"`
	BOID              uuid.UUID  `json:"bo_id"`
	FieldName         string     `json:"field_name"`
	SemanticTermID    uuid.UUID  `json:"semantic_term_id"`
	FieldRole         string     `json:"field_role"`          // KEY, DIMENSION, MEASURE, ATTRIBUTE
	RequirementStatus string     `json:"binding_requirement"` // REQUIRED, OPTIONAL, BACKEND_SPECIFIC
	BindingStatus     string     `json:"binding_status"`      // RESOLVED, PARTIAL, UNRESOLVED
	EligibilitySource string     `json:"eligibility_source"`  // DIRECT, INHERITED, OVERRIDE
	ParentFieldID     *uuid.UUID `json:"parent_field_id,omitempty"`
}

// BusinessObject represents the semantic layer shell contract.
type BusinessObject struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	ModelID       uuid.UUID  `json:"model_id"`
	BOKey         string     `json:"bo_key"`
	BOName        string     `json:"bo_name"`
	ParentBOID    *uuid.UUID `json:"parent_bo_id,omitempty"`
	DriverTableID uuid.UUID  `json:"driver_table_id"`
	Fields        []BOField  `json:"fields"`
}
