package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
)

// ValidationRuleService manages validation rules as catalog nodes - the
// same storage convention as calculated semantic terms (catalog_node,
// AST in config.rule_ast) and pre-aggregations (catalog_node, dedicated
// CRUD handler, seeded node type). See docs/validation_rules_migration_report.json
// for why this replaces catalog_validation_rules rather than extending it:
// that table's rules targeted a schema catalog_edge/business_objects no
// longer has any trace of, and its condition_json format has no path to
// internal/rules/vm.RuleNode, the engine this evaluates against.
type ValidationRuleService struct {
	db *sqlx.DB
}

func NewValidationRuleService(db *sqlx.DB) *ValidationRuleService {
	return &ValidationRuleService{db: db}
}

// UpsertValidationRule creates or updates a validation-rule catalog node,
// and ensures a GOVERNED_BY_RULE edge exists from the rule node to its
// target BO's catalog node - the BO relationship as a native catalog edge
// rather than a foreign-key column, so it's queryable the same way every
// other catalog relationship is.
func (s *ValidationRuleService) UpsertValidationRule(ctx context.Context, req models.UpsertValidationRuleRequest) (*models.ValidationRuleDescriptor, error) {
	if len(req.RuleAST) == 0 {
		return nil, fmt.Errorf("rule_ast is required")
	}
	var probe vm.RuleNode
	if err := json.Unmarshal(req.RuleAST, &probe); err != nil {
		return nil, fmt.Errorf("rule_ast is not a valid vm.RuleNode: %w", err)
	}

	var nodeTypeID string
	if err := s.db.GetContext(ctx, &nodeTypeID, `
		SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule' LIMIT 1
	`); err != nil {
		return nil, fmt.Errorf("validation_rule node type not found: %w", err)
	}

	// business_objects.id is not a catalog_node id - the BO's actual
	// catalog_node representation is classification_node_id (verified
	// live: business_objects.id for "order" doesn't exist as a
	// catalog_node.id at all, while classification_node_id resolves to a
	// real node with qualified_path "/orm/order", matching
	// driver_table_name). catalog_edge.source/target_node_id must
	// reference catalog_node.id for the edge to actually be traversable -
	// an edge built from business_objects.id inserts without error (no FK
	// enforces it) but silently can never be joined back to anything.
	var boNodeID string
	if err := s.db.GetContext(ctx, &boNodeID, `
		SELECT classification_node_id FROM business_objects
		WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = $2::uuid
		LIMIT 1
	`, req.BOName, req.TenantID); err != nil {
		return nil, fmt.Errorf("BO %q not found: %w", req.BOName, err)
	}

	props := models.ValidationRuleProperties{
		BOName:           req.BOName,
		TenantID:         req.TenantID,
		Severity:         req.Severity,
		Timing:           req.Timing,
		Category:         req.Category,
		GovernanceStatus: "draft",
	}
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}

	cfg := models.ValidationRuleConfig{RuleAST: req.RuleAST}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	nodeID := uuid.New()
	qualifiedPath := fmt.Sprintf("validation_rule/%s/%s", req.TenantID, req.Name)

	err = s.db.GetContext(ctx, &nodeID, `
		INSERT INTO catalog_node (id, node_name, description, node_type_id, tenant_id, qualified_path, properties, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			config = EXCLUDED.config,
			updated_at = NOW()
		RETURNING id
	`, nodeID, req.Name, req.Description, nodeTypeID, req.TenantID, qualifiedPath, propsJSON, cfgJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert validation rule node: %w", err)
	}

	if err := s.ensureGovernedByRuleEdge(ctx, nodeID, boNodeID, req.TenantID); err != nil {
		return nil, fmt.Errorf("failed to link validation rule to BO: %w", err)
	}

	return s.GetByID(ctx, nodeID)
}

// ensureGovernedByRuleEdge creates the GOVERNED_BY_RULE catalog_edge from
// the BO node to the validation-rule node if it doesn't already exist.
// GOVERNED_BY_RULE was seeded into catalog_edge_type before this session
// but had zero real usages anywhere in the codebase - this is its first.
func (s *ValidationRuleService) ensureGovernedByRuleEdge(ctx context.Context, ruleNodeID uuid.UUID, boNodeID string, tenantID string) error {
	var edgeTypeID string
	if err := s.db.GetContext(ctx, &edgeTypeID, `
		SELECT id FROM catalog_edge_type WHERE edge_type_name = 'GOVERNED_BY_RULE' LIMIT 1
	`); err != nil {
		return fmt.Errorf("GOVERNED_BY_RULE edge type not found: %w", err)
	}

	var exists bool
	if err := s.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM catalog_edge
			WHERE source_node_id = $1::uuid AND target_node_id = $2::uuid AND edge_type_id = $3::uuid
		)
	`, boNodeID, ruleNodeID, edgeTypeID); err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, tenant_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, $4::uuid, NOW(), NOW())
	`, boNodeID, ruleNodeID, edgeTypeID, tenantID)
	return err
}

// SetActive flips is_active on a validation-rule catalog node - the
// reversible retire/reactivate toggle the BO Validations tab exposes,
// same convention as every other rule retirement in this engagement
// (the 233-rule corpus, stale probe rules): is_active = false, not a
// delete.
func (s *ValidationRuleService) SetActive(ctx context.Context, tenantID string, id uuid.UUID, active bool) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE catalog_node SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3::uuid
		  AND node_type_id = (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule' LIMIT 1)
	`, active, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set validation rule active state: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("validation rule %s not found for tenant %s", id, tenantID)
	}
	return nil
}

// GetByID loads a single validation rule by its catalog_node id.
func (s *ValidationRuleService) GetByID(ctx context.Context, id uuid.UUID) (*models.ValidationRuleDescriptor, error) {
	var node struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
		CreatedAt   string          `db:"created_at"`
		UpdatedAt   string          `db:"updated_at"`
		IsActive    bool            `db:"is_active"`
	}

	err := s.db.GetContext(ctx, &node, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config, n.created_at, n.updated_at, n.is_active
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("validation rule not found: %w", err)
	}

	desc, err := descriptorFromNode(node.ID, node.NodeName, node.Description, node.Properties, node.Config)
	if err != nil {
		return nil, err
	}
	desc.IsActive = node.IsActive
	return desc, nil
}

// ListByBO returns all validation rules targeting the given BO.
func (s *ValidationRuleService) ListByBO(ctx context.Context, tenantID, boName string) ([]models.ValidationRuleDescriptor, error) {
	var nodes []struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
	}

	err := s.db.SelectContext(ctx, &nodes, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = $1
		  AND n.properties->>'bo_name' = $2
		  AND n.is_active = true
		ORDER BY n.node_name
	`, tenantID, boName)
	if err != nil {
		return nil, err
	}

	result := make([]models.ValidationRuleDescriptor, 0, len(nodes))
	for _, n := range nodes {
		desc, err := descriptorFromNode(n.ID, n.NodeName, n.Description, n.Properties, n.Config)
		if err != nil {
			return nil, err
		}
		desc.IsActive = true // ListByBO's query already filters is_active = true
		result = append(result, *desc)
	}
	return result, nil
}

// ListAll returns every validation rule for the tenant, across all BOs -
// the system validations page's data source (the spec's "unfiltered
// tenant-wide variant" of ListByBO, which is always scoped to one BO).
// includeInactive=true also returns retired rules (is_active = false),
// so the page can show history/the archived-corpus distinction rather
// than only ever showing the live set.
func (s *ValidationRuleService) ListAll(ctx context.Context, tenantID string, includeInactive bool) ([]models.ValidationRuleDescriptor, error) {
	var nodes []struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
		IsActive    bool            `db:"is_active"`
	}

	query := `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config, n.is_active
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = $1`
	if !includeInactive {
		query += ` AND n.is_active = true`
	}
	query += ` ORDER BY n.properties->>'bo_name', n.node_name`

	if err := s.db.SelectContext(ctx, &nodes, query, tenantID); err != nil {
		return nil, err
	}

	result := make([]models.ValidationRuleDescriptor, 0, len(nodes))
	for _, n := range nodes {
		desc, err := descriptorFromNode(n.ID, n.NodeName, n.Description, n.Properties, n.Config)
		if err != nil {
			return nil, err
		}
		desc.IsActive = n.IsActive
		result = append(result, *desc)
	}
	return result, nil
}

func descriptorFromNode(id uuid.UUID, name, description string, propsRaw, cfgRaw json.RawMessage) (*models.ValidationRuleDescriptor, error) {
	props, err := models.ParseValidationRuleProperties(propsRaw)
	if err != nil {
		return nil, err
	}
	cfg, err := models.ParseValidationRuleConfig(cfgRaw)
	if err != nil {
		return nil, err
	}
	return &models.ValidationRuleDescriptor{
		ID:               id,
		TenantID:         props.TenantID,
		BOName:           props.BOName,
		Name:             name,
		Description:      description,
		Severity:         props.Severity,
		Timing:           props.Timing,
		Category:         props.Category,
		RuleAST:          cfg.RuleAST,
		GovernanceStatus: props.GovernanceStatus,
	}, nil
}

// Evaluate loads a validation rule's rule_ast and evaluates it against
// data via the unified engine (internal/rules/vm.AdvancedEvaluator) - the
// same evaluator the browser wasm build and (once bytecode dispatch for
// FuncCall lands) the fast VM path use. This is what the oracle-rule
// verification (exec_price > 0 vs. the live DB CHECK constraint) calls.
func (s *ValidationRuleService) Evaluate(ctx context.Context, id uuid.UUID, data map[string]interface{}) (bool, error) {
	desc, err := s.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	var node vm.RuleNode
	if err := json.Unmarshal(desc.RuleAST, &node); err != nil {
		return false, fmt.Errorf("stored rule_ast for %s did not parse: %w", id, err)
	}
	ae := vm.NewAdvancedEvaluator()
	return ae.Evaluate(node, data)
}

// PhysicalField is one field a rule can reference: Name is a semantic
// term (business_object_fields.field_name, e.g. "TargetQuantity") - the
// vocabulary a rule should be *authored* against, since it's portable
// across whichever physical binding the BO currently resolves to. See
// ListSemanticFields; the physical-column resolution happens once, at
// evaluation time, via ResolveSemanticFieldMap - a rule referencing
// "TargetQuantity" keeps working if the BO's binding ever points at a
// different physical column for it, the way one authored against
// "target_qty" directly would not.
type PhysicalField struct {
	Name     string `json:"name" db:"field_name"`
	DataType string `json:"dataType" db:"data_type"`
}

// ListSemanticFields returns the semantic terms a BO exposes for rule
// authoring, each paired with its currently-bound physical column's data
// type (for the editor's type-aware operator list) via the same
// business_object_fields -> MAPS_TO catalog-edge -> physical column chain
// GenerateDDL's dimension resolution already uses. Read directly off live
// metadata, not guessed - verified against the Order BO's 17 fields
// before wiring the frontend to it.
func (s *ValidationRuleService) ListSemanticFields(ctx context.Context, tenantID, boName string) ([]PhysicalField, error) {
	var fields []PhysicalField
	err := s.db.SelectContext(ctx, &fields, `
		SELECT bf.field_name, COALESCE(col.properties->>'data_type', '') AS data_type
		FROM business_object_fields bf
		JOIN business_objects bo ON bo.id = bf.bo_id
		JOIN catalog_edge ce ON ce.source_node_id = bf.term_node_id
		JOIN catalog_edge_type et ON et.id = ce.edge_type_id
		JOIN catalog_node col ON col.id = ce.target_node_id
		WHERE bo.bo_key = $1 AND bo.tenant_id = $2::uuid
		  AND et.edge_type_name = 'MAPS_TO'
		  AND col.qualified_path LIKE bo.driver_table_name || '/%'
		ORDER BY bf.field_name
	`, boName, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list semantic fields for BO %q: %w", boName, err)
	}
	return fields, nil
}

// ResolveSemanticFieldMap returns {semantic term -> currently-bound
// physical column} for one BO - the runtime counterpart to
// ListSemanticFields, called once per rule evaluation (not per rule) to
// translate a just-written row's physical columns into the semantic
// vocabulary a rule was actually authored against. Reads catalog
// metadata only (business_object_fields/catalog_edge/catalog_node), so
// it deliberately runs against db directly rather than the write's own
// transaction - this mapping doesn't change mid-write, and doing it
// outside the transaction avoids holding that transaction's locks any
// longer than the write itself needs.
func ResolveSemanticFieldMap(ctx context.Context, db *sqlx.DB, boID, driverTableName string) (map[string]string, error) {
	var rows []struct {
		FieldName  string `db:"field_name"`
		ColumnName string `db:"node_name"`
	}
	err := db.SelectContext(ctx, &rows, `
		SELECT bf.field_name, col.node_name
		FROM business_object_fields bf
		JOIN catalog_edge ce ON ce.source_node_id = bf.term_node_id
		JOIN catalog_edge_type et ON et.id = ce.edge_type_id
		JOIN catalog_node col ON col.id = ce.target_node_id
		WHERE bf.bo_id = $1::uuid
		  AND et.edge_type_name = 'MAPS_TO'
		  AND col.qualified_path LIKE $2 || '/%'
	`, boID, driverTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve semantic field map for BO %s: %w", boID, err)
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.FieldName] = r.ColumnName
	}
	return m, nil
}

// ResolveSemanticFieldMapForBinding is ResolveSemanticFieldMap's
// counterpart for a *non-canonical* binding: bindingID names a
// business_object_bindings row, and the map comes from field_bindings
// (RESOLVED rows only) rather than MAPS_TO. MAPS_TO stays the canonical/
// default binding (unchanged, zero migration risk to what's already
// proven live) - field_bindings becomes the table for every additional
// binding a BO picks up, which is exactly what it was designed for and
// never used for anywhere in this system before this.
//
// Deliberately returns only the terms this specific binding actually has
// RESOLVED field_bindings rows for - a rule referencing a term this
// binding doesn't cover must fail loud (a persisted rule_error via the
// same unresolvedFieldRefs check every other unresolvable reference hits
// - see internal/metadata/shadow_evaluation.go), never silently fall
// back to the canonical binding's map. A partial binding silently
// borrowing from another binding would be a new instance of the same
// disease this whole retrofit exists to close.
func ResolveSemanticFieldMapForBinding(ctx context.Context, db *sqlx.DB, bindingID string) (map[string]string, error) {
	var rows []struct {
		FieldName  string `db:"field_name"`
		ColumnName string `db:"node_name"`
	}
	err := db.SelectContext(ctx, &rows, `
		SELECT bf.field_name, col.node_name
		FROM field_bindings fb
		JOIN business_object_fields bf ON bf.id = fb.field_id
		JOIN catalog_node col ON col.id = fb.source_node_id
		WHERE fb.binding_id = $1::uuid
		  AND fb.binding_status = 'RESOLVED'
		  AND fb.source_type = 'COLUMN'
	`, bindingID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve field_bindings map for binding %s: %w", bindingID, err)
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.FieldName] = r.ColumnName
	}
	return m, nil
}
