package analytics

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// BOStatusService determines the status of Business Objects
type BOStatusService struct {
	db *sqlx.DB
}

// NewBOStatusService creates a new status service
func NewBOStatusService(db *sqlx.DB) *BOStatusService {
	return &BOStatusService{db: db}
}

// BOStatus represents the current state of a Business Object
type BOStatus struct {
	Status              string            `json:"status"`
	Reason              string            `json:"reason"`
	PendingTerms        []string          `json:"pending_terms"`
	PendingCalculations []string          `json:"pending_calculations"`
	PendingDependencies []DependencyIssue `json:"pending_dependencies"`
	DiffRequired        bool              `json:"diff_required"`
	ImportPending       bool              `json:"import_pending"`
	ValidationErrors    []ValidationError `json:"validation_errors"`
	LastModified        time.Time         `json:"last_modified"`
	ModifiedBy          string            `json:"modified_by"`
	Version             string            `json:"version"`
	IsPublished         bool              `json:"is_published"`
	CanPublish          bool              `json:"can_publish"`
}

// DependencyIssue represents a dependency that blocks publishing
type DependencyIssue struct {
	Type          string `json:"type"` // "term", "calculation", "relationship"
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	BlocksPublish bool   `json:"blocks_publish"`
}

// ValidationError represents a validation issue
type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// GetBOStatus determines the current status of a BO
func (s *BOStatusService) GetBOStatus(boID string) (*BOStatus, error) {
	status := &BOStatus{
		PendingTerms:        []string{},
		PendingCalculations: []string{},
		PendingDependencies: []DependencyIssue{},
		ValidationErrors:    []ValidationError{},
	}

	// 1. Check if BO exists and get metadata
	var isPublished sql.NullBool
	var lastModified sql.NullTime
	var modifiedBy sql.NullString
	var version sql.NullString

	// Try business_objects table first (wizard-created BOs)
	err := s.db.QueryRow(`
		SELECT 
			is_active as is_published,
			last_modified_at,
			COALESCE(CAST(last_modified_by AS text), '') as modified_by,
			'v1' as version
		FROM business_objects
		WHERE id = $1::uuid
	`, boID).Scan(&isPublished, &lastModified, &modifiedBy, &version)

	if err != nil {
		// Fallback to catalog_node (legacy BOs)
		err = s.db.QueryRow(`
			SELECT 
				COALESCE((properties->>'is_published')::boolean, false) as is_published,
				updated_at,
				COALESCE(properties->>'modified_by', '') as modified_by,
				COALESCE(properties->>'version', 'v1') as version
			FROM catalog_node
			WHERE id = $1
		`, boID).Scan(&isPublished, &lastModified, &modifiedBy, &version)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch BO: %w", err)
		}
	}

	status.IsPublished = isPublished.Bool
	if lastModified.Valid {
		status.LastModified = lastModified.Time
	}
	if modifiedBy.Valid {
		status.ModifiedBy = modifiedBy.String
	}
	if version.Valid {
		status.Version = version.String
	}

	// 2. Check for pending terms (unapproved)
	pendingTerms, err := s.getPendingTerms(boID)
	if err == nil {
		status.PendingTerms = pendingTerms
	}

	// 3. Check for pending calculations (unapproved)
	pendingCalcs, err := s.getPendingCalculations(boID)
	if err == nil {
		status.PendingCalculations = pendingCalcs
	}

	// 4. Check for dependency issues
	deps, err := s.checkDependencies(boID)
	if err == nil {
		status.PendingDependencies = deps
	}

	// 5. Run validation
	errors := s.validateBO(boID)
	status.ValidationErrors = errors

	// 6. Check for pending import (stored in properties)
	status.ImportPending = s.hasImportPending(boID)

	// 7. Check for diff requirements
	status.DiffRequired = s.requiresDiffResolution(boID)

	// 8. Determine overall status
	status.Status = s.determineStatus(status)
	status.Reason = s.generateReason(status)
	status.CanPublish = s.canPublish(status)

	return status, nil
}

// getPendingTerms finds terms awaiting approval
func (s *BOStatusService) getPendingTerms(boID string) ([]string, error) {
	query := `
		SELECT cn.node_name
		FROM catalog_edge ce
		JOIN catalog_node cn ON ce.target_node_id = cn.id
		WHERE 
			ce.source_node_id = $1
			AND ce.edge_type = 'HAS_ATTRIBUTE'
			AND COALESCE((cn.properties->>'approved')::boolean, false) = false
	`

	var terms []string
	err := s.db.Select(&terms, query, boID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return terms, nil
}

// getPendingCalculations finds calculations awaiting approval
func (s *BOStatusService) getPendingCalculations(boID string) ([]string, error) {
	query := `
		SELECT cn.node_name
		FROM catalog_edge ce
		JOIN catalog_node cn ON ce.target_node_id = cn.id
		WHERE 
			ce.source_node_id = $1
			AND ce.edge_type = 'BO_HAS_CALC'
			AND COALESCE((cn.properties->>'approved')::boolean, false) = false
	`

	var calcs []string
	err := s.db.Select(&calcs, query, boID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return calcs, nil
}

// checkDependencies validates all dependencies
func (s *BOStatusService) checkDependencies(boID string) ([]DependencyIssue, error) {
	var issues []DependencyIssue

	// Check if any terms have missing physical mappings
	query := `
		SELECT 
			cn.id,
			cn.node_name,
			cn.node_type
		FROM catalog_edge ce
		JOIN catalog_node cn ON ce.target_node_id = cn.id
		WHERE 
			ce.source_node_id = $1
			AND ce.edge_type = 'HAS_ATTRIBUTE'
			AND (cn.properties->'physical_mapping' IS NULL 
			     OR cn.properties->'physical_mapping'->>'column' IS NULL)
	`

	rows, err := s.db.Query(query, boID)
	if err != nil {
		return issues, nil // Non-fatal
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, nodeType string
		if err := rows.Scan(&id, &name, &nodeType); err != nil {
			continue
		}

		issues = append(issues, DependencyIssue{
			Type:          "term",
			ID:            id,
			Name:          name,
			Status:        "missing_mapping",
			BlocksPublish: true,
		})
	}

	return issues, nil
}

// validateBO runs validation rules
func (s *BOStatusService) validateBO(boID string) []ValidationError {
	var errors []ValidationError

	// 1. Check for required fields
	var name, drivingTable sql.NullString

	// Try business_objects table first (wizard-created BOs)
	err := s.db.QueryRow(`
		SELECT 
			COALESCE(display_name, name) as name,
			driver_table_id as driving_table
		FROM business_objects
		WHERE id = $1::uuid
	`, boID).Scan(&name, &drivingTable)

	if err != nil {
		// Fallback to catalog_node (legacy BOs)
		err = s.db.QueryRow(`
			SELECT 
				node_name,
				properties->>'driving_table'
			FROM catalog_node
			WHERE id = $1
		`, boID).Scan(&name, &drivingTable)

		if err != nil {
			errors = append(errors, ValidationError{
				Field:    "bo",
				Message:  "Failed to fetch BO metadata",
				Severity: "error",
			})
			return errors
		}
	}

	if !name.Valid || name.String == "" {
		errors = append(errors, ValidationError{
			Field:    "name",
			Message:  "BO name is required",
			Severity: "error",
		})
	}

	if !drivingTable.Valid || drivingTable.String == "" {
		errors = append(errors, ValidationError{
			Field:    "driving_table",
			Message:  "Driving table is required for SQL generation",
			Severity: "warning",
		})
	}

	// 2. Check for at least one term (check persistence layer, not graph)
	var termCount int
	s.db.QueryRow(`
		SELECT COUNT(*)
		FROM bo_fields
		WHERE business_object_id = $1::uuid
	`, boID).Scan(&termCount)

	if termCount == 0 {
		errors = append(errors, ValidationError{
			Field:    "terms",
			Message:  "BO must have at least one term",
			Severity: "error",
		})
	}

	return errors
}

// hasImportPending checks if BO has a pending import
func (s *BOStatusService) hasImportPending(boID string) bool {
	var importPending sql.NullBool
	s.db.QueryRow(`
		SELECT COALESCE((properties->>'import_pending')::boolean, false)
		FROM catalog_node
		WHERE id = $1
	`, boID).Scan(&importPending)

	return importPending.Valid && importPending.Bool
}

// requiresDiffResolution checks if BO needs diff resolution
func (s *BOStatusService) requiresDiffResolution(boID string) bool {
	var diffRequired sql.NullBool
	s.db.QueryRow(`
		SELECT COALESCE((properties->>'diff_required')::boolean, false)
		FROM catalog_node
		WHERE id = $1
	`, boID).Scan(&diffRequired)

	return diffRequired.Valid && diffRequired.Bool
}

// determineStatus calculates the overall status
func (s *BOStatusService) determineStatus(status *BOStatus) string {
	// Priority order (highest to lowest)
	hasErrors := false
	for _, err := range status.ValidationErrors {
		if err.Severity == "error" {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		return "error"
	}
	if status.ImportPending {
		return "pending_import"
	}
	if status.DiffRequired {
		return "pending_diff_resolution"
	}
	if len(status.PendingDependencies) > 0 {
		return "pending_dependencies"
	}
	if len(status.PendingTerms) > 0 || len(status.PendingCalculations) > 0 {
		return "pending_review"
	}
	if !status.IsPublished && status.CanPublish {
		return "pending_publish"
	}
	if !status.IsPublished {
		return "draft"
	}

	return "published"
}

// generateReason creates a human-readable reason
func (s *BOStatusService) generateReason(status *BOStatus) string {
	switch status.Status {
	case "error":
		errorCount := 0
		for _, err := range status.ValidationErrors {
			if err.Severity == "error" {
				errorCount++
			}
		}
		return fmt.Sprintf("%d validation errors must be resolved", errorCount)
	case "pending_import":
		return "This BO was imported and requires confirmation"
	case "pending_diff_resolution":
		return "This BO has unresolved differences compared to another environment"
	case "pending_dependencies":
		return fmt.Sprintf("%d dependencies require approval", len(status.PendingDependencies))
	case "pending_review":
		count := len(status.PendingTerms) + len(status.PendingCalculations)
		return fmt.Sprintf("%d terms/calculations require approval", count)
	case "pending_publish":
		return "This BO is ready to publish"
	case "draft":
		return "This Business Object is in draft state"
	default:
		return "This BO is published and ready to use"
	}
}

// canPublish determines if BO can be published
func (s *BOStatusService) canPublish(status *BOStatus) bool {
	// Cannot publish if there are errors
	for _, err := range status.ValidationErrors {
		if err.Severity == "error" {
			return false
		}
	}

	// Cannot publish if dependencies are blocking
	for _, dep := range status.PendingDependencies {
		if dep.BlocksPublish {
			return false
		}
	}

	// Cannot publish if import is pending
	if status.ImportPending {
		return false
	}

	return true
}
