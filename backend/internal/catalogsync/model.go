package catalogsync

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ChangeType reflects how a row was mutated.
type ChangeType string

const (
	ChangeNone   ChangeType = "none"
	ChangeInsert ChangeType = "insert"
	ChangeUpdate ChangeType = "update"
	ChangeDelete ChangeType = "delete"
)

// CatalogChangeEvent matches the cross-service event contract.
type CatalogChangeEvent struct {
	EventID    string            `json:"eventId"`
	EntityType string            `json:"entityType"`
	ChangeType ChangeType        `json:"changeType"`
	TenantID   string            `json:"tenantId"`
	OccurredAt time.Time         `json:"occurredAt"`
	Before     map[string]string `json:"before,omitempty"`
	After      map[string]string `json:"after,omitempty"`
	Source     string            `json:"source"`
}

// EventPublisher emits catalog change events.
type EventPublisher interface {
	Publish(ctx context.Context, event CatalogChangeEvent) error
	Close(ctx context.Context) error
}

// NodeInput represents the semantic fields needed to upsert a catalog_node.
type NodeInput struct {
	ID                 uuid.UUID      `json:"id"`
	TypeID             uuid.UUID      `json:"typeId"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	QualifiedPath      string         `json:"qualifiedPath"`
	ParentID           *uuid.UUID     `json:"parentId"`
	Properties         map[string]any `json:"properties"`
	Config             map[string]any `json:"config"`
	TenantID           uuid.UUID      `json:"tenantId"`
	TenantDatasourceID *uuid.UUID     `json:"tenantDatasourceId"`
}

// EdgeInput represents the semantic fields needed to upsert a catalog_edge.
type EdgeInput struct {
	ID                 uuid.UUID      `json:"id"`
	SourceNodeID       uuid.UUID      `json:"sourceNodeId"`
	TargetNodeID       uuid.UUID      `json:"targetNodeId"`
	EdgeType           string         `json:"edgeType"`
	EdgeTypeID         uuid.UUID      `json:"edgeTypeId"`
	RelationshipType   *string        `json:"relationshipType"`
	Properties         map[string]any `json:"properties"`
	TenantID           uuid.UUID      `json:"tenantId"`
	TenantDatasourceID *uuid.UUID     `json:"tenantDatasourceId"`
}

// UpsertResult captures what changed along with snapshots.
type UpsertResult struct {
	Change ChangeType
	Before map[string]string
	After  map[string]string
}
