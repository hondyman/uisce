package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
)

// CalcTermService manages calculated semantic terms as catalog nodes -
// the same storage convention as validation rules (catalog_node, AST in
// config.rule_ast, ValidationRuleService's sibling) and the thing
// PreAggregationService.compileCalcTermToSQL (pre_aggregation_service.go)
// has looked up by node_name + properties.term_type='calculated' since
// item 26, but that nothing before this file could actually create:
// cmd/verify_calc_measure hand-wrote the catalog_node row directly via
// SQL because no service or API endpoint existed. This is that endpoint.
//
// Deliberately separate from the three pre-existing, unrelated "calc"
// systems already in this codebase - internal/services.SemanticResolver
// (regex substitution over raw SQL text), internal/handlers.CalcHandler's
// public.calc_fields (raw sql_expr strings, string-interpolated into
// queries), and catalog_validation_rules' own legacy calc concept - none
// of which produce or consume a vm.Expression. Same reasoning
// ValidationRuleService documented for replacing catalog_validation_rules
// rather than extending it: a real AST needs a real, dedicated home, not
// a field bolted onto a format that was never built to carry one.
type CalcTermService struct {
	db *sqlx.DB
}

func NewCalcTermService(db *sqlx.DB) *CalcTermService {
	return &CalcTermService{db: db}
}

// UpsertCalcTerm parses req.Expression via vm.ParseExpression and stores
// the resulting AST as a "calculated" catalog_node - the term becomes
// immediately usable by PreAggregationService.GenerateDDL (by node_name)
// and by direct evaluation (GetByID + Evaluate) without any other
// wiring. A syntax error in the expression is returned as-is (a
// *vm.ParseError, carrying a byte position) rather than wrapped, so an
// HTTP handler can surface the position to the editor.
func (s *CalcTermService) UpsertCalcTerm(ctx context.Context, req models.UpsertCalcTermRequest) (*models.CalcTermDescriptor, error) {
	if strings.TrimSpace(req.Expression) == "" {
		return nil, fmt.Errorf("expression is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	expr, err := vm.ParseExpression(req.Expression)
	if err != nil {
		return nil, err
	}
	ruleAST, err := json.Marshal(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parsed expression: %w", err)
	}

	var nodeTypeID string
	if err := s.db.GetContext(ctx, &nodeTypeID, `
		SELECT id FROM catalog_node_type WHERE catalog_type_name = 'calculation_term' LIMIT 1
	`); err != nil {
		return nil, fmt.Errorf("calculation_term node type not found: %w", err)
	}

	props := models.CalcTermProperties{
		BOName:   req.BOName,
		TenantID: req.TenantID,
		TermType: "calculated",
		DataType: req.DataType,
	}
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}

	cfg := models.CalcTermConfig{Expression: req.Expression, RuleAST: ruleAST}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	// Scanning RETURNING id back into nodeID (rather than trusting the
	// client-generated uuid.New() passed into VALUES) is what keeps this
	// upsert correct on the ON CONFLICT DO UPDATE path - a sibling
	// function (PreAggregationService.UpsertPreAggregation) doesn't do
	// this and returns a stale id when a conflict fires (see
	// docs/unified-rule-engine-handoff.md item 27); ValidationRuleService
	// already gets this right and is the pattern followed here.
	nodeID := uuid.New()
	qualifiedPath := fmt.Sprintf("calc_term/%s/%s", req.TenantID, req.Name)

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
		return nil, fmt.Errorf("failed to upsert calc term node: %w", err)
	}

	return s.GetByID(ctx, nodeID)
}

// GetByID loads a single calc term by its catalog_node id.
func (s *CalcTermService) GetByID(ctx context.Context, id uuid.UUID) (*models.CalcTermDescriptor, error) {
	var node struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
	}
	err := s.db.GetContext(ctx, &node, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'calculation_term'
		  AND n.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("calc term not found: %w", err)
	}
	return calcTermDescriptorFromNode(node.ID, node.NodeName, node.Description, node.Properties, node.Config)
}

// ListByBO returns every calc term scoped to the given BO.
func (s *CalcTermService) ListByBO(ctx context.Context, tenantID, boName string) ([]models.CalcTermDescriptor, error) {
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
		WHERE nt.catalog_type_name = 'calculation_term'
		  AND n.tenant_id = $1
		  AND n.properties->>'bo_name' = $2
		  AND n.is_active = true
		ORDER BY n.node_name
	`, tenantID, boName)
	if err != nil {
		return nil, err
	}
	result := make([]models.CalcTermDescriptor, 0, len(nodes))
	for _, n := range nodes {
		desc, err := calcTermDescriptorFromNode(n.ID, n.NodeName, n.Description, n.Properties, n.Config)
		if err != nil {
			return nil, err
		}
		result = append(result, *desc)
	}
	return result, nil
}

// PreviewSQL parses expression text and compiles it to SQL for the given
// BO, without saving anything - the "does this compile, and to what?"
// check an editor needs before Save. Uses the exact same
// ResolveSemanticFieldMap + vm.CompileToSQL chain GenerateDDL uses for a
// saved term (see the comment on that resolveColumn closure) - a preview
// that compiled differently than the real DDL generator would be worse
// than no preview at all.
func (s *CalcTermService) PreviewSQL(ctx context.Context, tenantID, boName, expression string) (string, error) {
	expr, err := vm.ParseExpression(expression)
	if err != nil {
		return "", err
	}

	var bo struct {
		ID              string `db:"id"`
		DriverTableName string `db:"driver_table_name"`
	}
	if err := s.db.GetContext(ctx, &bo, `
		SELECT id, COALESCE(driver_table_name, '') AS driver_table_name
		FROM business_objects
		WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = $2::uuid
		LIMIT 1
	`, boName, tenantID); err != nil {
		return "", fmt.Errorf("BO %q not found: %w", boName, err)
	}

	fieldMap, err := ResolveSemanticFieldMap(ctx, s.db, bo.ID, bo.DriverTableName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve semantic field map for BO %s: %w", bo.ID, err)
	}
	resolveColumn := func(path string) (string, error) {
		col, ok := fieldMap[path]
		if !ok {
			return "", fmt.Errorf("no MAPS_TO binding for field %q on BO %s", path, boName)
		}
		return col, nil
	}
	return vm.CompileToSQL(expr, resolveColumn)
}

// Evaluate parses expression text and evaluates it numerically against a
// data context - the server-side counterpart to the WASM build's
// evaluateExpressionText (cmd/wasm/main.go), so "Test with sample data"
// works identically whether or not the editor's WASM runtime has loaded.
func (s *CalcTermService) Evaluate(expression string, data map[string]interface{}) (float64, error) {
	expr, err := vm.ParseExpression(expression)
	if err != nil {
		return 0, err
	}
	ae := vm.NewAdvancedEvaluator()
	return ae.EvaluateNumeric(vm.RuleNode{Type: vm.NodeTypeExpression, Expression: expr}, data)
}

func calcTermDescriptorFromNode(id uuid.UUID, name, description string, propsRaw, cfgRaw json.RawMessage) (*models.CalcTermDescriptor, error) {
	props, err := models.ParseCalcTermProperties(propsRaw)
	if err != nil {
		return nil, err
	}
	cfg, err := models.ParseCalcTermConfig(cfgRaw)
	if err != nil {
		return nil, err
	}
	return &models.CalcTermDescriptor{
		ID:          id,
		TenantID:    props.TenantID,
		BOName:      props.BOName,
		Name:        name,
		Description: description,
		Expression:  cfg.Expression,
		RuleAST:     cfg.RuleAST,
		DataType:    props.DataType,
	}, nil
}
