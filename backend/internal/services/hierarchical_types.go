package services

import "github.com/jmoiron/sqlx/types"

// HierarchicalNode represents a node in a hierarchical structure, with parent-child relationships.
type HierarchicalNode struct {
	ID       string      `json:"id"`
	ParentID *string     `json:"parentId,omitempty"`
	Data     interface{} `json:"data"`
	Type     string      `json:"type"`
}

// ContainerNodeData holds data specific to container nodes (schemas, tables).
type ContainerNodeData struct {
	Label      string `json:"label"`
	Expandable bool   `json:"expandable"`
	IsExpanded bool   `json:"isExpanded"`
}

// DatabaseHierarchy represents the full schema/table/column structure of a database.
type DatabaseHierarchy struct {
	Databases []Database `json:"databases"`
}

// Database represents a single database in the hierarchy.
type Database struct {
	Name    string   `json:"name"`
	Schemas []Schema `json:"schemas"`
}

// Schema represents a schema within a database.
type Schema struct {
	Name   string  `json:"name"`
	Tables []Table `json:"tables"`
}

// Table represents a table within a schema.
type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Column represents a column within a table.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ReactFlowNode represents a node in the ReactFlow format.
type ReactFlowNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Data     types.JSONText `json:"data"`
	Position NodePosition   `json:"position"`
	ParentID *string        `json:"parentNode,omitempty"`
	Extent   string         `json:"extent,omitempty"`
}

// NodePosition represents the x, y coordinates of a node.
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
