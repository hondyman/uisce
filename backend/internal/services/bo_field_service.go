package services

import (
	"context"
	"fmt"
	"os"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// ============================================================================
// BUSINESS OBJECT FIELD MANAGEMENT WITH INHERITANCE
// ============================================================================

// BusinessObjectFieldService manages BO fields with Workday-style inheritance
type BusinessObjectFieldService struct {
	hasuraClient *hasuraclient.HasuraClient
	logger       *zap.Logger
	isAdminCore  bool // When true, new fields are core enhancements, not customizations
}

// NewBusinessObjectFieldService creates a new BO field service
func NewBusinessObjectFieldService(hasuraClient *hasuraclient.HasuraClient) *BusinessObjectFieldService {
	logger, _ := zap.NewProduction()

	// Check ADMIN_CORE environment variable
	isAdminCore := os.Getenv("ADMIN_CORE") == "true"

	return &BusinessObjectFieldService{
		hasuraClient: hasuraClient,
		logger:       logger,
		isAdminCore:  isAdminCore,
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
	// Determine if this field is custom based on ADMIN_CORE env var
	isCustom := !s.isAdminCore

	mutation := `
		mutation InsertBOField($object: bo_fields_insert_input!) {
			insert_bo_fields_one(object: $object) {
				id
			}
		}
	`

	object := map[string]interface{}{
		"tenant_id":        tenantID,
		"key":              input.Key,
		"name":             input.Name,
		"display_name":     input.DisplayName,
		"technical_name":   input.TechnicalName,
		"type":             input.Type,
		"is_core":          !isCustom, // Core if ADMIN_CORE=true
		"is_custom":        isCustom,
		"is_required":      input.IsRequired,
		"role":             input.Role,
		"semantic_term_id": input.SemanticTermID,
		"sequence":         input.Sequence,
	}

	if input.BusinessObjectID != "" {
		object["business_object_id"] = input.BusinessObjectID
	}
	if input.SubtypeID != "" {
		object["subtype_id"] = input.SubtypeID
	}
	if input.Description != "" {
		object["description"] = input.Description
	}
	if input.ReferenceEntity != "" {
		object["reference_entity"] = input.ReferenceEntity
	}

	result, err := s.hasuraClient.Mutate(mutation, map[string]interface{}{
		"object": object,
	})

	if err != nil {
		s.logger.Error("Failed to add field", zap.Error(err))
		return "", err
	}

	data := result["insert_bo_fields_one"].(map[string]interface{})
	fieldID := data["id"].(string)

	s.logger.Info("Field added",
		zap.String("field_id", fieldID),
		zap.String("key", input.Key),
		zap.Bool("is_custom", isCustom),
		zap.Bool("admin_core_mode", s.isAdminCore))

	return fieldID, nil
}

// GetAllFields gets all fields for a BO including inherited fields
func (s *BusinessObjectFieldService) GetAllFields(ctx context.Context, businessObjectID string) ([]*BOField, error) {
	// Use the database function to get all fields (inherited + own)
	query := `
		query GetAllBOFields($boID: uuid!) {
			get_all_bo_fields(args: {bo_id: $boID}) {
				field_id
				field_key
				field_name
				field_type
				is_core
				is_required
				is_custom
				is_inherited
				inherited_from
				role
				semantic_term_id
				sequence
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"boID": businessObjectID,
	})

	if err != nil {
		return nil, err
	}

	items, ok := result["get_all_bo_fields"].([]interface{})
	if !ok {
		return []*BOField{}, nil
	}

	fields := make([]*BOField, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		field := &BOField{
			ID:             getString(data, "field_id"),
			Key:            getString(data, "field_key"),
			Name:           getString(data, "field_name"),
			Type:           getString(data, "field_type"),
			IsCore:         getBool(data, "is_core"),
			IsCustom:       getBool(data, "is_custom"),
			IsInherited:    getBool(data, "is_inherited"),
			IsRequired:     getBool(data, "is_required"),
			Role:           getString(data, "role"),
			SemanticTermID: getString(data, "semantic_term_id"),
			Sequence:       getInt(data, "sequence"),
		}

		if inheritedFrom, ok := data["inherited_from"].(string); ok && inheritedFrom != "" {
			field.InheritedFromKey = inheritedFrom
		}

		fields = append(fields, field)
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
	query := `
		query GetParentBO($boID: uuid!) {
			business_objects_by_pk(id: $boID) {
				parent_bo_id
				parent_bo {
					id
					key
					display_name
				}
			}
		}
	`

	result, err := s.hasuraClient.Query(query, map[string]interface{}{
		"boID": businessObjectID,
	})

	if err != nil {
		return nil, err
	}

	bo, ok := result["business_objects_by_pk"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("business object not found")
	}

	if parentBO, ok := bo["parent_bo"].(map[string]interface{}); ok {
		return parentBO, nil
	}

	return nil, nil
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
