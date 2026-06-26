package catalogsync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// NodeRepository provides idempotent upserts with before/after snapshots.
type NodeRepository struct {
	db *sqlx.DB
}

func NewNodeRepository(db *sqlx.DB) *NodeRepository { return &NodeRepository{db: db} }

// ResolveNodeID returns the catalog_node.id for a given natural key.
func (r *NodeRepository) ResolveNodeID(ctx context.Context, tenantDatasourceID *uuid.UUID, typeID uuid.UUID, qualifiedPath string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, `
		SELECT id
		FROM catalog_node
		WHERE tenant_datasource_id IS NOT DISTINCT FROM $1
		  AND node_type_id = $2
		  AND qualified_path = $3
	`, tenantDatasourceID, typeID, qualifiedPath)
	return id, err
}

// EdgeRepository provides idempotent upserts with before/after snapshots.
type EdgeRepository struct {
	db *sqlx.DB
}

func NewEdgeRepository(db *sqlx.DB) *EdgeRepository { return &EdgeRepository{db: db} }

type dbNode struct {
	ID                 uuid.UUID       `db:"id"`
	TypeID             uuid.UUID       `db:"node_type_id"`
	Name               string          `db:"node_name"`
	Description        sql.NullString  `db:"description"`
	QualifiedPath      string          `db:"qualified_path"`
	ParentID           uuid.NullUUID   `db:"parent_id"`
	PropertiesRaw      json.RawMessage `db:"properties"`
	ConfigRaw          json.RawMessage `db:"config"`
	SchemaHash         string          `db:"schema_hash"`
	TenantID           uuid.UUID       `db:"tenant_id"`
	TenantDatasourceID uuid.NullUUID   `db:"tenant_datasource_id"`
}

type dbEdge struct {
	ID                 uuid.UUID       `db:"id"`
	SourceNodeID       uuid.UUID       `db:"source_node_id"`
	TargetNodeID       uuid.UUID       `db:"target_node_id"`
	EdgeType           string          `db:"edge_type"`
	EdgeTypeID         uuid.UUID       `db:"edge_type_id"`
	RelationshipType   sql.NullString  `db:"relationship_type"`
	PropertiesRaw      json.RawMessage `db:"properties"`
	SchemaHash         string          `db:"schema_hash"`
	TenantID           uuid.NullUUID   `db:"tenant_id"`
	TenantDatasourceID uuid.NullUUID   `db:"tenant_datasource_id"`
}

func (r *NodeRepository) Upsert(ctx context.Context, input NodeInput, schemaHash string) (*UpsertResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existing dbNode
	err = tx.Get(&existing, `
		SELECT id, node_type_id, node_name, description, qualified_path, parent_id, properties, config, schema_hash, tenant_id, tenant_datasource_id
		FROM catalog_node
		WHERE tenant_datasource_id IS NOT DISTINCT FROM $1
		  AND node_type_id = $2
		  AND qualified_path = $3
	`, input.TenantDatasourceID, input.TypeID, input.QualifiedPath)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	isInsert := errors.Is(err, sql.ErrNoRows)

	if !isInsert && existing.SchemaHash == schemaHash {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &UpsertResult{Change: ChangeNone, Before: nodeSnapshot(existing), After: nodeSnapshot(existing)}, nil
	}

	if isInsert {
		if input.ID == uuid.Nil {
			input.ID = uuid.New()
		}
		props, _ := json.Marshal(input.Properties)
		cfg, _ := json.Marshal(input.Config)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (
				id, node_type_id, node_name, description, properties, qualified_path, parent_id,
				created_at, updated_at, tenant_id, schema_hash, tenant_datasource_id, config
			) VALUES ($1,$2,$3,$4,$5,$6,$7,now(),now(),$8,$9,$10,$11)
		`, input.ID, input.TypeID, input.Name, input.Description, props, input.QualifiedPath, input.ParentID, input.TenantID, schemaHash, input.TenantDatasourceID, cfg)
		if err != nil {
			return nil, err
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &UpsertResult{Change: ChangeInsert, Before: nil, After: nodeSnapshotFromInput(input, schemaHash)}, nil
	}

	props, _ := json.Marshal(input.Properties)
	cfg, _ := json.Marshal(input.Config)
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_node
		SET node_name = $2,
			description = $3,
			properties = $4,
			parent_id = $5,
			updated_at = now(),
			schema_hash = $6,
			config = $7
		WHERE id = $1
	`, existing.ID, input.Name, input.Description, props, input.ParentID, schemaHash, cfg)
	if err != nil {
		return nil, err
	}

	before := nodeSnapshot(existing)
	after := nodeSnapshotFromInput(NodeInput{
		ID:                 existing.ID,
		TypeID:             input.TypeID,
		Name:               input.Name,
		Description:        input.Description,
		QualifiedPath:      input.QualifiedPath,
		ParentID:           input.ParentID,
		Properties:         input.Properties,
		Config:             input.Config,
		TenantID:           input.TenantID,
		TenantDatasourceID: input.TenantDatasourceID,
	}, schemaHash)

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, commitErr
	}

	return &UpsertResult{Change: ChangeUpdate, Before: before, After: after}, nil
}

func (r *EdgeRepository) Upsert(ctx context.Context, input EdgeInput, schemaHash string) (*UpsertResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existing dbEdge
	err = tx.Get(&existing, `
		SELECT id, source_node_id, target_node_id, edge_type, edge_type_id, relationship_type, properties, schema_hash, tenant_id, tenant_datasource_id
		FROM catalog_edge
		WHERE tenant_datasource_id IS NOT DISTINCT FROM $1
		  AND source_node_id = $2
		  AND edge_type_id = $3
		  AND target_node_id = $4
	`, input.TenantDatasourceID, input.SourceNodeID, input.EdgeTypeID, input.TargetNodeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	isInsert := errors.Is(err, sql.ErrNoRows)
	if !isInsert && existing.SchemaHash == schemaHash {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &UpsertResult{Change: ChangeNone, Before: edgeSnapshot(existing), After: edgeSnapshot(existing)}, nil
	}

	if isInsert {
		if input.ID == uuid.Nil {
			input.ID = uuid.New()
		}
		props, _ := json.Marshal(input.Properties)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (
				id, source_node_id, target_node_id, edge_type, edge_type_id, tenant_id, tenant_datasource_id,
				relationship_type, properties, created_at, updated_at, schema_hash
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now(),$10)
		`, input.ID, input.SourceNodeID, input.TargetNodeID, input.EdgeType, input.EdgeTypeID, input.TenantID, input.TenantDatasourceID, input.RelationshipType, props, schemaHash)
		if err != nil {
			return nil, err
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &UpsertResult{Change: ChangeInsert, Before: nil, After: edgeSnapshotFromInput(input, schemaHash)}, nil
	}

	props, _ := json.Marshal(input.Properties)
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_edge
		SET edge_type = $2,
			relationship_type = $3,
			properties = $4,
			updated_at = now(),
			schema_hash = $5
		WHERE id = $1
	`, existing.ID, input.EdgeType, input.RelationshipType, props, schemaHash)
	if err != nil {
		return nil, err
	}

	before := edgeSnapshot(existing)
	after := edgeSnapshotFromInput(EdgeInput{
		ID:                 existing.ID,
		SourceNodeID:       input.SourceNodeID,
		TargetNodeID:       input.TargetNodeID,
		EdgeType:           input.EdgeType,
		EdgeTypeID:         input.EdgeTypeID,
		RelationshipType:   input.RelationshipType,
		Properties:         input.Properties,
		TenantID:           input.TenantID,
		TenantDatasourceID: input.TenantDatasourceID,
	}, schemaHash)

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, commitErr
	}

	return &UpsertResult{Change: ChangeUpdate, Before: before, After: after}, nil
}

func nodeSnapshot(n dbNode) map[string]string {
	return map[string]string{
		"id":                n.ID.String(),
		"node_type_id":      n.TypeID.String(),
		"node_name":         n.Name,
		"description":       nullString(n.Description),
		"qualified_path":    n.QualifiedPath,
		"parent_id":         nullUUID(n.ParentID),
		"properties":        rawToString(n.PropertiesRaw),
		"config":            rawToString(n.ConfigRaw),
		"schema_hash":       n.SchemaHash,
		"tenant_id":         n.TenantID.String(),
		"tenant_datasource": nullUUID(n.TenantDatasourceID),
	}
}

func nodeSnapshotFromInput(n NodeInput, schemaHash string) map[string]string {
	props, _ := json.Marshal(n.Properties)
	cfg, _ := json.Marshal(n.Config)
	return map[string]string{
		"id":                n.ID.String(),
		"node_type_id":      n.TypeID.String(),
		"node_name":         n.Name,
		"description":       n.Description,
		"qualified_path":    n.QualifiedPath,
		"parent_id":         uuidStringPtr(n.ParentID),
		"properties":        string(props),
		"config":            string(cfg),
		"schema_hash":       schemaHash,
		"tenant_id":         n.TenantID.String(),
		"tenant_datasource": uuidStringPtr(n.TenantDatasourceID),
	}
}

func edgeSnapshot(e dbEdge) map[string]string {
	return map[string]string{
		"id":                e.ID.String(),
		"source_node_id":    e.SourceNodeID.String(),
		"target_node_id":    e.TargetNodeID.String(),
		"edge_type":         e.EdgeType,
		"edge_type_id":      e.EdgeTypeID.String(),
		"relationship_type": nullString(e.RelationshipType),
		"properties":        rawToString(e.PropertiesRaw),
		"schema_hash":       e.SchemaHash,
		"tenant_id":         nullUUID(e.TenantID),
		"tenant_datasource": nullUUID(e.TenantDatasourceID),
	}
}

func edgeSnapshotFromInput(e EdgeInput, schemaHash string) map[string]string {
	props, _ := json.Marshal(e.Properties)
	return map[string]string{
		"id":                e.ID.String(),
		"source_node_id":    e.SourceNodeID.String(),
		"target_node_id":    e.TargetNodeID.String(),
		"edge_type":         e.EdgeType,
		"edge_type_id":      e.EdgeTypeID.String(),
		"relationship_type": stringPtr(e.RelationshipType),
		"properties":        string(props),
		"schema_hash":       schemaHash,
		"tenant_id":         e.TenantID.String(),
		"tenant_datasource": uuidStringPtr(e.TenantDatasourceID),
	}
}

func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func nullUUID(id uuid.NullUUID) string {
	if !id.Valid {
		return ""
	}
	return id.UUID.String()
}

func nullString(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}
