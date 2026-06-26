package models

import (
	"encoding/json"
	"time"
)

// CatalogNode represents a node in the semantic catalog
type CatalogNode struct {
	ID                 string          `db:"id" json:"id"`
	NodeName           string          `db:"node_name" json:"node_name"`
	QualifiedPath      string          `db:"qualified_path" json:"qualified_path"`
	NodeTypeID         string          `db:"node_type_id" json:"node_type_id"`
	TenantDatasourceID *string         `db:"tenant_datasource_id" json:"tenant_datasource_id"`
	CatalogTypeName    string          `db:"catalog_type" json:"catalog_type"` // Maps to node_type in DB if present, or generic label
	Description        *string         `db:"description" json:"description"`
	IsActive           *bool           `db:"is_active" json:"is_active"`
	ParentTypeID       *string         `db:"parent_type_id" json:"parent_type_id"`
	ParentID           *string         `db:"parent_id" json:"parent_id"`
	Config             *string         `db:"config" json:"config"`
	Properties         json.RawMessage `db:"properties" json:"properties"`
	IsMapped           bool            `db:"is_mapped" json:"is_mapped"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at" json:"updated_at"`
	TenantID           string          `db:"tenant_id" json:"tenant_id"`
	CoreID             *string         `db:"core_id" json:"core_id"`
}

// CatalogEdge represents a relationship between nodes
type CatalogEdge struct {
	ID                string          `db:"id" json:"id"`
	EdgeTypeName      string          `db:"edge_type_name" json:"predicate"`
	Description       *string         `db:"description" json:"description"`
	SubjectNodeTypeID string          `db:"subject_node_type_id" json:"subject_node_type_id"`
	ObjectNodeTypeID  string          `db:"object_node_type_id" json:"object_node_type_id"`
	Properties        json.RawMessage `db:"properties" json:"properties"`
	IsActive          *bool           `db:"is_active" json:"is_active"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
	TenantID          string          `db:"tenant_id" json:"tenant_id"`
	CoreID            *string         `db:"core_id" json:"core_id"`
}
