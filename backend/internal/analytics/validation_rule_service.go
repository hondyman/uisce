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
	}

	err := s.db.GetContext(ctx, &node, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config, n.created_at, n.updated_at
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("validation rule not found: %w", err)
	}

	return descriptorFromNode(node.ID, node.NodeName, node.Description, node.Properties, node.Config)
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
