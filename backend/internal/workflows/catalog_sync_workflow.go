package workflows

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/models"
)

func safeString(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

// CatalogSyncWorkflow synchronizes API endpoint changes to catalog nodes and edges
func CatalogSyncWorkflow(ctx workflow.Context, event events.DomainEvent) error {
	// Set workflow options
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Route event to appropriate handler
	switch evt := event.(type) {
	case *events.APIEndpointEvent:
		return handleAPIEndpointEvent(ctx, evt)
	case *events.EntityMappingEvent:
		return handleEntityMappingEvent(ctx, evt)
	case *events.DatasourceMappingEvent:
		return handleDatasourceMappingEvent(ctx, evt)
	default:
		return nil
	}
}

// handleAPIEndpointEvent processes API endpoint lifecycle events
func handleAPIEndpointEvent(ctx workflow.Context, event *events.APIEndpointEvent) error {
	var result interface{}

	switch event.EventType {
	case events.APIEndpointCreated:
		// Create catalog node for new endpoint
		err := workflow.ExecuteActivity(
			ctx,
			CreateEndpointCatalogNodeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog node created event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogNodeCreatedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err

	case events.APIEndpointUpdated:
		// Update catalog node
		err := workflow.ExecuteActivity(
			ctx,
			UpdateEndpointCatalogNodeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog node updated event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogNodeUpdatedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err

	case events.APIEndpointDeleted:
		// Delete catalog node (soft delete)
		err := workflow.ExecuteActivity(
			ctx,
			DeleteEndpointCatalogNodeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog node deleted event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogNodeDeletedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err

	case events.APIEndpointActivated:
		// Reactivate catalog node and related edges
		err := workflow.ExecuteActivity(
			ctx,
			ActivateEndpointNodesActivity,
			event,
		).Get(ctx, &result)
		return err
	}

	return nil
}

// handleEntityMappingEvent processes entity mapping events
func handleEntityMappingEvent(ctx workflow.Context, event *events.EntityMappingEvent) error {
	var result interface{}

	switch event.EventType {
	case events.EntityMappingCreated:
		// Create catalog edge between endpoint and entity
		err := workflow.ExecuteActivity(
			ctx,
			CreateMappingCatalogEdgeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog edge created event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogEdgeCreatedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err

	case events.EntityMappingDeleted:
		// Delete catalog edge
		err := workflow.ExecuteActivity(
			ctx,
			DeleteMappingCatalogEdgeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog edge deleted event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogEdgeDeletedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err
	}

	return nil
}

// handleDatasourceMappingEvent processes datasource mapping events
func handleDatasourceMappingEvent(ctx workflow.Context, event *events.DatasourceMappingEvent) error {
	var result interface{}

	switch event.EventType {
	case events.DatasourceMappingCreated:
		// Create catalog edge between endpoint and datasource
		err := workflow.ExecuteActivity(
			ctx,
			CreateDatasourceMappingEdgeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog edge created event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogEdgeCreatedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err

	case events.DatasourceMappingDeleted:
		// Delete catalog edge
		err := workflow.ExecuteActivity(
			ctx,
			DeleteDatasourceMappingEdgeActivity,
			event,
		).Get(ctx, &result)
		if err != nil {
			return err
		}

		// Emit catalog edge deleted event
		err = workflow.ExecuteActivity(
			ctx,
			PublishCatalogEdgeDeletedActivity,
			result,
			event.TenantID,
		).Get(ctx, nil)
		return err
	}

	return nil
}

// CreateEndpointCatalogNodeActivity creates a catalog node for an API endpoint
func CreateEndpointCatalogNodeActivity(ctx workflow.Context, event *events.APIEndpointEvent) (models.CatalogNode, error) {
	tenantID, _ := uuid.Parse(event.TenantID)
	datasourceID, _ := uuid.Parse(event.DatasourceID)
	endpointID, _ := uuid.Parse(event.EndpointID)

	properties, _ := json.Marshal(map[string]interface{}{
		"endpoint_id":     event.EndpointID,
		"endpoint_uuid":   endpointID.String(),
		"endpoint_name":   safeString(event.Endpoint["endpoint_name"]),
		"http_method":     safeString(event.Endpoint["http_method"]),
		"url_path":        safeString(event.Endpoint["url_path"]),
		"category":        safeString(event.Endpoint["category"]),
		"description":     safeString(event.Endpoint["description"]),
		"request_schema":  event.Endpoint["request_schema"],
		"response_schema": event.Endpoint["response_schema"],
		"is_active":       true,
	})

	node := models.CatalogNode{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceId: datasourceID,
		NodeTypeID:         uuid.Nil,
		NodeName:           safeString(event.Endpoint["endpoint_name"]),
		Description:        safeString(event.Endpoint["description"]),
		Properties:         properties,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		QualifiedPath:      safeString(event.Endpoint["url_path"]),
		IsAlpha:            true,
		CoreID:             uuid.NullUUID{},
		ParentID:           uuid.NullUUID{},
	}

	return node, nil
}

// UpdateEndpointCatalogNodeActivity updates a catalog node for an API endpoint
func UpdateEndpointCatalogNodeActivity(ctx workflow.Context, event *events.APIEndpointEvent) (models.CatalogNode, error) {
	tenantID, _ := uuid.Parse(event.TenantID)
	datasourceID, _ := uuid.Parse(event.DatasourceID)
	endpointUUID, parseErr := uuid.Parse(event.EndpointID)
	if parseErr != nil {
		endpointUUID = uuid.New()
	}

	properties, _ := json.Marshal(map[string]interface{}{
		"endpoint_id":     event.EndpointID,
		"endpoint_uuid":   endpointUUID.String(),
		"endpoint_name":   safeString(event.Endpoint["endpoint_name"]),
		"http_method":     safeString(event.Endpoint["http_method"]),
		"url_path":        safeString(event.Endpoint["url_path"]),
		"category":        safeString(event.Endpoint["category"]),
		"description":     safeString(event.Endpoint["description"]),
		"request_schema":  event.Endpoint["request_schema"],
		"response_schema": event.Endpoint["response_schema"],
		"is_active":       event.Endpoint["is_active"],
	})

	node := models.CatalogNode{
		ID:                 endpointUUID,
		TenantID:           tenantID,
		TenantDatasourceId: datasourceID,
		NodeTypeID:         uuid.Nil,
		NodeName:           safeString(event.Endpoint["endpoint_name"]),
		Description:        safeString(event.Endpoint["description"]),
		Properties:         properties,
		UpdatedAt:          time.Now(),
		QualifiedPath:      safeString(event.Endpoint["url_path"]),
		IsAlpha:            true,
		CoreID:             uuid.NullUUID{},
		ParentID:           uuid.NullUUID{},
	}

	return node, nil
}

// DeleteEndpointCatalogNodeActivity deletes a catalog node for an API endpoint
func DeleteEndpointCatalogNodeActivity(ctx workflow.Context, event *events.APIEndpointEvent) (string, error) {
	// Return the node ID that was deleted
	return event.EndpointID, nil
}

// ActivateEndpointNodesActivity reactivates a catalog node and related edges
func ActivateEndpointNodesActivity(ctx workflow.Context, event *events.APIEndpointEvent) (string, error) {
	return event.EndpointID, nil
}

// CreateMappingCatalogEdgeActivity creates a catalog edge for an entity mapping
func CreateMappingCatalogEdgeActivity(ctx workflow.Context, event *events.EntityMappingEvent) (models.CatalogEdge, error) {
	tenantID, _ := uuid.Parse(event.TenantID)
	sourceID, _ := uuid.Parse(event.APIEndpointID)
	targetID, _ := uuid.Parse(event.EntityID)

	edge := models.CatalogEdge{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceId: uuid.Nil,
		SourceNodeID:       sourceID,
		TargetNodeID:       targetID,
		CreatedAt:          time.Now(),
		CoreID:             uuid.NullUUID{},
		EdgeTypeID:         uuid.Nil,
	}

	return edge, nil
}

// DeleteMappingCatalogEdgeActivity deletes a catalog edge
func DeleteMappingCatalogEdgeActivity(ctx workflow.Context, event *events.EntityMappingEvent) (string, error) {
	edgeID := uuid.New().String() // In real implementation, lookup existing edge
	return edgeID, nil
}

// CreateDatasourceMappingEdgeActivity creates a catalog edge for a datasource mapping
func CreateDatasourceMappingEdgeActivity(ctx workflow.Context, event *events.DatasourceMappingEvent) (models.CatalogEdge, error) {
	tenantID, _ := uuid.Parse(event.TenantID)
	sourceID, _ := uuid.Parse(event.APIEndpointID)
	targetID, _ := uuid.Parse(event.DatasourceID)

	edge := models.CatalogEdge{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceId: uuid.Nil,
		SourceNodeID:       sourceID,
		TargetNodeID:       targetID,
		CreatedAt:          time.Now(),
		CoreID:             uuid.NullUUID{},
		EdgeTypeID:         uuid.Nil,
	}

	return edge, nil
}

// DeleteDatasourceMappingEdgeActivity deletes a datasource mapping edge
func DeleteDatasourceMappingEdgeActivity(ctx workflow.Context, event *events.DatasourceMappingEvent) (string, error) {
	edgeID := uuid.New().String() // In real implementation, lookup existing edge
	return edgeID, nil
}

// PublishCatalogNodeCreatedActivity publishes a catalog node created event
func PublishCatalogNodeCreatedActivity(ctx workflow.Context, node models.CatalogNode, tenantID string) error {
	// In real implementation, this would publish to RabbitMQ
	return nil
}

// PublishCatalogNodeUpdatedActivity publishes a catalog node updated event
func PublishCatalogNodeUpdatedActivity(ctx workflow.Context, node models.CatalogNode, tenantID string) error {
	return nil
}

// PublishCatalogNodeDeletedActivity publishes a catalog node deleted event
func PublishCatalogNodeDeletedActivity(ctx workflow.Context, nodeID string, tenantID string) error {
	return nil
}

// PublishCatalogEdgeCreatedActivity publishes a catalog edge created event
func PublishCatalogEdgeCreatedActivity(ctx workflow.Context, edge models.CatalogEdge, tenantID string) error {
	return nil
}

// PublishCatalogEdgeDeletedActivity publishes a catalog edge deleted event
func PublishCatalogEdgeDeletedActivity(ctx workflow.Context, edgeID string, tenantID string) error {
	return nil
}
