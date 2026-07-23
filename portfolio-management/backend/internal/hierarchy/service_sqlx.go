package hierarchy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// HierarchySQLXServiceImpl is a sqlx-backed implementation of the hierarchy Service.
type HierarchySQLXServiceImpl struct {
	db     *sqlx.DB
	hasura HasuraClient
}

// NewHierarchyServiceSQLXImpl constructs a new sqlx-backed hierarchy service and returns it as the Service interface.
func NewHierarchyServiceSQLXImpl(db *sqlx.DB) Service {
	return &HierarchySQLXServiceImpl{db: db}
}

// NewHierarchyServiceWithHasura constructs a Hasura-enabled hierarchy service
func NewHierarchyServiceWithHasura(db *sqlx.DB, hasura HasuraClient) Service {
	return &HierarchySQLXServiceImpl{db: db, hasura: hasura}
}

func (s *HierarchySQLXServiceImpl) ValidateHierarchy(ctx context.Context, tenantID, parentModelType, childModelType string) (*HierarchyValidationResult, error) {
	if s.hasura != nil {
		return s.validateHierarchyWithHasura(ctx, tenantID, parentModelType, childModelType)
	}

	// TODO: Hasura-first pattern already implemented via validateHierarchyWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See validateHierarchyWithHasura() for the Hasura query:
	// query ValidateHierarchy($tenant_id: uuid!, $parent_type: String!, $child_type: String!)
	var r HierarchyValidationResult
	var rules []HierarchyRule
	query := `SELECT id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at FROM entity_hierarchy_rules WHERE tenant_id=$1 AND parent_model_type=$2 AND child_model_type=$3`
	if err := s.db.SelectContext(ctx, &rules, query, tenantID, parentModelType, childModelType); err != nil {
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

func (s *HierarchySQLXServiceImpl) validateHierarchyWithHasura(ctx context.Context, tenantID, parentModelType, childModelType string) (*HierarchyValidationResult, error) {
	query := `
        query ValidateHierarchy($tenant_id: uuid!, $parent_type: String!, $child_type: String!) {
            entity_hierarchy_rules(
                where: {
                    tenant_id: {_eq: $tenant_id},
                    parent_model_type: {_eq: $parent_type},
                    child_model_type: {_eq: $child_type}
                }
            ) {
                id
                tenant_id
                parent_model_type
                child_model_type
                allowed
                ownership_types
                max_children
                description
                notes
                created_at
                updated_at
            }
        }
    `

	result, err := s.hasura.Query(query, map[string]interface{}{
		"tenant_id":   tenantID,
		"parent_type": parentModelType,
		"child_type":  childModelType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate hierarchy via Hasura: %w", err)
	}

	rulesData, ok := result["entity_hierarchy_rules"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from Hasura")
	}

	rules := s.parseHierarchyRulesFromHasura(rulesData)
	return &HierarchyValidationResult{
		Valid:           len(rules) > 0,
		ParentModelType: parentModelType,
		ChildModelType:  childModelType,
		MatchingRules:   rules,
	}, nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchyRules(ctx context.Context, tenantID string) ([]HierarchyRule, error) {
	if s.hasura != nil {
		return s.getHierarchyRulesWithHasura(ctx, tenantID)
	}

	// TODO: Hasura-first pattern already implemented via getHierarchyRulesWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getHierarchyRulesWithHasura() for the Hasura query:
	// query GetHierarchyRules($tenant_id: uuid!)
	var rules []HierarchyRule
	query := `SELECT id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at FROM entity_hierarchy_rules WHERE tenant_id=$1`
	if err := s.db.SelectContext(ctx, &rules, query, tenantID); err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *HierarchySQLXServiceImpl) getHierarchyRulesWithHasura(ctx context.Context, tenantID string) ([]HierarchyRule, error) {
	query := `
        query GetHierarchyRules($tenant_id: uuid!) {
            entity_hierarchy_rules(
                where: {tenant_id: {_eq: $tenant_id}}
                order_by: [{parent_model_type: asc}, {child_model_type: asc}]
            ) {
                id
                tenant_id
                parent_model_type
                child_model_type
                allowed
                ownership_types
                max_children
                description
                notes
                created_at
                updated_at
            }
        }
    `

	result, err := s.hasura.Query(query, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy rules via Hasura: %w", err)
	}

	rulesData, ok := result["entity_hierarchy_rules"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from Hasura")
	}

	return s.parseHierarchyRulesFromHasura(rulesData), nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchySummary(ctx context.Context, tenantID string) ([]HierarchySummary, error) {
	if s.hasura != nil {
		return s.getHierarchySummaryWithHasura(ctx, tenantID)
	}

	// TODO: Hasura-first pattern already implemented via getHierarchySummaryWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getHierarchySummaryWithHasura() for the Hasura query:
	// query GetHierarchySummary($tenant_id: uuid!) { v_hierarchy_summary }
	var sums []HierarchySummary
	query := `SELECT tenant_id, parent_model_type, child_model_type, allowed, ownership_types, active_relationships, description FROM v_hierarchy_summary WHERE tenant_id=$1`
	if err := s.db.SelectContext(ctx, &sums, query, tenantID); err != nil {
		return nil, err
	}
	return sums, nil
}

func (s *HierarchySQLXServiceImpl) getHierarchySummaryWithHasura(ctx context.Context, tenantID string) ([]HierarchySummary, error) {
	query := `
        query GetHierarchySummary($tenant_id: uuid!) {
            v_hierarchy_summary(
                where: {tenant_id: {_eq: $tenant_id}}
            ) {
                tenant_id
                parent_model_type
                child_model_type
                allowed
                ownership_types
                active_relationships
                description
            }
        }
    `

	result, err := s.hasura.Query(query, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy summary via Hasura: %w", err)
	}

	summaryData, ok := result["v_hierarchy_summary"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from Hasura")
	}

	var summaries []HierarchySummary
	for _, item := range summaryData {
		summaryMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		summary := HierarchySummary{}
		if tid, ok := summaryMap["tenant_id"].(string); ok {
			summary.TenantID = tid
		}
		if pmt, ok := summaryMap["parent_model_type"].(string); ok {
			summary.ParentModelType = pmt
		}
		if cmt, ok := summaryMap["child_model_type"].(string); ok {
			summary.ChildModelType = cmt
		}
		if allowed, ok := summaryMap["allowed"].(bool); ok {
			summary.Allowed = allowed
		}
		if ot, ok := summaryMap["ownership_types"].(string); ok {
			json.Unmarshal([]byte(ot), &summary.OwnershipTypes)
		}
		if ar, ok := summaryMap["active_relationships"].(float64); ok {
			summary.ActiveRelationships = int64(ar)
		}
		if desc, ok := summaryMap["description"].(string); ok {
			summary.Description = desc
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetEntityHierarchy builds a tree rooted at rootID. If maxDepth < 0 then depth is unlimited.
func (s *HierarchySQLXServiceImpl) GetEntityHierarchy(ctx context.Context, rootID string, maxDepth int) (*EntityHierarchyNode, error) {
	// TODO: Consider implementing Hasura-first pattern for recursive hierarchy queries
	// While WITH RECURSIVE CTEs are complex, Hasura supports recursive relationships
	// through nested queries. Consider implementing a Hasura version that:
	// 1. Queries entity by rootID with nested entity_relationships
	// 2. Uses Hasura's relationship traversal for tree building
	// 3. Implements depth limiting via query structure
	// This would follow the same pattern as other methods with Hasura primary + SQL fallback
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
	if s.hasura != nil {
		return s.getHierarchyStatsWithHasura(ctx, tenantID)
	}

	// TODO: Hasura-first pattern already implemented via getHierarchyStatsWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getHierarchyStatsWithHasura() for the Hasura query:
	// query GetHierarchyStats($tenant_id: uuid!) { entities_aggregate { aggregate { count } } }
	var stats HierarchyStats
	var totalEntities int64
	if err := s.db.GetContext(ctx, &totalEntities, `SELECT count(*) FROM entities WHERE tenant_id=$1`, tenantID); err != nil {
		return nil, err
	}
	stats.TotalEntities = totalEntities
	return &stats, nil
}

func (s *HierarchySQLXServiceImpl) getHierarchyStatsWithHasura(ctx context.Context, tenantID string) (*HierarchyStats, error) {
	query := `
        query GetHierarchyStats($tenant_id: uuid!) {
            entities_aggregate(where: {tenant_id: {_eq: $tenant_id}}) {
                aggregate {
                    count
                }
            }
        }
    `

	result, err := s.hasura.Query(query, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy stats via Hasura: %w", err)
	}

	aggregateData, ok := result["entities_aggregate"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from Hasura")
	}

	aggregate, ok := aggregateData["aggregate"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected aggregate format from Hasura")
	}

	var stats HierarchyStats
	if count, ok := aggregate["count"].(float64); ok {
		stats.TotalEntities = int64(count)
	}

	return &stats, nil
}

func (s *HierarchySQLXServiceImpl) CreateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	if s.hasura != nil {
		return s.createHierarchyRuleWithHasura(ctx, rule)
	}

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

	// TODO: Hasura-first pattern already implemented via createHierarchyRuleWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See createHierarchyRuleWithHasura() for the Hasura mutation:
	// mutation CreateHierarchyRule($object: entity_hierarchy_rules_insert_input!)
	query := `INSERT INTO entity_hierarchy_rules (id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (id) DO UPDATE SET allowed=EXCLUDED.allowed, ownership_types=EXCLUDED.ownership_types, max_children=EXCLUDED.max_children, description=EXCLUDED.description, notes=EXCLUDED.notes, updated_at=EXCLUDED.updated_at`
	_, err = s.db.ExecContext(ctx, query, rule.ID, rule.TenantID, rule.ParentModelType, rule.ChildModelType, rule.Allowed, string(ownershipJSON), rule.MaxChildren, rule.Description, rule.Notes, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (s *HierarchySQLXServiceImpl) createHierarchyRuleWithHasura(ctx context.Context, rule *HierarchyRule) error {
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

	mutation := `
        mutation CreateHierarchyRule($object: entity_hierarchy_rules_insert_input!) {
            insert_entity_hierarchy_rules_one(
                object: $object,
                on_conflict: {
                    constraint: entity_hierarchy_rules_pkey,
                    update_columns: [allowed, ownership_types, max_children, description, notes, updated_at]
                }
            ) {
                id
                created_at
                updated_at
            }
        }
    `

	object := map[string]interface{}{
		"id":                rule.ID,
		"tenant_id":         rule.TenantID,
		"parent_model_type": rule.ParentModelType,
		"child_model_type":  rule.ChildModelType,
		"allowed":           rule.Allowed,
		"ownership_types":   string(ownershipJSON),
		"description":       rule.Description,
		"notes":             rule.Notes,
		"created_at":        rule.CreatedAt.Format(time.RFC3339),
		"updated_at":        rule.UpdatedAt.Format(time.RFC3339),
	}
	if rule.MaxChildren != nil {
		object["max_children"] = *rule.MaxChildren
	}

	_, err = s.hasura.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		return fmt.Errorf("failed to create hierarchy rule via Hasura: %w", err)
	}

	return nil
}

func (s *HierarchySQLXServiceImpl) UpdateHierarchyRule(ctx context.Context, rule *HierarchyRule) error {
	if s.hasura != nil {
		return s.updateHierarchyRuleWithHasura(ctx, rule)
	}

	// TODO: Hasura-first pattern already implemented via updateHierarchyRuleWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See updateHierarchyRuleWithHasura() for the Hasura mutation:
	// mutation UpdateHierarchyRule($id: uuid!, $tenant_id: uuid!, $_set: entity_hierarchy_rules_set_input!)
	rule.UpdatedAt = time.Now().UTC()
	ownershipJSON, err := json.Marshal(rule.OwnershipTypes)
	if err != nil {
		return err
	}
	query := `UPDATE entity_hierarchy_rules SET allowed=$1, ownership_types=$2, max_children=$3, description=$4, notes=$5, updated_at=$6 WHERE id=$7 AND tenant_id=$8`
	_, err = s.db.ExecContext(ctx, query, rule.Allowed, string(ownershipJSON), rule.MaxChildren, rule.Description, rule.Notes, rule.UpdatedAt, rule.ID, rule.TenantID)
	return err
}

func (s *HierarchySQLXServiceImpl) updateHierarchyRuleWithHasura(ctx context.Context, rule *HierarchyRule) error {
	rule.UpdatedAt = time.Now().UTC()
	ownershipJSON, err := json.Marshal(rule.OwnershipTypes)
	if err != nil {
		return err
	}

	mutation := `
        mutation UpdateHierarchyRule($id: uuid!, $tenant_id: uuid!, $_set: entity_hierarchy_rules_set_input!) {
            update_entity_hierarchy_rules(
                where: {
                    id: {_eq: $id},
                    tenant_id: {_eq: $tenant_id}
                },
                _set: $_set
            ) {
                affected_rows
                returning {
                    id
                    updated_at
                }
            }
        }
    `

	setObject := map[string]interface{}{
		"allowed":         rule.Allowed,
		"ownership_types": string(ownershipJSON),
		"description":     rule.Description,
		"notes":           rule.Notes,
		"updated_at":      rule.UpdatedAt.Format(time.RFC3339),
	}
	if rule.MaxChildren != nil {
		setObject["max_children"] = *rule.MaxChildren
	}

	result, err := s.hasura.Mutate(mutation, map[string]interface{}{
		"id":        rule.ID,
		"tenant_id": rule.TenantID,
		"_set":      setObject,
	})
	if err != nil {
		return fmt.Errorf("failed to update hierarchy rule via Hasura: %w", err)
	}

	updateData, ok := result["update_entity_hierarchy_rules"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from Hasura")
	}

	affectedRows, _ := updateData["affected_rows"].(float64)
	if affectedRows == 0 {
		return fmt.Errorf("no hierarchy rule found to update")
	}

	return nil
}

func (s *HierarchySQLXServiceImpl) DeleteHierarchyRule(ctx context.Context, tenantID, parentType, childType string) error {
	if s.hasura != nil {
		return s.deleteHierarchyRuleWithHasura(ctx, tenantID, parentType, childType)
	}

	// TODO: Hasura-first pattern already implemented via deleteHierarchyRuleWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See deleteHierarchyRuleWithHasura() for the Hasura mutation:
	// mutation DeleteHierarchyRule($tenant_id: uuid!, $parent_type: String!, $child_type: String!)
	_, err := s.db.ExecContext(ctx, `DELETE FROM entity_hierarchy_rules WHERE tenant_id=$1 AND parent_model_type=$2 AND child_model_type=$3`, tenantID, parentType, childType)
	return err
}

func (s *HierarchySQLXServiceImpl) deleteHierarchyRuleWithHasura(ctx context.Context, tenantID, parentType, childType string) error {
	mutation := `
        mutation DeleteHierarchyRule($tenant_id: uuid!, $parent_type: String!, $child_type: String!) {
            delete_entity_hierarchy_rules(
                where: {
                    tenant_id: {_eq: $tenant_id},
                    parent_model_type: {_eq: $parent_type},
                    child_model_type: {_eq: $child_type}
                }
            ) {
                affected_rows
            }
        }
    `

	result, err := s.hasura.Mutate(mutation, map[string]interface{}{
		"tenant_id":   tenantID,
		"parent_type": parentType,
		"child_type":  childType,
	})
	if err != nil {
		return fmt.Errorf("failed to delete hierarchy rule via Hasura: %w", err)
	}

	deleteData, ok := result["delete_entity_hierarchy_rules"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from Hasura")
	}

	affectedRows, _ := deleteData["affected_rows"].(float64)
	if affectedRows == 0 {
		return fmt.Errorf("no hierarchy rule found to delete")
	}

	return nil
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
			// TODO: Refactor to Hasura GraphQL bulk mutation
			// mutation BulkCreateRelationships($objects: [entity_relationships_insert_input!]!) {
			//   insert_entity_relationships(objects: $objects) { returning { id } affected_rows }
			// }
			_, err = tx.ExecContext(ctx, `INSERT INTO entity_relationships (id, tenant_id, owner_id, owned_id, ownership_percentage, ownership_type, incepting_date, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.New().String(), tenantID, op.OwnerID, op.OwnedID, op.OwnershipPct, op.OwnershipType, op.InceptingDate, time.Now().UTC())
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
	if s.hasura != nil {
		return s.logHierarchyAuditWithHasura(ctx, log)
	}

	// TODO: Hasura-first pattern already implemented via logHierarchyAuditWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See logHierarchyAuditWithHasura() for the Hasura mutation:
	// mutation LogHierarchyAudit($object: entity_hierarchy_audit_log_insert_input!)
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO entity_hierarchy_audit_log (id, entity_id, tenant_id, action, created_by, parent_model_type, child_model_type, reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, log.ID, log.EntityID, log.TenantID, log.Action, log.CreatedBy, log.ParentModelType, log.ChildModelType, log.Reason, log.CreatedAt)
	return err
}

func (s *HierarchySQLXServiceImpl) logHierarchyAuditWithHasura(ctx context.Context, log *HierarchyAuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	mutation := `
		mutation LogHierarchyAudit($object: entity_hierarchy_audit_log_insert_input!) {
			insert_entity_hierarchy_audit_log_one(object: $object) {
				id
				created_at
			}
		}
	`

	object := map[string]interface{}{
		"id":                log.ID,
		"entity_id":         log.EntityID,
		"tenant_id":         log.TenantID,
		"action":            log.Action,
		"parent_model_type": log.ParentModelType,
		"child_model_type":  log.ChildModelType,
		"reason":            log.Reason,
		"created_at":        log.CreatedAt.Format(time.RFC3339),
	}
	if log.PositionID != nil {
		object["position_id"] = *log.PositionID
	}
	if log.CreatedBy != nil {
		object["created_by"] = *log.CreatedBy
	}

	_, err := s.hasura.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		return fmt.Errorf("failed to log hierarchy audit via Hasura: %w", err)
	}

	return nil
}

func (s *HierarchySQLXServiceImpl) GetHierarchyAuditLog(ctx context.Context, entityID string, limit int) ([]HierarchyAuditLog, error) {
	if s.hasura != nil {
		return s.getHierarchyAuditLogWithHasura(ctx, entityID, limit)
	}

	// TODO: Hasura-first pattern already implemented via getHierarchyAuditLogWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getHierarchyAuditLogWithHasura() for the Hasura query:
	// query GetHierarchyAuditLog($entity_id: uuid!, $limit: Int!)
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

func (s *HierarchySQLXServiceImpl) getHierarchyAuditLogWithHasura(ctx context.Context, entityID string, limit int) ([]HierarchyAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		query GetHierarchyAuditLog($entity_id: uuid!, $limit: Int!) {
			entity_hierarchy_audit_log(
				where: {entity_id: {_eq: $entity_id}}
				order_by: [{created_at: desc}]
				limit: $limit
			) {
				id
				entity_id
				tenant_id
				position_id
				action
				parent_model_type
				child_model_type
				reason
				created_by
				created_at
			}
		}
	`

	result, err := s.hasura.Query(query, map[string]interface{}{
		"entity_id": entityID,
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy audit log via Hasura: %w", err)
	}

	logsData, ok := result["entity_hierarchy_audit_log"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from Hasura")
	}

	var logs []HierarchyAuditLog
	for _, item := range logsData {
		logMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		log := HierarchyAuditLog{}
		if id, ok := logMap["id"].(string); ok {
			log.ID = id
		}
		if eid, ok := logMap["entity_id"].(string); ok {
			log.EntityID = eid
		}
		if tid, ok := logMap["tenant_id"].(string); ok {
			log.TenantID = tid
		}
		if pid, ok := logMap["position_id"].(string); ok {
			log.PositionID = &pid
		}
		if action, ok := logMap["action"].(string); ok {
			log.Action = action
		}
		if pmt, ok := logMap["parent_model_type"].(string); ok {
			log.ParentModelType = pmt
		}
		if cmt, ok := logMap["child_model_type"].(string); ok {
			log.ChildModelType = cmt
		}
		if reason, ok := logMap["reason"].(string); ok {
			log.Reason = reason
		}
		if cb, ok := logMap["created_by"].(string); ok {
			log.CreatedBy = &cb
		}
		if createdAt, ok := logMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				log.CreatedAt = t
			}
		}

		logs = append(logs, log)
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
	// TODO: Refactor to Hasura GraphQL bulk mutation
	// mutation ImportHierarchyRules($objects: [entity_hierarchy_rules_insert_input!]!) {
	//   insert_entity_hierarchy_rules(
	//     objects: $objects
	//     on_conflict: {constraint: entity_hierarchy_rules_pkey, update_columns: []}
	//   ) { returning { id } affected_rows }
	// }
	for _, r := range req.Rules {
		ruleID := uuid.New().String()
		ownershipJSON, _ := json.Marshal(r.OwnershipTypes)
		res, err := tx.ExecContext(ctx, `INSERT INTO entity_hierarchy_rules (id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, description, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO NOTHING`, ruleID, tenantID, r.ParentModelType, r.ChildModelType, true, string(ownershipJSON), r.Description, time.Now().UTC(), time.Now().UTC())
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
	// TODO: Refactor to Hasura GraphQL
	// query {
	//   entities_aggregate(where: {tenant_id: {_eq: $tenant_id}}) {
	//     aggregate { count }
	//   }
	// }
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT count(*) FROM entities WHERE tenant_id=$1`, tenantID); err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no entities found for tenant")
	}
	return nil
}

// parseHierarchyRulesFromHasura is a helper function to parse hierarchy rules from Hasura response
func (s *HierarchySQLXServiceImpl) parseHierarchyRulesFromHasura(rulesData []interface{}) []HierarchyRule {
	var rules []HierarchyRule
	for _, item := range rulesData {
		ruleMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		rule := HierarchyRule{}
		if id, ok := ruleMap["id"].(string); ok {
			rule.ID = id
		}
		if tid, ok := ruleMap["tenant_id"].(string); ok {
			rule.TenantID = tid
		}
		if pmt, ok := ruleMap["parent_model_type"].(string); ok {
			rule.ParentModelType = pmt
		}
		if cmt, ok := ruleMap["child_model_type"].(string); ok {
			rule.ChildModelType = cmt
		}
		if allowed, ok := ruleMap["allowed"].(bool); ok {
			rule.Allowed = allowed
		}
		if ot, ok := ruleMap["ownership_types"].(string); ok {
			json.Unmarshal([]byte(ot), &rule.OwnershipTypes)
		}
		if maxChildren, ok := ruleMap["max_children"].(float64); ok {
			mc := int(maxChildren)
			rule.MaxChildren = &mc
		}
		if desc, ok := ruleMap["description"].(string); ok {
			rule.Description = desc
		}
		if notes, ok := ruleMap["notes"].(string); ok {
			rule.Notes = notes
		}
		if createdAt, ok := ruleMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				rule.CreatedAt = t
			}
		}
		if updatedAt, ok := ruleMap["updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				rule.UpdatedAt = t
			}
		}

		rules = append(rules, rule)
	}
	return rules
}
