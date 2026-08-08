package hierarchy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/db/queries"
	"github.com/jmoiron/sqlx"
)

// HierarchySQLXServiceImpl is a sqlx-backed implementation of the hierarchy Service.
type HierarchySQLXServiceImpl struct {
	db *sqlx.DB
}

// NewHierarchyServiceSQLXImpl constructs a new sqlx-backed hierarchy service and returns it as the Service interface.
func NewHierarchyServiceSQLXImpl(db *sqlx.DB) Service {
	return &HierarchySQLXServiceImpl{db: db}
}

func (s *HierarchySQLXServiceImpl) ValidateHierarchy(ctx context.Context, tenantID, parentModelType, childModelType string) (*HierarchyValidationResult, error) {
	var r HierarchyValidationResult
	var rules []HierarchyRule
	if err := s.db.SelectContext(ctx, &rules, queries.GetHierarchyRuleByTypes, tenantID, parentModelType, childModelType); err != nil {
		if err == sql.ErrNoRows {
			r.Valid = false
			return &r, nil
		}
		return nil, err
	}
	r.MatchingRules = rules
	r.ParentModelType = parentModelType
	r.ChildModelType = childModelType
	r.Valid = len(rules) > 0
	return &r, nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchyRules(ctx context.Context, tenantID string) ([]HierarchyRule, error) {
	var rules []HierarchyRule
	if err := s.db.SelectContext(ctx, &rules, queries.GetHierarchyRules, tenantID); err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchySummary(ctx context.Context, tenantID string) ([]HierarchySummary, error) {
	var sums []HierarchySummary
	query := `SELECT tenant_id, parent_model_type, child_model_type, allowed, ownership_types, active_relationships, description FROM v_hierarchy_summary WHERE tenant_id=$1`
	if err := s.db.SelectContext(ctx, &sums, query, tenantID); err != nil {
		return nil, err
	}
	return sums, nil
}

// GetEntityHierarchy builds a tree rooted at rootID. If maxDepth < 0 then depth is unlimited.
func (s *HierarchySQLXServiceImpl) GetEntityHierarchy(ctx context.Context, rootID string, maxDepth int) (*EntityHierarchyNode, error) {
	if rootID == "" {
		return nil, errors.New("rootID required")
	}

	query := `WITH RECURSIVE tree AS (
        SELECT e.id, e.tenant_id, e.model_type, e.display_name, NULL::uuid AS parent_id, 0 AS depth, ARRAY[e.id]::text[] AS path_ids, ARRAY[e.display_name]::text[] AS path_names
        FROM entities e
        WHERE e.id = $1
    UNION ALL
        SELECT c.id, c.tenant_id, c.model_type, c.display_name, r.owner_id AS parent_id, t.depth + 1 AS depth, t.path_ids || c.id, t.path_names || c.display_name
        FROM entity_relationships r
        JOIN entities c ON c.id = r.owned_id
        JOIN tree t ON r.owner_id = t.id
        WHERE ($2 < 0 OR t.depth + 1 <= $2)
    )
    SELECT id, tenant_id, model_type, display_name, parent_id, depth, to_json(path_ids) AS path_ids_json, to_json(path_names) AS path_names_json FROM tree ORDER BY depth, id;`

	type nodeRow struct {
		ID            string         `db:"id"`
		TenantID      string         `db:"tenant_id"`
		ModelType     string         `db:"model_type"`
		DisplayName   string         `db:"display_name"`
		ParentID      sql.NullString `db:"parent_id"`
		Depth         int            `db:"depth"`
		PathIDsJSON   string         `db:"path_ids_json"`
		PathNamesJSON string         `db:"path_names_json"`
	}

	var rows []nodeRow
	if err := s.db.SelectContext(ctx, &rows, query, rootID, maxDepth); err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("root entity not found")
	}

	nodes := make(map[string]*EntityHierarchyNode)
	for _, r := range rows {
		var pathIDs []string
		var pathNames []string
		if r.PathIDsJSON != "" {
			_ = json.Unmarshal([]byte(r.PathIDsJSON), &pathIDs)
		}
		if r.PathNamesJSON != "" {
			_ = json.Unmarshal([]byte(r.PathNamesJSON), &pathNames)
		}

		var parentPtr *string
		if r.ParentID.Valid {
			p := r.ParentID.String
			parentPtr = &p
		}

		node := &EntityHierarchyNode{
			ID:          r.ID,
			TenantID:    r.TenantID,
			ModelType:   r.ModelType,
			DisplayName: r.DisplayName,
			ParentID:    parentPtr,
			Depth:       r.Depth,
			PathIDs:     pathIDs,
			PathNames:   pathNames,
			Level:       r.Depth,
			Children:    []EntityHierarchyNode{},
		}
		nodes[r.ID] = node
	}

	var root *EntityHierarchyNode
	for _, n := range nodes {
		if n.ParentID != nil {
			if parentNode, ok := nodes[*n.ParentID]; ok {
				parentNode.Children = append(parentNode.Children, *n)
			}
		}
		if n.ID == rootID {
			root = n
		}
	}

	if root == nil {
		for _, n := range nodes {
			if n.Depth == 0 {
				root = n
				break
			}
		}
	}

	if root == nil {
		return nil, errors.New("unable to locate root node")
	}
	return root, nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchyStats(ctx context.Context, tenantID string) (*HierarchyStats, error) {
	var stats HierarchyStats
	var totalEntities int64
	if err := s.db.GetContext(ctx, &totalEntities, queries.CountEntitiesByTenant, tenantID); err != nil {
		return nil, err
	}
	stats.TotalEntities = totalEntities
	return &stats, nil
}

func (s *HierarchySQLXServiceImpl) CreateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	ownershipJSON, err := json.Marshal(rule.OwnershipTypes)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, queries.InsertHierarchyRule, rule.ID, rule.TenantID, rule.ParentModelType, rule.ChildModelType, rule.Allowed, string(ownershipJSON), rule.MaxChildren, rule.Description, rule.Notes, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (s *HierarchySQLXServiceImpl) UpdateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	rule.UpdatedAt = time.Now().UTC()
	ownershipJSON, err := json.Marshal(rule.OwnershipTypes)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, queries.UpdateHierarchyRule, rule.Allowed, string(ownershipJSON), rule.MaxChildren, rule.Description, rule.Notes, rule.UpdatedAt, rule.ID, rule.TenantID)
	return err
}

func (s *HierarchySQLXServiceImpl) DeleteHierarchyRule(ctx context.Context, tenantID, parentType, childType string) error {
	_, err := s.db.ExecContext(ctx, queries.DeleteHierarchyRule, tenantID, parentType, childType)
	return err
}

func (s *HierarchySQLXServiceImpl) BulkCreateOperations(ctx context.Context, tenantID string, req *HierarchyBulkRequest) (*HierarchyBulkResponse, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	resp := &HierarchyBulkResponse{Successful: 0, Failed: 0, Results: []HierarchyOperationResult{}}
	for _, op := range req.Operations {
		if op.Operation == "CREATE" {
			_, err = tx.ExecContext(ctx, queries.InsertEntityRelationship, uuid.New().String(), tenantID, op.OwnerID, op.OwnedID, op.OwnershipPct, op.OwnershipType, op.InceptingDate, time.Now().UTC())
			if err != nil {
				resp.Failed++
				resp.Results = append(resp.Results, HierarchyOperationResult{Operation: op, Success: false, Message: "failed to create relationship", Error: err.Error()})
				continue
			}
			resp.Successful++
			resp.Results = append(resp.Results, HierarchyOperationResult{Operation: op, Success: true, Message: "created"})
		} else {
			resp.Failed++
			resp.Results = append(resp.Results, HierarchyOperationResult{Operation: op, Success: false, Message: "unsupported operation"})
		}
	}
	return resp, nil
}

func (s *HierarchySQLXServiceImpl) LogHierarchyAudit(ctx context.Context, log *HierarchyAuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, queries.InsertHierarchyAuditLog, log.ID, log.EntityID, log.TenantID, log.Action, log.CreatedBy, log.ParentModelType, log.ChildModelType, log.Reason, log.CreatedAt)
	return err
}

func (s *HierarchySQLXServiceImpl) GetHierarchyAuditLog(ctx context.Context, entityID string, limit int) ([]HierarchyAuditLog, error) {
	var logs []HierarchyAuditLog
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, entity_id, tenant_id, position_id, action, parent_model_type, child_model_type, reason, created_by, created_at FROM entity_hierarchy_audit_log WHERE entity_id=$1 ORDER BY created_at DESC LIMIT $2`
	if err := s.db.SelectContext(ctx, &logs, query, entityID, limit); err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *HierarchySQLXServiceImpl) ImportHierarchyRules(ctx context.Context, tenantID string, req *HierarchyImportRequest) (*HierarchyImportResponse, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	resp := &HierarchyImportResponse{Imported: 0, Skipped: 0, Failed: 0, Errors: []string{}, CreatedAt: time.Now().UTC()}
	for _, r := range req.Rules {
		ruleID := uuid.New().String()
		ownershipJSON, _ := json.Marshal(r.OwnershipTypes)
		res, err := tx.ExecContext(ctx, queries.InsertHierarchyRuleNoConflict, ruleID, tenantID, r.ParentModelType, r.ChildModelType, true, string(ownershipJSON), r.Description, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, err.Error())
			continue
		}
		cnt, _ := res.RowsAffected()
		if cnt > 0 {
			resp.Imported++
		} else {
			resp.Skipped++
		}
	}
	return resp, nil
}

func (s *HierarchySQLXServiceImpl) ValidateEntityConsistency(ctx context.Context, tenantID string) error {
	var count int
	if err := s.db.GetContext(ctx, &count, queries.CountEntitiesByTenant, tenantID); err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no entities found for tenant")
	}
	return nil
}
