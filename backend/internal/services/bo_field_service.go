package services

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ============================================================================
// BUSINESS OBJECT FIELD MANAGEMENT WITH INHERITANCE
// ============================================================================

// BusinessObjectFieldService manages BO fields with Workday-style inheritance
type BusinessObjectFieldService struct {
	db          *sqlx.DB
	logger      *zap.Logger
	isAdminCore bool // When true, new fields are core enhancements, not customizations
}

// NewBusinessObjectFieldService creates a new BO field service
func NewBusinessObjectFieldService(db *sqlx.DB) *BusinessObjectFieldService {
	logger, _ := zap.NewProduction()

	// Check ADMIN_CORE environment variable
	isAdminCore := os.Getenv("ADMIN_CORE") == "true"

	return &BusinessObjectFieldService{
		db:          db,
		logger:      logger,
		isAdminCore: isAdminCore,
	}
}

// BOFieldInput represents input for creating/updating a field
type BOFieldInput struct {
	BusinessObjectID string `json:"business_object_id"`
	SubtypeID        string `json:"subtype_id,omitempty"`
	Key              string `json:"key"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	TechnicalName    string `json:"technical_name"`
	Type             string `json:"type"`
	IsRequired       bool   `json:"is_required"`
	Description      string `json:"description,omitempty"`
	Role             string `json:"role,omitempty"`
	SemanticTermID   string `json:"semantic_term_id,omitempty"`
	ReferenceEntity  string `json:"reference_entity,omitempty"`
	Sequence         int    `json:"sequence"`
}

// BOField represents a business object field with metadata
type BOField struct {
	ID               string `json:"id"`
	Key              string `json:"key"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	Type             string `json:"type"`
	IsCore           bool   `json:"is_core"`
	IsCustom         bool   `json:"is_custom"`
	IsInherited      bool   `json:"is_inherited"`
	IsRequired       bool   `json:"is_required"`
	Role             string `json:"role,omitempty"`
	SemanticTermID   string `json:"semantic_term_id,omitempty"`
	InheritedFrom    string `json:"inherited_from,omitempty"`
	InheritedFromKey string `json:"inherited_from_key,omitempty"`
	Sequence         int    `json:"sequence"`
}

// AddField adds a new field to a business object
func (s *BusinessObjectFieldService) AddField(ctx context.Context, tenantID string, input BOFieldInput) (string, error) {
	isCustom := !s.isAdminCore

	fieldID := uuid.New().String()

	var subtypeIDArg interface{} = nil
	if input.SubtypeID != "" {
		subtypeIDArg = input.SubtypeID
	}
	var boIDArg interface{} = nil
	if input.BusinessObjectID != "" {
		boIDArg = input.BusinessObjectID
	}
	var descArg interface{} = nil
	if input.Description != "" {
		descArg = input.Description
	}
	var roleArg interface{} = nil
	if input.Role != "" {
		roleArg = input.Role
	}
	var semTermArg interface{} = nil
	if input.SemanticTermID != "" {
		semTermArg = input.SemanticTermID
	}
	var refEntityArg interface{} = nil
	if input.ReferenceEntity != "" {
		refEntityArg = input.ReferenceEntity
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bo_fields (
			id, tenant_id, business_object_id, subtype_id, key, name,
			display_name, technical_name, type, is_core, is_custom, is_required,
			role, semantic_term_id, reference_entity, description, sequence,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			NOW(), NOW()
		)
	`, fieldID, tenantID, boIDArg, subtypeIDArg, input.Key, input.Name,
		input.DisplayName, input.TechnicalName, input.Type, !isCustom, isCustom, input.IsRequired,
		roleArg, semTermArg, refEntityArg, descArg, input.Sequence)

	if err != nil {
		s.logger.Error("Failed to add field", zap.Error(err))
		return "", err
	}

	s.logger.Info("Field added",
		zap.String("field_id", fieldID),
		zap.String("key", input.Key),
		zap.Bool("is_custom", isCustom),
		zap.Bool("admin_core_mode", s.isAdminCore))

	return fieldID, nil
}

// GetAllFields gets all fields for a BO including inherited fields
func (s *BusinessObjectFieldService) GetAllFields(ctx context.Context, businessObjectID string) ([]*BOField, error) {
	type row struct {
		ID             string `db:"id"`
		Key            string `db:"key"`
		Name           string `db:"name"`
		Type           string `db:"type"`
		IsCore         bool   `db:"is_core"`
		IsCustom       bool   `db:"is_custom"`
		IsInherited    bool   `db:"is_inherited"`
		IsRequired     bool   `db:"is_required"`
		Role           string `db:"role"`
		SemanticTermID string `db:"semantic_term_id"`
		Sequence       int    `db:"sequence"`
	}

	var rows []row
	// Own fields
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, key, name, type,
		       COALESCE(is_core, false) as is_core,
		       COALESCE(is_custom, false) as is_custom,
		       false as is_inherited,
		       COALESCE(is_required, false) as is_required,
		       COALESCE(role, '') as role,
		       COALESCE(semantic_term_id::text, '') as semantic_term_id,
		       COALESCE(sequence, 0) as sequence
		FROM bo_fields
		WHERE business_object_id = $1
		ORDER BY sequence
	`, businessObjectID)

	if err != nil {
		return nil, err
	}

	fields := make([]*BOField, 0, len(rows))
	for _, r := range rows {
		fields = append(fields, &BOField{
			ID:             r.ID,
			Key:            r.Key,
			Name:           r.Name,
			Type:           r.Type,
			IsCore:         r.IsCore,
			IsCustom:       r.IsCustom,
			IsInherited:    r.IsInherited,
			IsRequired:     r.IsRequired,
			Role:           r.Role,
			SemanticTermID: r.SemanticTermID,
			Sequence:       r.Sequence,
		})
	}
	return fields, nil
}

// GetFieldsByCategory returns fields grouped by category
func (s *BusinessObjectFieldService) GetFieldsByCategory(ctx context.Context, businessObjectID string) (map[string][]*BOField, error) {
	allFields, err := s.GetAllFields(ctx, businessObjectID)
	if err != nil {
		return nil, err
	}

	categorized := map[string][]*BOField{
		"inherited": {},
		"core":      {},
		"custom":    {},
	}

	for _, field := range allFields {
		if field.IsInherited {
			categorized["inherited"] = append(categorized["inherited"], field)
		} else if field.IsCustom {
			categorized["custom"] = append(categorized["custom"], field)
		} else {
			categorized["core"] = append(categorized["core"], field)
		}
	}

	return categorized, nil
}

// GetParentBO gets the parent business object
func (s *BusinessObjectFieldService) GetParentBO(ctx context.Context, businessObjectID string) (map[string]interface{}, error) {
	var parentID string
	err := s.db.GetContext(ctx, &parentID, `
		SELECT COALESCE(parent_bo_id::text, '') FROM business_objects WHERE id = $1
	`, businessObjectID)

	if err != nil || parentID == "" {
		return nil, fmt.Errorf("business object not found or no parent")
	}

	type parentRow struct {
		ID          string `db:"id"`
		Key         string `db:"key"`
		DisplayName string `db:"display_name"`
	}
	var p parentRow
	err = s.db.GetContext(ctx, &p, `
		SELECT id, key, COALESCE(display_name, '') as display_name
		FROM business_objects WHERE id = $1
	`, parentID)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":           p.ID,
		"key":          p.Key,
		"display_name": p.DisplayName,
	}, nil
}

// Helper functions
func getInt(data map[string]interface{}, key string) int {
	if v, ok := data[key].(float64); ok {
		return int(v)
	}
	if v, ok := data[key].(int); ok {
		return v
	}
	return 0
}
