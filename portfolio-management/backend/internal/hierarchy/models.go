package hierarchy

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Investment Entity Types

// HierarchyRule defines allowed parent-child relationships between entity types
type HierarchyRule struct {
	ID              string      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID        string      `gorm:"type:uuid;index" json:"tenant_id"`
	ParentModelType string      `gorm:"type:varchar(100);index" json:"parent_model_type"`
	ChildModelType  string      `gorm:"type:varchar(100);index" json:"child_model_type"`
	Allowed         bool        `gorm:"default:true" json:"allowed"`
	OwnershipTypes  StringArray `gorm:"type:text[]" json:"ownership_types"` // ["PERCENT_BASED", "SHARE_BASED", "VALUE_BASED"]
	MaxChildren     *int        `json:"max_children,omitempty"`
	Description     string      `json:"description"`
	Notes           string      `json:"notes,omitempty"`
	CreatedAt       time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

// StringArray is a custom type for handling text arrays
type StringArray []string

// Value implements the driver.Valuer interface
func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	return json.Unmarshal(value.([]byte), &a)
}

// HierarchySummary represents a summary view of hierarchy rules with active relationships
type HierarchySummary struct {
	TenantID            string      `json:"tenant_id"`
	ParentModelType     string      `json:"parent_model_type"`
	ChildModelType      string      `json:"child_model_type"`
	Allowed             bool        `json:"allowed"`
	OwnershipTypes      StringArray `json:"ownership_types"`
	ActiveRelationships int64       `json:"active_relationships"`
	Description         string      `json:"description"`
}

// EntityHierarchyNode represents a single node in the hierarchy tree
type EntityHierarchyNode struct {
	ID          string                `json:"id"`
	TenantID    string                `json:"tenant_id"`
	ModelType   string                `json:"model_type"`
	DisplayName string                `json:"display_name"`
	ParentID    *string               `json:"parent_id,omitempty"`
	Depth       int                   `json:"depth"`
	PathIDs     []string              `json:"path_ids"`
	PathNames   []string              `json:"path_names"`
	Level       int                   `json:"level"`
	Children    []EntityHierarchyNode `json:"children,omitempty"`
}

// EntityHierarchyTree represents a complete hierarchy tree
type EntityHierarchyTree struct {
	Roots []EntityHierarchyNode `json:"roots"`
	Stats HierarchyStats        `json:"stats"`
}

// HierarchyStats provides statistics about the entity hierarchy
type HierarchyStats struct {
	TotalEntities    int64 `json:"total_entities"`
	TotalPositions   int64 `json:"total_positions"`
	MaxDepth         int   `json:"max_depth"`
	TopLevelEntities int64 `json:"top_level_entities"`
	LeafNodes        int64 `json:"leaf_nodes"`
	AllowedRules     int64 `json:"allowed_rules"`
	DisallowedRules  int64 `json:"disallowed_rules"`
}

// HierarchyValidationResult contains validation results for hierarchy operations
type HierarchyValidationResult struct {
	Valid               bool            `json:"valid"`
	Errors              []string        `json:"errors"`
	Warnings            []string        `json:"warnings"`
	ParentModelType     string          `json:"parent_model_type"`
	ChildModelType      string          `json:"child_model_type"`
	MatchingRules       []HierarchyRule `json:"matching_rules,omitempty"`
	RecommendedParents  []string        `json:"recommended_parents,omitempty"`
	RecommendedChildren []string        `json:"recommended_children,omitempty"`
}

// HierarchyAuditLog tracks changes to entity hierarchies
type HierarchyAuditLog struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID        string    `gorm:"type:uuid;index" json:"tenant_id"`
	EntityID        string    `gorm:"type:uuid;index" json:"entity_id"`
	PositionID      *string   `gorm:"type:uuid" json:"position_id,omitempty"`
	Action          string    `gorm:"type:varchar(50)" json:"action"` // CREATE, UPDATE, DELETE, VALIDATE_FAIL
	ParentModelType string    `json:"parent_model_type,omitempty"`
	ChildModelType  string    `json:"child_model_type,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	CreatedBy       *string   `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// HierarchyBulkRequest contains multiple hierarchy operations
type HierarchyBulkRequest struct {
	Operations []HierarchyOperation `json:"operations"`
	Validate   bool                 `json:"validate"` // Pre-validate all before executing
}

// HierarchyOperation represents a single hierarchy operation
type HierarchyOperation struct {
	Operation     string     `json:"operation"` // "CREATE", "UPDATE", "DELETE"
	OwnerID       string     `json:"owner_id"`
	OwnedID       string     `json:"owned_id"`
	OwnershipPct  float64    `json:"ownership_percentage,omitempty"`
	OwnershipType string     `json:"ownership_type,omitempty"`
	InceptingDate *time.Time `json:"incepting_date,omitempty"`
}

// HierarchyBulkResponse contains results of bulk operations
type HierarchyBulkResponse struct {
	Successful    int                        `json:"successful"`
	Failed        int                        `json:"failed"`
	Results       []HierarchyOperationResult `json:"results"`
	ErrorsSummary []string                   `json:"errors_summary"`
}

// HierarchyOperationResult represents the result of a single operation
type HierarchyOperationResult struct {
	Operation  HierarchyOperation `json:"operation"`
	Success    bool               `json:"success"`
	Message    string             `json:"message"`
	PositionID *string            `json:"position_id,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// HierarchyGraphRequest represents a request to build a graph between entities
type HierarchyGraphRequest struct {
	RootEntityID string `json:"root_entity_id"`
	Depth        int    `json:"depth,omitempty"` // Default: -1 (unlimited)
	IncludeStats bool   `json:"include_stats"`
	Format       string `json:"format,omitempty"` // "tree", "flat", "dot", "mermaid"
}

// HierarchyGraphResponse provides hierarchy visualization data
type HierarchyGraphResponse struct {
	Root     EntityHierarchyNode `json:"root"`
	Depth    int                 `json:"depth"`
	Stats    HierarchyStats      `json:"stats,omitempty"`
	Format   string              `json:"format"`
	GraphDOT string              `json:"graph_dot,omitempty"` // GraphViz DOT format
	Mermaid  string              `json:"mermaid,omitempty"`   // Mermaid diagram
}

// ModelType represents an investment entity type definition
type ModelType struct {
	ID            string                 `gorm:"type:uuid;primaryKey" json:"id"`
	ModelType     string                 `gorm:"uniqueIndex;type:varchar(100)" json:"model_type"`
	DisplayName   string                 `json:"display_name"`
	OwnershipType string                 `json:"ownership_type"`
	Description   string                 `json:"description"`
	Category      string                 `gorm:"index" json:"category"`
	IsActive      bool                   `gorm:"default:true;index" json:"is_active"`
	Attributes    map[string]interface{} `gorm:"type:jsonb" json:"attributes,omitempty"`
	CreatedAt     time.Time              `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time              `gorm:"autoUpdateTime" json:"updated_at"`
}

// HierarchyImportRequest allows importing pre-defined hierarchy rules
type HierarchyImportRequest struct {
	Rules []struct {
		ParentModelType string   `json:"parent_model_type"`
		ChildModelType  string   `json:"child_model_type"`
		OwnershipTypes  []string `json:"ownership_types"`
		Description     string   `json:"description"`
	} `json:"rules"`
}

// HierarchyImportResponse provides import results
type HierarchyImportResponse struct {
	Imported  int       `json:"imported"`
	Skipped   int       `json:"skipped"`
	Failed    int       `json:"failed"`
	Errors    []string  `json:"errors"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for GORM
func (HierarchyRule) TableName() string {
	return "entity_hierarchy_rules"
}

func (HierarchyAuditLog) TableName() string {
	return "entity_hierarchy_audit_log"
}

func (ModelType) TableName() string {
	return "model_types"
}

// Constants for audit actions
const (
	AuditActionCreate       = "CREATE"
	AuditActionUpdate       = "UPDATE"
	AuditActionDelete       = "DELETE"
	AuditActionValidateFail = "VALIDATE_FAIL"
)

// Constants for entity categories
const (
	CategoryOrganization = "organization"
	CategoryFund         = "fund"
	CategoryContainer    = "container"
	CategorySecurity     = "security"
	CategoryDerivative   = "derivative"
	CategoryAlternative  = "alternative"
	CategoryInsurance    = "insurance"
	CategoryDebt         = "debt"
	CategoryCash         = "cash"
	CategoryDigital      = "digital"
	CategoryStructured   = "structured"
	CategoryLegacy       = "legacy"
	CategoryCustom       = "custom"
)

// Constants for ownership types
const (
	OwnershipTypePercentBased = "PERCENT_BASED"
	OwnershipTypeShareBased   = "SHARE_BASED"
	OwnershipTypeValueBased   = "VALUE_BASED"
	OwnershipTypeAny          = "Any"
)
