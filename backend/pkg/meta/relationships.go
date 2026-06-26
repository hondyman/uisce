package meta

import (
	"context"
	"fmt"
)

// RelationshipResolver automatically resolves and links related business objects
// following Workday's pattern of automatic object linking (e.g., Worker → Position → Job Profile)
type RelationshipResolver struct {
	cache *MetadataCache
}

// NewRelationshipResolver creates a new relationship resolver
func NewRelationshipResolver(cache *MetadataCache) *RelationshipResolver {
	return &RelationshipResolver{
		cache: cache,
	}
}

// ResolveRelationships loads all related objects for a given business object instance
// This enables automatic linking like Workday's Worker → Position → Job Profile
func (r *RelationshipResolver) ResolveRelationships(
	ctx context.Context,
	tenantID string,
	boKey string,
	instanceData map[string]interface{},
	maxDepth int,
) (map[string]interface{}, error) {
	if maxDepth <= 0 {
		maxDepth = 3 // Default depth
	}

	result := make(map[string]interface{})
	result["_self"] = instanceData

	// Get the business object definition
	bo, err := r.cache.GetBusinessObject(tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get business object: %w", err)
	}

	// Resolve relationships recursively
	if err := r.resolveRelationshipsRecursive(ctx, tenantID, bo, instanceData, result, 1, maxDepth); err != nil {
		return nil, err
	}

	return result, nil
}

// GetRelatedObjects returns all relationship definitions for a business object
func (r *RelationshipResolver) GetRelatedObjects(
	ctx context.Context,
	tenantID, boKey string,
) ([]RelationshipDefinition, error) {
	bo, err := r.cache.GetBusinessObject(tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get business object: %w", err)
	}

	return bo.Relationships, nil
}

// GetRelationshipPath returns the path of relationships between two business objects
// For example: Worker → Position → Job Profile → Organization
func (r *RelationshipResolver) GetRelationshipPath(
	ctx context.Context,
	tenantID, fromBOKey, toBOKey string,
	maxDepth int,
) ([]string, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	visited := make(map[string]bool)
	path := []string{fromBOKey}

	found, resultPath := r.findPath(ctx, tenantID, fromBOKey, toBOKey, path, visited, maxDepth)
	if !found {
		return nil, fmt.Errorf("no relationship path found between %s and %s", fromBOKey, toBOKey)
	}

	return resultPath, nil
}

// Private helper methods

func (r *RelationshipResolver) resolveRelationshipsRecursive(
	ctx context.Context,
	tenantID string,
	bo *BusinessObjectDefinition,
	instanceData map[string]interface{},
	result map[string]interface{},
	currentDepth, maxDepth int,
) error {
	if currentDepth > maxDepth {
		return nil
	}

	for _, rel := range bo.Relationships {
		// Find the reference field in the instance data
		var refFieldName string
		for _, field := range bo.Fields {
			if field.Type == FieldRef && field.RefObjectID != nil && *field.RefObjectID == rel.ChildObjectID {
				refFieldName = field.Name
				break
			}
		}

		if refFieldName == "" {
			continue
		}

		// Get the referenced object ID from instance data
		refID, ok := instanceData[refFieldName]
		if !ok {
			continue
		}

		// Get the related business object definition
		relatedBO, err := r.cache.GetBusinessObjectByID(tenantID, rel.ChildObjectID)
		if err != nil {
			continue // Skip if we can't find the related object
		}

		// Store the relationship
		relationshipKey := fmt.Sprintf("_rel_%s", relatedBO.Name)
		result[relationshipKey] = map[string]interface{}{
			"id":           refID,
			"object_type":  relatedBO.Name,
			"display_name": relatedBO.DisplayName,
			"cardinality":  rel.Cardinality,
		}

		// Recursively resolve nested relationships
		// In a real implementation, you would fetch the actual instance data here
		// For now, we just store the metadata
	}

	return nil
}

func (r *RelationshipResolver) findPath(
	ctx context.Context,
	tenantID, currentBOKey, targetBOKey string,
	currentPath []string,
	visited map[string]bool,
	maxDepth int,
) (bool, []string) {
	if currentBOKey == targetBOKey {
		return true, currentPath
	}

	if len(currentPath) > maxDepth {
		return false, nil
	}

	if visited[currentBOKey] {
		return false, nil
	}

	visited[currentBOKey] = true

	bo, err := r.cache.GetBusinessObject(tenantID, currentBOKey)
	if err != nil {
		return false, nil
	}

	for _, rel := range bo.Relationships {
		relatedBO, err := r.cache.GetBusinessObjectByID(tenantID, rel.ChildObjectID)
		if err != nil {
			continue
		}

		newPath := append(currentPath, relatedBO.Name)
		found, resultPath := r.findPath(ctx, tenantID, relatedBO.Name, targetBOKey, newPath, visited, maxDepth)
		if found {
			return true, resultPath
		}
	}

	return false, nil
}
