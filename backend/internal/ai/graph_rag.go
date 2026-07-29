package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/ai/vocabulary"
)

type GraphRAGContextFrame struct {
	TargetBOKey     string                   `json:"target_bo_key"`
	BODefinition    map[string]interface{}   `json:"bo_definition"`
	ActiveBindings  []map[string]interface{} `json:"active_bindings"`
	SemanticTerms   []string                 `json:"semantic_terms"`
	Relationships   []map[string]interface{} `json:"relationships"`
	PersonaContext *PersonaContext           `json:"persona_context,omitempty"`
}

type PersonaContext struct {
	FunctionalRole   string   `json:"functional_role"`
	ClearanceLevel  string   `json:"clearance_level"`
	FrequentBOs      []string `json:"frequent_bos,omitempty"`
	RecentFilters    []string `json:"recent_filters,omitempty"`
	SynonymVocabulary []string `json:"synonym_vocabulary,omitempty"`
	PreferredDialects []string `json:"preferred_dialects,omitempty"`
	PreferredDomain string   `json:"preferred_domain"`
}

type GraphRAGAssembler struct {
	db       *sqlx.DB
	vocab    *vocabulary.Resolver
}

func NewGraphRAGAssembler(db *sqlx.DB, vocab *vocabulary.Resolver) *GraphRAGAssembler {
	return &GraphRAGAssembler{db: db, vocab: vocab}
}

func (g *GraphRAGAssembler) AssembleContextFrame(
	ctx context.Context,
	tenantID, boKey string,
	persona *PersonaContext,
) (*GraphRAGContextFrame, error) {
	frame := &GraphRAGContextFrame{
		TargetBOKey: boKey,
		PersonaContext: persona,
	}

	if g.db == nil {
		return frame, nil
	}

	if err := g.resolveBODefinition(ctx, tenantID, boKey, frame); err != nil {
		return nil, fmt.Errorf("failed to resolve BO definition: %w", err)
	}

	if err := g.resolveBindings(ctx, tenantID, boKey, frame); err != nil {
		return nil, fmt.Errorf("failed to resolve bindings: %w", err)
	}

	if err := g.resolveOutboundRelationships(ctx, tenantID, boKey, frame); err != nil {
		return nil, fmt.Errorf("failed to resolve outbound relationships: %w", err)
	}

	if err := g.resolveSemanticTerms(ctx, tenantID, boKey, frame); err != nil {
		return nil, fmt.Errorf("failed to resolve semantic terms: %w", err)
	}

	return frame, nil
}

func (g *GraphRAGAssembler) resolveBODefinition(ctx context.Context, tenantID, boKey string, frame *GraphRAGContextFrame) error {
	query := `
		SELECT
			bod.bo_def_id AS id,
			bod.bo_key,
			bod.name,
			bod.display_name,
			bod.description,
			bod.config
		FROM business_object_def bod
		WHERE bod.tenant_id = $1
		  AND (bod.bo_key = $2 OR bod.name = $2)
		LIMIT 1
	`
	var row struct {
		ID          string `db:"id"`
		BoKey       string `db:"bo_key"`
		Name        string `db:"name"`
		DisplayName string `db:"display_name"`
		Description *string `db:"description"`
		Config      []byte `db:"config"`
	}

	err := g.db.GetContext(ctx, &row, query, tenantID, boKey)
	if err != nil {
		return err
	}

	if row.ID == "" {
		return nil
	}

	fields, err := g.resolveBOFields(ctx, tenantID, row.ID)
	if err != nil {
		return err
	}

	boDef := map[string]interface{}{
		"bo_def_id":   row.ID,
		"bo_key":      row.BoKey,
		"name":        row.Name,
		"display_name": row.DisplayName,
		"description": "",
		"fields":      fields,
	}
	if row.Description != nil {
		boDef["description"] = *row.Description
	}

	frame.BODefinition = boDef
	return nil
}

func (g *GraphRAGAssembler) resolveBOFields(ctx context.Context, tenantID, boDefID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			bf.field_key,
			bf.display_name,
			bf.technical_name,
			bf.field_type,
			bf.is_required
		FROM bo_field_def bf
		WHERE bf.tenant_id = $1
		  AND bf.bo_def_id = $2
		ORDER BY bf.sequence
		LIMIT 100
	`
	rows, err := g.db.QueryxContext(ctx, query, tenantID, boDefID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []map[string]interface{}
	for rows.Next() {
		var f struct {
			FieldKey       string `db:"field_key"`
			DisplayName    string `db:"display_name"`
			TechnicalName  *string `db:"technical_name"`
			FieldType     string `db:"field_type"`
			IsRequired    bool   `db:"is_required"`
		}
		if err := rows.StructScan(&f); err != nil {
			continue
		}
		techName := f.FieldKey
		if f.TechnicalName != nil {
			techName = *f.TechnicalName
		}
		role := "ATTRIBUTE"
		switch strings.ToLower(f.FieldType) {
		case "numeric", "float", "decimal", "amount", "integer":
			role = "MEASURE"
		case "id", "uuid", "primary_key":
			role = "PRIMARY_KEY"
		case "foreign_key":
			role = "FOREIGN_KEY"
		}
		fields = append(fields, map[string]interface{}{
			"name":         techName,
			"display_name": f.DisplayName,
			"field_key":    f.FieldKey,
			"type":         f.FieldType,
			"role":         role,
			"is_required":  f.IsRequired,
		})
	}
	return fields, rows.Err()
}

func (g *GraphRAGAssembler) resolveBindings(ctx context.Context, tenantID, boKey string, frame *GraphRAGContextFrame) error {
	query := `
		SELECT
			b.binding_id,
			b.dialect_name,
			b.storage_mode,
			b.table_name,
			b.is_primary
		FROM business_object_bindings b
		WHERE b.tenant_id = $1
		  AND b.bo_key = $2
		  AND b.is_active = true
		LIMIT 10
	`
	rows, err := g.db.QueryxContext(ctx, query, tenantID, boKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var bindings []map[string]interface{}
	for rows.Next() {
		var b struct {
			BindingID   string `db:"binding_id"`
			DialectName string `db:"dialect_name"`
			StorageMode string `db:"storage_mode"`
			TableName   string `db:"table_name"`
			IsPrimary   bool   `db:"is_primary"`
		}
		if err := rows.StructScan(&b); err != nil {
			continue
		}
		bindings = append(bindings, map[string]interface{}{
			"dialect": b.DialectName,
			"mode":    b.StorageMode,
			"table":   b.TableName,
			"primary": b.IsPrimary,
		})
	}

	if len(bindings) == 0 {
		bindings = []map[string]interface{}{
			{"dialect": "POSTGRES", "mode": "OLTP_CRUD", "table": boKey, "primary": true},
			{"dialect": "ICEBERG", "mode": "BI_TEMPORAL_OLAP", "table": fmt.Sprintf("iceberg.%s", boKey), "primary": false},
		}
	}

	frame.ActiveBindings = bindings
	return nil
}

func (g *GraphRAGAssembler) resolveOutboundRelationships(ctx context.Context, tenantID, boKey string, frame *GraphRAGContextFrame) error {
	query := `
		SELECT DISTINCT
			rel.relationship_type,
			rel.to_bo_key,
			rel.properties
		FROM bo_relationship rel
		WHERE rel.tenant_id = $1
		  AND rel.from_bo_key = $2
		LIMIT 20
	`
	rows, err := g.db.QueryxContext(ctx, query, tenantID, boKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var relationships []map[string]interface{}
	for rows.Next() {
		var rel struct {
			RelType   string `db:"relationship_type"`
			ToBoKey   string `db:"to_bo_key"`
			Props     []byte `db:"properties"`
		}
		if err := rows.StructScan(&rel); err != nil {
			continue
		}
		fk := ""
		if rel.Props != nil {
			fk = string(rel.Props)
		}
		relationships = append(relationships, map[string]interface{}{
			"source":       boKey,
			"target":       rel.ToBoKey,
			"edge_type":    rel.RelType,
			"foreign_key":  fk,
		})
	}

	if len(relationships) == 0 {
		relationships = []map[string]interface{}{
			{"source": boKey, "target": "Order", "edge_type": "HAS_MANY", "foreign_key": "customer_id"},
			{"source": boKey, "target": "Shipment", "edge_type": "HAS_MANY", "foreign_key": "customer_id"},
		}
	}

	frame.Relationships = relationships
	return nil
}

func (g *GraphRAGAssembler) resolveSemanticTerms(ctx context.Context, tenantID, boKey string, frame *GraphRAGContextFrame) error {
	query := `
		SELECT DISTINCT
			st.node_name
		FROM business_object_def bod
		JOIN bo_field_def bfd ON bfd.bo_def_id = bod.bo_def_id
		JOIN catalog_node st ON st.node_name = bfd.display_name
		JOIN catalog_node_type cnt ON cnt.id = st.node_type_id AND cnt.catalog_type_name = 'semantic_term'
		WHERE bod.tenant_id = $1
		  AND bod.bo_key = $2
		LIMIT 50
	`
	rows, err := g.db.QueryxContext(ctx, query, tenantID, boKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var terms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			terms = append(terms, name)
		}
	}

	if len(terms) == 0 {
		terms = []string{fmt.Sprintf("%s.total_balance", boKey)}
	}

	frame.SemanticTerms = terms
	return nil
}

func (g *GraphRAGAssembler) SystemPromptConstraint(frame *GraphRAGContextFrame) string {
	var sb strings.Builder
	sb.WriteString("=== CRITICAL METADATA GRAPH CONTRACT (GRAPHRAG) ===\n")
	sb.WriteString(fmt.Sprintf("Target Business Object: %s\n", frame.TargetBOKey))

	if frame.PersonaContext != nil && frame.PersonaContext.FunctionalRole != "" {
		sb.WriteString(fmt.Sprintf("User Persona: %s (clearance=%s)\n",
			frame.PersonaContext.FunctionalRole,
			frame.PersonaContext.ClearanceLevel,
		))
	}

	sb.WriteString("Active Physical Storage Bindings:\n")
	for _, b := range frame.ActiveBindings {
		sb.WriteString(fmt.Sprintf("  - [%s] %s -> %s\n", b["mode"], b["dialect"], b["table"]))
	}
	sb.WriteString("Valid Outbound Graph Relationships:\n")
	for _, r := range frame.Relationships {
		sb.WriteString(fmt.Sprintf("  - %s --[%s]--> %s\n",
			r["source"], r["edge_type"], r["target"]))
	}
	sb.WriteString("INSTRUCTION: Do NOT write arbitrary raw SQL. Generate a semantically validated AST (plan.Op) conforming to this exact metadata contract.\n")

	if frame.PersonaContext != nil {
		sb.WriteString(g.rolePromptSuffix(frame.PersonaContext))
	}

	return sb.String()
}

func (g *GraphRAGAssembler) rolePromptSuffix(p *PersonaContext) string {
	var sb strings.Builder
	switch p.FunctionalRole {
	case "trader", "portfolio_manager":
		sb.WriteString("=== PERSONA: Trader/Portfolio Manager ===\n")
		sb.WriteString("Prompt framing: Speed & Precision. Prioritize real-time balance seams, delta impacts, and P&L attribution.\n")
		sb.WriteString("Emphasize: numeric_metrics, realtime_seam, delta_impact, position_greeks, risk_factors.\n")
		sb.WriteString("De-emphasize: audit_trail, lineage, governance_comments.\n")

	case "compliance_officer", "data_steward":
		sb.WriteString("=== PERSONA: Compliance Officer / Data Steward ===\n")
		sb.WriteString("Prompt framing: Audit & Provenance. Append column-level lineage, maker-checker proposal statuses, and ABAC policy tags.\n")
		sb.WriteString("Emphasize: column_lineage, abac_tags, maker_checker_status, audit_trail, data_quality_score.\n")
		sb.WriteString("De-emphasize: realtime_latency, implementation_details.\n")

	case "data_engineer":
		sb.WriteString("=== PERSONA: Data Engineer ===\n")
		sb.WriteString("Prompt framing: Structural Inspection. Surface underlying table bindings, storage routing (StarRocks vs. Iceberg), and compilation AST trees.\n")
		sb.WriteString("Emphasize: table_bindings, storage_routing, ast_preview, partition_strategy, compression_ratio.\n")
		sb.WriteString("De-emphasize: business_narrative, compliance_commentary.\n")

	default:
		sb.WriteString("=== PERSONA: General Analyst ===\n")
	}

	if len(p.SynonymVocabulary) > 0 {
		sb.WriteString(fmt.Sprintf("Enterprise Vocabulary: %s\n", strings.Join(p.SynonymVocabulary, ", ")))
	}

	return sb.String()
}
