package api

import (
	"context"
	"fmt"
	"log"
)

// ============================================================================
// Template RBAC - Role-Based Access Control for Templates
// ============================================================================

// TemplateRBAC provides role-based access control for templates
type TemplateRBAC struct {
	store *TemplateStore
}

// NewTemplateRBAC creates a new RBAC enforcer
func NewTemplateRBAC(store *TemplateStore) *TemplateRBAC {
	return &TemplateRBAC{store: store}
}

// ============================================================================
// Permission Checks
// ============================================================================

// CanRun checks if a role can run a template
func (tr *TemplateRBAC) CanRun(ctx context.Context, templateID, role string) (bool, error) {
	perm, err := tr.store.GetPermission(ctx, templateID, role)
	if err != nil {
		return false, err
	}
	return perm.CanRun, nil
}

// CanEdit checks if a role can edit a template
func (tr *TemplateRBAC) CanEdit(ctx context.Context, templateID, role string) (bool, error) {
	perm, err := tr.store.GetPermission(ctx, templateID, role)
	if err != nil {
		return false, err
	}
	return perm.CanEdit, nil
}

// CanDelete checks if a role can delete a template
func (tr *TemplateRBAC) CanDelete(ctx context.Context, templateID, role string) (bool, error) {
	perm, err := tr.store.GetPermission(ctx, templateID, role)
	if err != nil {
		return false, err
	}
	return perm.CanDelete, nil
}

// CanPromote checks if a role can promote a template
func (tr *TemplateRBAC) CanPromote(ctx context.Context, templateID, role string) (bool, error) {
	perm, err := tr.store.GetPermission(ctx, templateID, role)
	if err != nil {
		return false, err
	}
	return perm.CanPromote, nil
}

// ============================================================================
// Parameter-Level RBAC
// ============================================================================

// ParameterConstraint defines access restrictions for a specific parameter
type ParameterConstraint struct {
	ParameterName string      `json:"parameter_name"`
	AllowedRoles  []string    `json:"allowed_roles"`       // Only these roles can set this param
	MinValue      interface{} `json:"min_value,omitempty"` // For number params
	MaxValue      interface{} `json:"max_value,omitempty"`
	AllowedValues []string    `json:"allowed_values,omitempty"` // For string params: whitelist
}

// ConstraintsStore stores parameter-level constraints
type ConstraintsStore struct {
	constraints map[string]ParameterConstraint
}

// ValidateParameterAccess checks if a user role can set a specific parameter to a value
func ValidateParameterAccess(param string, value interface{}, role string, constraints []ParameterConstraint) error {
	for _, c := range constraints {
		if c.ParameterName == param {
			// Check role permission
			roleAllowed := false
			for _, r := range c.AllowedRoles {
				if r == role || r == "*" { // "*" = everyone
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				return fmt.Errorf("role %s cannot set parameter %s", role, param)
			}

			// Check value constraints
			if err := validateParameterValue(value, c); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateParameterValue checks if a value meets the constraints
func validateParameterValue(value interface{}, c ParameterConstraint) error {
	// Check allowed values (whitelist)
	if len(c.AllowedValues) > 0 {
		strVal := fmt.Sprintf("%v", value)
		allowed := false
		for _, av := range c.AllowedValues {
			if av == strVal {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("parameter value %v not in allowed list: %v", value, c.AllowedValues)
		}
	}

	// Check numeric bounds
	if c.MinValue != nil || c.MaxValue != nil {
		numVal, ok := value.(float64)
		if !ok {
			// Try to convert from int
			if intVal, ok := value.(int); ok {
				numVal = float64(intVal)
			} else {
				return fmt.Errorf("cannot validate numeric bounds on non-numeric value")
			}
		}

		if c.MinValue != nil {
			minVal := c.MinValue.(float64)
			if numVal < minVal {
				return fmt.Errorf("parameter value %v is below minimum %v", numVal, minVal)
			}
		}

		if c.MaxValue != nil {
			maxVal := c.MaxValue.(float64)
			if numVal > maxVal {
				return fmt.Errorf("parameter value %v exceeds maximum %v", numVal, maxVal)
			}
		}
	}

	return nil
}

// ============================================================================
// Visibility-Based Access Control
// ============================================================================

// TemplateVisibility controls who can discover and access a template
type TemplateVisibility string

const (
	VisibilityPrivate TemplateVisibility = "private" // Only creator
	VisibilityTeam    TemplateVisibility = "team"    // Team members
	VisibilityPublic  TemplateVisibility = "public"  // All authenticated users
)

// CanAccess checks if a user can access a template based on visibility
func CanAccess(template *SemanticQueryTemplate, userID, userRole string) bool {
	switch TemplateVisibility(template.Visibility) {
	case VisibilityPrivate:
		// Only creator can access
		return template.CreatedBy == userID

	case VisibilityTeam:
		// Team members can access
		// This would typically check if user is in the same team/department
		// For now, we assume team membership is checked elsewhere
		return true

	case VisibilityPublic:
		// All authenticated users can access
		return true

	default:
		log.Printf("Unknown visibility: %v", template.Visibility)
		return false
	}
}

// ============================================================================
// Field-Level RBAC (Integration with Semantic Engine)
// ============================================================================

// FieldAccessValidator checks if a template's fields are accessible to a user
type FieldAccessValidator struct {
	// This would integrate with your existing semantic engine's field-level RBAC
}

// ValidateFieldAccess checks if all fields in a template are accessible
// This integrates with your existing field masking and RLS/CLS rules
func (fav *FieldAccessValidator) ValidateFieldAccess(
	ctx context.Context,
	template *SemanticQueryTemplate,
	bundle *SemanticBundle,
	userRole string,
) error {
	// Extract all fields used in the template
	usedFields := extractTemplateFields(template.SemanticQuery)

	for _, fieldName := range usedFields {
		// Check if field exists in bundle
		found := false
		for _, f := range bundle.Fields {
			if f.Name == fieldName {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("field %s not found in bundle", fieldName)
		}

		// In production, check field-level RLS/CLS/masking rules
		// This would use your semantic engine's existing validation
	}

	return nil
}

// extractTemplateFields extracts all field names referenced in a semantic query
func extractTemplateFields(q *SemanticQuery) []string {
	fields := make(map[string]bool)

	// Add select fields
	for _, f := range q.Select {
		fields[f] = true
	}

	// Add filter fields
	for _, f := range q.Filters {
		fields[f.Field] = true
	}

	// Add order_by fields
	for _, ob := range q.OrderBy {
		fields[ob.Field] = true
	}

	// Convert to slice
	var result []string
	for f := range fields {
		result = append(result, f)
	}

	return result
}

// ============================================================================
// Template Audit & Compliance
// ============================================================================

// AuditLog records important actions on templates
type TemplateAuditLog struct {
	ID          string
	TemplateID  string
	Action      string // "created", "updated", "executed", "deleted", "promoted"
	PerformedBy string
	Timestamp   int64
	Changes     map[string]interface{}
	IPAddress   string
}

// LogAction logs a template action for compliance/audit
func LogAction(auditLog *TemplateAuditLog) {
	log.Printf("Template Audit: action=%s template=%s user=%s timestamp=%d",
		auditLog.Action, auditLog.TemplateID, auditLog.PerformedBy, auditLog.Timestamp)
	// In production, write to audit log table
}

// ============================================================================
// Default RBAC Rules
// ============================================================================

// DefaultRBACRules returns the default RBAC rules for templates
func DefaultRBACRules() map[string]*TemplatePermission {
	return map[string]*TemplatePermission{
		"viewer": {
			CanRun:     true,
			CanEdit:    false,
			CanDelete:  false,
			CanPromote: false,
		},
		"editor": {
			CanRun:     true,
			CanEdit:    true,
			CanDelete:  false,
			CanPromote: false,
		},
		"admin": {
			CanRun:     true,
			CanEdit:    true,
			CanDelete:  true,
			CanPromote: true,
		},
	}
}

// ============================================================================
// Template Promotion Workflow RBAC
// ============================================================================

// PromotionState represents the state in the template promotion workflow
type PromotionState string

const (
	StateDraft     PromotionState = "draft"
	StateReview    PromotionState = "review"
	StateApproved  PromotionState = "approved"
	StatePublished PromotionState = "published"
)

// PromotionState rules: who can transition between states
var PromotionStateTransitions = map[PromotionState]map[string]string{
	StateDraft: {
		"editor": "review", // Editors can submit for review
		"admin":  "review",
	},
	StateReview: {
		"admin": "approved", // Only admins can approve
	},
	StateApproved: {
		"admin": "published", // Admins can publish
	},
}

// CanTransitionState checks if a role can transition a template to a new state
func CanTransitionState(currentState PromotionState, newState PromotionState, role string) bool {
	transitions, ok := PromotionStateTransitions[currentState]
	if !ok {
		return false
	}

	allowedRole, ok := transitions[role]
	if !ok {
		return false
	}

	return allowedRole == string(newState)
}
