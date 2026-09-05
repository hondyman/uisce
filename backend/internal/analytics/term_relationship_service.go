package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// Standard semantic relationship edge type constants
const (
	EdgeTypeIsSpecializationOf = "IS_SPECIALIZATION_OF"
	EdgeTypeIsGeneralizationOf = "IS_GENERALIZATION_OF"
	EdgeTypeIsPeerIdentifierOf = "IS_PEER_IDENTIFIER_OF"
	EdgeTypeDifferentiatedFrom = "DIFFERENTIATED_FROM"
	EdgeTypeRelatesTo          = "RELATES_TO"
)

// RelatedTermInfo represents detailed intelligence on an associated business/semantic node.
type RelatedTermInfo struct {
	TermID               string                 `json:"term_id"`
	TermName             string                 `json:"term_name"`
	QualifiedPath        string                 `json:"qualified_path"`
	Category             string                 `json:"category,omitempty"`
	DataType             string                 `json:"data_type,omitempty"`
	Domain               string                 `json:"domain,omitempty"`
	Role                 string                 `json:"role,omitempty"` // Specialization, Parent, Peer, Sibling
	RelationshipType     string                 `json:"relationship_type"`
	DifferentiationNotes string                 `json:"differentiation_notes,omitempty"`
	FormatPattern        string                 `json:"format_pattern,omitempty"`
	Standard             string                 `json:"standard,omitempty"`
	Confidence           float64                `json:"confidence"`
	IsGoldCopy           bool                   `json:"is_gold_copy"`
	Properties           map[string]interface{} `json:"properties,omitempty"`
}

// TermDisambiguation provides the focal term along with its peers, specializations, and differentiator rationale.
type TermDisambiguation struct {
	PrimaryTerm           RelatedTermInfo   `json:"primary_term"`
	RelatedTerms          []RelatedTermInfo `json:"related_terms"`
	DifferentiatorSummary string            `json:"differentiator_summary"`
	DomainScope           string            `json:"domain_scope,omitempty"`
}

// AITermDefinition represents a high-density, prompt-optimized node for LLMs.
type AITermDefinition struct {
	TermName             string            `json:"term_name"`
	QualifiedPath        string            `json:"qualified_path"`
	Domain               string            `json:"domain"`
	Category             string            `json:"category"`
	DataType             string            `json:"data_type"`
	Standard             string            `json:"standard,omitempty"`
	FormatPattern        string            `json:"format_pattern,omitempty"`
	Definition           string            `json:"definition"`
	DifferentiatorNotes  string            `json:"differentiator_notes"`
	ParentTerm           string            `json:"parent_term,omitempty"`
	PeerIdentifiers      []string          `json:"peer_identifiers,omitempty"`
	SpecializedSubTerms  []string          `json:"specialized_sub_terms,omitempty"`
	DisambiguationGuidance string          `json:"disambiguation_guidance"`
}

// AIGraphEdge represents an associative or hierarchical connection in the AI context.
type AIGraphEdge struct {
	SourceTerm   string `json:"source_term"`
	Predicate    string `json:"predicate"`
	TargetTerm   string `json:"target_term"`
	Explanation  string `json:"explanation,omitempty"`
}

// AIContextPayload is the complete package sent to downstream AI agents / LLM prompt generators.
type AIContextPayload struct {
	Version            string                 `json:"version"`
	GeneratedAt        time.Time              `json:"generated_at"`
	TenantID           string                 `json:"tenant_id"`
	Domain             string                 `json:"domain,omitempty"`
	TermCount          int                    `json:"term_count"`
	Terms              []AITermDefinition     `json:"terms"`
	TaxonomyEdges      []AIGraphEdge          `json:"taxonomy_edges"`
	ActiveDirectives   []string               `json:"active_directives"`
	PromptContextBlock string                 `json:"prompt_context_block"`
	JSONLDSchema       map[string]interface{} `json:"json_ld_schema"`
}

// TermRelationshipService manages term associations, graph reasoning, and AI context export.
type TermRelationshipService struct {
	db *sqlx.DB
}

// NewTermRelationshipService creates a new TermRelationshipService instance.
func NewTermRelationshipService(db *sqlx.DB) *TermRelationshipService {
	return &TermRelationshipService{db: db}
}

// GetGoldTenantID retrieves the UUID of the master gold copy tenant.
func (s *TermRelationshipService) GetGoldTenantID(ctx context.Context) (string, error) {
	if s.db == nil {
		return "", nil
	}
	var goldID string
	err := s.db.GetContext(ctx, &goldID, "SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1")
	if err != nil || goldID == "" {
		// Fallback to earliest created tenant
		_ = s.db.GetContext(ctx, &goldID, "SELECT id FROM public.tenants ORDER BY created_at LIMIT 1")
	}
	return goldID, nil
}

// GetRelatedTerms retrieves related and differentiating terms for a given term ID or Name.
func (s *TermRelationshipService) GetRelatedTerms(ctx context.Context, tenantID, datasourceID, termIDOrName string) (*TermDisambiguation, error) {
	if s.db == nil {
		return s.buildFallbackDisambiguation(termIDOrName), nil
	}
	goldTenantID, _ := s.GetGoldTenantID(ctx)

	// 1. Resolve target primary term node
	var primaryNode struct {
		ID            string         `db:"id"`
		TenantID      string         `db:"tenant_id"`
		NodeName      string         `db:"node_name"`
		QualifiedPath string         `db:"qualified_path"`
		Description   sql.NullString `db:"description"`
		Properties    []byte         `db:"properties"`
	}

	isUUID := strings.Contains(termIDOrName, "-") && len(termIDOrName) == 36
	var err error
	if isUUID {
		query := `
			SELECT id, tenant_id, node_name, qualified_path, description, properties
			FROM catalog_node
			WHERE id = $1
			  AND (tenant_id = $2 OR tenant_id = $3 OR tenant_id = '00000000-0000-0000-0000-000000000000' OR $2 = '')
			ORDER BY (tenant_id = $2) DESC, (tenant_id = $3) DESC, created_at DESC
			LIMIT 1
		`
		err = s.db.GetContext(ctx, &primaryNode, query, termIDOrName, tenantID, goldTenantID)
		if err != nil {
			// Fallback: search by ID regardless of tenant
			err = s.db.GetContext(ctx, &primaryNode, "SELECT id, tenant_id, node_name, qualified_path, description, properties FROM catalog_node WHERE id = $1 LIMIT 1", termIDOrName)
		}
	} else {
		query := `
			SELECT id, tenant_id, node_name, qualified_path, description, properties
			FROM catalog_node
			WHERE (UPPER(node_name) = UPPER($1) OR UPPER(qualified_path) LIKE '%' || UPPER($1))
			  AND (tenant_id = $2 OR tenant_id = $3 OR tenant_id = '00000000-0000-0000-0000-000000000000' OR $2 = '')
			ORDER BY (tenant_id = $2) DESC, (tenant_id = $3) DESC, created_at DESC
			LIMIT 1
		`
		err = s.db.GetContext(ctx, &primaryNode, query, strings.TrimSpace(termIDOrName), tenantID, goldTenantID)
		if err != nil {
			// Fallback: search without tenant restriction
			err = s.db.GetContext(ctx, &primaryNode, "SELECT id, tenant_id, node_name, qualified_path, description, properties FROM catalog_node WHERE UPPER(node_name) = UPPER($1) ORDER BY created_at DESC LIMIT 1", strings.TrimSpace(termIDOrName))
		}
	}

	if err != nil {
		// If not found in catalog, construct fallback from knowledge base heuristics
		return s.buildFallbackDisambiguation(termIDOrName), nil
	}

	var primaryProps map[string]interface{}
	if len(primaryNode.Properties) > 0 {
		_ = json.Unmarshal(primaryNode.Properties, &primaryProps)
	}

	primaryInfo := s.nodeToRelatedTermInfo(
		primaryNode.ID,
		primaryNode.NodeName,
		primaryNode.QualifiedPath,
		primaryNode.Description.String,
		primaryProps,
		"Primary",
		"",
		primaryNode.TenantID == goldTenantID,
		1.0,
	)

	// 2. Query incoming & outgoing relationship edges
	type EdgeRow struct {
		EdgeID       string         `db:"id"`
		SourceID     string         `db:"source_id"`
		TargetID     string         `db:"target_id"`
		EdgeTypeName string         `db:"edge_type_name"`
		Properties   []byte         `db:"properties"`
		OtherNodeID  string         `db:"other_node_id"`
		OtherName    string         `db:"other_name"`
		OtherPath    string         `db:"other_path"`
		OtherDesc    sql.NullString `db:"other_desc"`
		OtherProps   []byte         `db:"other_props"`
		OtherTenant  string         `db:"other_tenant"`
		IsOutgoing   bool           `db:"is_outgoing"`
	}

	edgeQuery := `
		WITH combined_edges AS (
			SELECT 
				ce.id, ce.source_id, ce.target_id, ce.edge_type_name, ce.properties, ce.tenant_id,
				ROW_NUMBER() OVER (
					PARTITION BY ce.source_id, ce.target_id, ce.edge_type_name 
					ORDER BY CASE WHEN ce.tenant_id = $2 THEN 1 ELSE 2 END
				) AS precedence_rank
			FROM catalog_edge ce
			WHERE (ce.source_id = $1 OR ce.target_id = $1)
			  AND (ce.tenant_id = $2 OR ce.tenant_id = $3 OR ce.tenant_id = '00000000-0000-0000-0000-000000000000' OR $2 = '')
			  AND ce.is_active = true
			  AND NOT EXISTS (
				  SELECT 1 FROM catalog_edge_rejection_store r
				  WHERE ($2 != '' AND r.tenant_id = CAST($2 AS UUID))
				    AND ( (r.source_node_id = ce.source_id AND r.rejected_target_id = ce.target_id) OR (r.source_node_id = ce.target_id AND r.rejected_target_id = ce.source_id) )
			  )
		)
		SELECT 
			ce.id, ce.source_id, ce.target_id, ce.edge_type_name, ce.properties,
			cn.id as other_node_id, cn.node_name as other_name, cn.qualified_path as other_path,
			cn.description as other_desc, cn.properties as other_props, cn.tenant_id as other_tenant,
			(ce.source_id = $1) as is_outgoing
		FROM combined_edges ce
		JOIN catalog_node cn ON (
			CASE 
				WHEN ce.source_id = $1 THEN ce.target_id = cn.id
				ELSE ce.source_id = cn.id
			END
		)
		WHERE ce.precedence_rank = 1
		  AND (cn.tenant_id = $2 OR cn.tenant_id = $3 OR cn.tenant_id = '00000000-0000-0000-0000-000000000000' OR $2 = '')
		ORDER BY ce.edge_type_name ASC
	`

	var edgeRows []EdgeRow
	err = s.db.SelectContext(ctx, &edgeRows, edgeQuery, primaryNode.ID, tenantID, goldTenantID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Error querying related term edges for %s: %v", primaryNode.ID, err)
	}

	relatedMap := make(map[string]RelatedTermInfo)
	var diffSummaries []string

	for _, row := range edgeRows {
		var edgeProps map[string]interface{}
		if len(row.Properties) > 0 {
			_ = json.Unmarshal(row.Properties, &edgeProps)
		}

		var targetNodeProps map[string]interface{}
		if len(row.OtherProps) > 0 {
			_ = json.Unmarshal(row.OtherProps, &targetNodeProps)
		}

		// Determine role relative to primary
		role := "Related"
		edgeType := row.EdgeTypeName
		diffNote := ""

		if d, ok := edgeProps["differentiation"].(string); ok && d != "" {
			diffNote = d
		} else if kd, ok := edgeProps["key_distinction"].(string); ok && kd != "" {
			diffNote = kd
		} else if dn, ok := targetNodeProps["differentiator_notes"].(string); ok && dn != "" {
			diffNote = dn
		}

		switch edgeType {
		case EdgeTypeIsSpecializationOf:
			if row.IsOutgoing {
				role = "Parent (Generalization)"
			} else {
				role = "Specialization (Sub-type)"
			}
		case EdgeTypeIsGeneralizationOf:
			if row.IsOutgoing {
				role = "Specialization (Sub-type)"
			} else {
				role = "Parent (Generalization)"
			}
		case EdgeTypeIsPeerIdentifierOf:
			role = "Peer Symbology / Alternate ID"
		case EdgeTypeDifferentiatedFrom:
			role = "Differentiated Alternative"
		case EdgeTypeRelatesTo:
			role = "Associated Concept"
		}

		if diffNote != "" {
			diffSummaries = append(diffSummaries, fmt.Sprintf("• %s vs %s: %s", primaryNode.NodeName, row.OtherName, diffNote))
		}

		info := s.nodeToRelatedTermInfo(
			row.OtherNodeID,
			row.OtherName,
			row.OtherPath,
			row.OtherDesc.String,
			targetNodeProps,
			role,
			edgeType,
			row.OtherTenant == goldTenantID,
			0.90,
		)
		if diffNote != "" {
			info.DifferentiationNotes = diffNote
		}
		relatedMap[row.OtherNodeID] = info
	}

	// 3. If no edges exist in DB yet, supplement with built-in heuristic associations
	if len(relatedMap) == 0 {
		heuristicDisambiguation := s.buildFallbackDisambiguation(primaryNode.NodeName)
		if heuristicDisambiguation != nil && len(heuristicDisambiguation.RelatedTerms) > 0 {
			for _, t := range heuristicDisambiguation.RelatedTerms {
				relatedMap[t.TermName] = t
			}
			if len(diffSummaries) == 0 && heuristicDisambiguation.DifferentiatorSummary != "" {
				diffSummaries = append(diffSummaries, heuristicDisambiguation.DifferentiatorSummary)
			}
		}
	}

	var relatedList []RelatedTermInfo
	for _, term := range relatedMap {
		relatedList = append(relatedList, term)
	}

	summary := strings.Join(diffSummaries, "\n")
	if summary == "" {
		summary = fmt.Sprintf("Standard domain term in category %s.", primaryInfo.Category)
	}

	return &TermDisambiguation{
		PrimaryTerm:           primaryInfo,
		RelatedTerms:          relatedList,
		DifferentiatorSummary: summary,
		DomainScope:           primaryInfo.Domain,
	}, nil
}

// BuildAIContext creates the LLM prompt payload, JSON-LD schema, and taxonomy edges for a set of terms.
func (s *TermRelationshipService) BuildAIContext(ctx context.Context, tenantID, datasourceID string, termIDs []string, domain string) (*AIContextPayload, error) {
	if s.db == nil {
		return s.buildFallbackAIContext(domain), nil
	}
	goldTenantID, _ := s.GetGoldTenantID(ctx)

	// Fetch all candidate business terms if no specific IDs passed
	var nodes []struct {
		ID            string         `db:"id"`
		TenantID      string         `db:"tenant_id"`
		NodeName      string         `db:"node_name"`
		QualifiedPath string         `db:"qualified_path"`
		Description   sql.NullString `db:"description"`
		Properties    []byte         `db:"properties"`
	}

	var err error
	if len(termIDs) > 0 {
		query, args, qErr := sqlx.In(`
			SELECT id, tenant_id, node_name, qualified_path, description, properties
			FROM catalog_node
			WHERE id IN (?) AND (tenant_id = ? OR tenant_id = ?)
		`, termIDs, tenantID, goldTenantID)
		if qErr != nil {
			return nil, qErr
		}
		query = s.db.Rebind(query)
		err = s.db.SelectContext(ctx, &nodes, query, args...)
	} else {
		query := `
			SELECT id, tenant_id, node_name, qualified_path, description, properties
			FROM catalog_node
			WHERE node_type_id = $1 AND (tenant_id = $2 OR tenant_id = $3)
			ORDER BY node_name ASC
			LIMIT 100
		`
		err = s.db.SelectContext(ctx, &nodes, query, BusinessTermNodeTypeID, tenantID, goldTenantID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch nodes for AI context: %w", err)
	}

	var aiTerms []AITermDefinition
	var promptBuilder strings.Builder

	promptBuilder.WriteString("### SEMANTIC ONTOLOGY & NODE DIFFERENTIATION GUIDE\n")
	promptBuilder.WriteString("Use these precise definitions and relationship rules to distinguish and map business concepts correctly:\n\n")

	for _, node := range nodes {
		var props map[string]interface{}
		if len(node.Properties) > 0 {
			_ = json.Unmarshal(node.Properties, &props)
		}

		termDef := AITermDefinition{
			TermName:             node.NodeName,
			QualifiedPath:        node.QualifiedPath,
			Definition:           node.Description.String,
			Domain:               getRelStringProp(props, "domain", "Enterprise Core"),
			Category:             getRelStringProp(props, "category", "General"),
			DataType:             getRelStringProp(props, "data_type", "string"),
			Standard:             getRelStringProp(props, "standard", ""),
			FormatPattern:        getRelStringProp(props, "format_pattern", ""),
			DifferentiatorNotes:  getRelStringProp(props, "differentiator_notes", ""),
		}

		// Query relationship context for this node
		disambig, _ := s.GetRelatedTerms(ctx, tenantID, datasourceID, node.ID)
		if disambig != nil {
			for _, r := range disambig.RelatedTerms {
				if r.Role == "Parent (Generalization)" {
					termDef.ParentTerm = r.TermName
				} else if r.Role == "Peer Symbology / Alternate ID" {
					termDef.PeerIdentifiers = append(termDef.PeerIdentifiers, r.TermName)
				} else if strings.Contains(r.Role, "Specialization") {
					termDef.SpecializedSubTerms = append(termDef.SpecializedSubTerms, r.TermName)
				}
			}
			termDef.DisambiguationGuidance = disambig.DifferentiatorSummary
		}

		aiTerms = append(aiTerms, termDef)

		// Append to prompt markdown block
		promptBuilder.WriteString(fmt.Sprintf("#### Term: `%s`\n", termDef.TermName))
		promptBuilder.WriteString(fmt.Sprintf("- **Definition**: %s\n", termDef.Definition))
		promptBuilder.WriteString(fmt.Sprintf("- **Domain/Category**: %s / %s (Data Type: %s)\n", termDef.Domain, termDef.Category, termDef.DataType))
		if termDef.Standard != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Standard / Format**: %s (`%s`)\n", termDef.Standard, termDef.FormatPattern))
		}
		if termDef.ParentTerm != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Parent Concept**: `%s`\n", termDef.ParentTerm))
		}
		if len(termDef.SpecializedSubTerms) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Specialized Variants**: %s\n", strings.Join(termDef.SpecializedSubTerms, ", ")))
		}
		if len(termDef.PeerIdentifiers) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Peer Symbologies**: %s\n", strings.Join(termDef.PeerIdentifiers, ", ")))
		}
		if termDef.DifferentiatorNotes != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Differentiator & Usage**: %s\n", termDef.DifferentiatorNotes))
		}
		promptBuilder.WriteString("\n")
	}

	// Fetch taxonomy edges
	var edges []AIGraphEdge
	edgeQuery := `
		SELECT 
			c1.node_name as source_term, 
			ce.edge_type_name as predicate, 
			c2.node_name as target_term,
			COALESCE(ce.properties->>'differentiation', ce.properties->>'key_distinction', '') as explanation
		FROM catalog_edge ce
		JOIN catalog_node c1 ON ce.source_id = c1.id
		JOIN catalog_node c2 ON ce.target_id = c2.id
		WHERE (ce.tenant_id = $1 OR ce.tenant_id = $2)
	`
	_ = s.db.SelectContext(ctx, &edges, edgeQuery, tenantID, goldTenantID)

	// Construct JSON-LD Schema
	jsonLD := map[string]interface{}{
		"@context": map[string]interface{}{
			"schema": "http://schema.org/",
			"fibo":   "https://spec.edmcouncil.org/fibo/ontology/",
			"terms":  "http://uisce.io/ontology/terms#",
		},
		"@graph": aiTerms,
	}

	directives := []string{
		"Do NOT substitute custodial_account_code when allocation_account_code is requested in trade allocation contexts.",
		"When security symbology is ambiguous, inspect IS_PEER_IDENTIFIER_OF edges to resolve CUSIP/SEDOL cross-references.",
		"Enforce strict distinction between trade_date (market agreement) and settlement_date (cash delivery).",
		"LEI identifies corporate legal entity issuers/counterparties, NOT specific traded instruments (ISIN/CUSIP/SEDOL).",
	}

	return &AIContextPayload{
		Version:            "1.0.0",
		GeneratedAt:        time.Now().UTC(),
		TenantID:           tenantID,
		Domain:             domain,
		TermCount:          len(aiTerms),
		Terms:              aiTerms,
		TaxonomyEdges:      edges,
		ActiveDirectives:   directives,
		PromptContextBlock: promptBuilder.String(),
		JSONLDSchema:       jsonLD,
	}, nil
}

// SuggestRelatedTermsForColumn provides candidate related business terms based on column name & table name.
func (s *TermRelationshipService) SuggestRelatedTermsForColumn(ctx context.Context, tenantID, datasourceID, columnName, entityName string) ([]RelatedTermInfo, error) {
	colNorm := strings.ToUpper(strings.TrimSpace(columnName))
	entityNorm := strings.ToUpper(strings.TrimSpace(entityName))

	// Match heuristics
	var targetFamilyTerm string
	if strings.Contains(colNorm, "ACCT") || strings.Contains(colNorm, "ACCOUNT") {
		if strings.Contains(colNorm, "ALLOC") {
			targetFamilyTerm = "Allocation Account Code"
		} else if strings.Contains(colNorm, "CUST") || strings.Contains(colNorm, "BANK") {
			targetFamilyTerm = "Custodial Account Code"
		} else if strings.Contains(colNorm, "GL") || strings.Contains(colNorm, "LEDGER") {
			targetFamilyTerm = "GL Account Code"
		} else if strings.Contains(colNorm, "CLIENT") || strings.Contains(colNorm, "CUST_NO") {
			targetFamilyTerm = "Client Account Number"
		} else {
			targetFamilyTerm = "Account Code"
		}
	} else if strings.Contains(colNorm, "ISIN") {
		targetFamilyTerm = "ISIN"
	} else if strings.Contains(colNorm, "CUSIP") {
		targetFamilyTerm = "CUSIP"
	} else if strings.Contains(colNorm, "SEDOL") {
		targetFamilyTerm = "SEDOL"
	} else if strings.Contains(colNorm, "FIGI") || strings.Contains(colNorm, "BBG") {
		targetFamilyTerm = "FIGI"
	} else if strings.Contains(colNorm, "LEI") {
		targetFamilyTerm = "LEI"
	} else if strings.Contains(colNorm, "TICKER") || strings.Contains(colNorm, "SYMBOL") {
		targetFamilyTerm = "Primary Ticker"
	} else if strings.Contains(colNorm, "TRADE") && strings.Contains(colNorm, "DT") || strings.Contains(colNorm, "DATE") {
		targetFamilyTerm = "Trade Date"
	} else if strings.Contains(colNorm, "SETTLE") {
		targetFamilyTerm = "Settlement Date"
	}

	if targetFamilyTerm != "" {
		disambig, err := s.GetRelatedTerms(ctx, tenantID, datasourceID, targetFamilyTerm)
		if err == nil && disambig != nil {
			var results []RelatedTermInfo
			results = append(results, disambig.PrimaryTerm)
			results = append(results, disambig.RelatedTerms...)
			return results, nil
		}
	}

	// Default fallback
	disambig := s.buildFallbackDisambiguation(entityNorm + "_" + colNorm)
	if disambig != nil {
		var results []RelatedTermInfo
		results = append(results, disambig.PrimaryTerm)
		results = append(results, disambig.RelatedTerms...)
		return results, nil
	}

	return nil, nil
}

func isUUIDString(s string) bool {
	return strings.Contains(s, "-") && len(s) == 36
}

// CreateTermRelationship writes a semantic relationship edge into catalog_edge.
func (s *TermRelationshipService) CreateTermRelationship(ctx context.Context, tenantID, datasourceID, sourceTermID, targetTermID, edgeTypeName string, props map[string]interface{}) (string, error) {
	if s.db == nil {
		return uuid.New().String(), nil
	}

	if tenantID == "" {
		tenantID, _ = s.GetGoldTenantID(ctx)
		if tenantID == "" {
			tenantID = "00000000-0000-0000-0000-000000000000"
		}
	}
	if edgeTypeName == "" {
		edgeTypeName = EdgeTypeIsSpecializationOf
	}

	// 1. Resolve source node UUID if passed as name
	if !isUUIDString(sourceTermID) {
		var resolvedSourceID string
		_ = s.db.GetContext(ctx, &resolvedSourceID, `
			SELECT id FROM catalog_node 
			WHERE UPPER(node_name) = UPPER($1) 
			ORDER BY (tenant_id = $2) DESC, created_at DESC 
			LIMIT 1
		`, sourceTermID, tenantID)
		if resolvedSourceID != "" {
			sourceTermID = resolvedSourceID
		}
	}

	// 2. Resolve target node UUID if passed as name
	if !isUUIDString(targetTermID) {
		var resolvedTargetID string
		_ = s.db.GetContext(ctx, &resolvedTargetID, `
			SELECT id FROM catalog_node 
			WHERE UPPER(node_name) = UPPER($1) 
			ORDER BY (tenant_id = $2) DESC, created_at DESC 
			LIMIT 1
		`, targetTermID, tenantID)

		if resolvedTargetID != "" {
			targetTermID = resolvedTargetID
		} else {
			// Create target node in catalog_node if it doesn't exist yet
			newTargetID := uuid.New().String()
			_, err := s.db.ExecContext(ctx, `
				INSERT INTO catalog_node (
					id, tenant_id, node_type_id, node_name, qualified_path, description, is_active, created_at, updated_at
				) VALUES (
					$1, $2, '21645d21-de5f-4feb-af99-99273ea75626', $3, $3, $3, true, NOW(), NOW()
				)
				ON CONFLICT DO NOTHING
			`, newTargetID, tenantID, targetTermID)
			if err == nil {
				targetTermID = newTargetID
			}
		}
	}

	if sourceTermID == "" || targetTermID == "" {
		return "", fmt.Errorf("could not resolve source or target term node")
	}

	// Resolve edge_type_id from catalog_edge_type (insert if missing so legacy
	// predicates still validate). This matches the canonical schema used by
	// GlossaryHandler.ListEdges / CreateEdge (source_node_id, target_node_id,
	// edge_type_id) so newly approved relationships surface in subsequent
	// edge queries.
	edgeTypeID, err := s.resolveOrCreateEdgeType(ctx, tenantID, edgeTypeName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve edge type %q: %w", edgeTypeName, err)
	}

	edgeID := uuid.New().String()
	if props == nil {
		props = make(map[string]interface{})
	}
	propsJSON, _ := json.Marshal(props)

	var nullDsID *string
	if datasourceID != "" {
		nullDsID = &datasourceID
	}

	query := `
		INSERT INTO catalog_edge (
			id, tenant_id, tenant_datasource_id,
			source_node_id, target_node_id,
			edge_type_id, edge_type_name,
			properties, is_active, created_at, updated_at
		) VALUES (
			$1, $2::uuid, $3,
			$4::uuid, $5::uuid,
			$6::uuid, $7,
			$8::jsonb, true, NOW(), NOW()
		)
		ON CONFLICT (tenant_id, source_node_id, target_node_id, edge_type_id)
		DO UPDATE SET
			edge_type_name = EXCLUDED.edge_type_name,
			properties = EXCLUDED.properties,
			is_active = true,
			updated_at = NOW()
		RETURNING id
	`

	var returnedID string
	err = s.db.QueryRowContext(ctx, query, edgeID, tenantID, nullDsID, sourceTermID, targetTermID, edgeTypeID, edgeTypeName, string(propsJSON)).Scan(&returnedID)
	if err != nil {
		// Fallback for simple UUID string types without ::uuid cast
		fallbackQuery := `
			INSERT INTO catalog_edge (
				id, tenant_id, tenant_datasource_id,
				source_node_id, target_node_id,
				edge_type_id, edge_type_name,
				properties, is_active, created_at, updated_at
			) VALUES (
				$1, $2, $3,
				$4, $5,
				$6, $7,
				$8::jsonb, true, NOW(), NOW()
			)
			ON CONFLICT DO NOTHING
			RETURNING id
		`
		_ = s.db.QueryRowContext(ctx, fallbackQuery, edgeID, tenantID, nullDsID, sourceTermID, targetTermID, edgeTypeID, edgeTypeName, string(propsJSON)).Scan(&returnedID)
		if returnedID == "" {
			returnedID = edgeID
		}
	}
	return returnedID, nil
}

// resolveOrCreateEdgeType returns the UUID of a catalog_edge_type row for the
// given edge type name, inserting a tenant-scoped row if the name is unknown.
// This mirrors GlossaryHandler.CreateEdge so semantic-relationship approvals
// populate the same lookup table that ListEdges joins against.
func (s *TermRelationshipService) resolveOrCreateEdgeType(ctx context.Context, tenantID, edgeTypeName string) (string, error) {
	if s.db == nil {
		return uuid.Nil.String(), nil
	}
	if edgeTypeName == "" {
		edgeTypeName = EdgeTypeIsSpecializationOf
	}

	var edgeTypeID string
	if err := s.db.GetContext(ctx, &edgeTypeID, `
		SELECT id FROM catalog_edge_type
		WHERE edge_type_name = $1
		ORDER BY (tenant_id = $2) DESC
		LIMIT 1
	`, edgeTypeName, tenantID); err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if edgeTypeID != "" {
		return edgeTypeID, nil
	}

	if err := s.db.GetContext(ctx, &edgeTypeID, `
		INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET edge_type_name = EXCLUDED.edge_type_name
		RETURNING id
	`, tenantID, edgeTypeName); err != nil {
		return "", err
	}
	return edgeTypeID, nil
}

// RejectionRecord represents an entry in catalog_edge_rejection_store.
type RejectionRecord struct {
	RejectionID      string    `json:"rejection_id" db:"rejection_id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	SourceNodeID     string    `json:"source_node_id" db:"source_node_id"`
	RejectedTargetID string    `json:"rejected_target_id" db:"rejected_target_id"`
	EdgeTypeID       string    `json:"edge_type_id" db:"edge_type_id"`
	RejectedBy       string    `json:"rejected_by" db:"rejected_by"`
	Reason           string    `json:"reason" db:"reason"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// RecordRejection records a user or system rejection into catalog_edge_rejection_store.
func (s *TermRelationshipService) RecordRejection(ctx context.Context, tenantID, sourceNodeID, rejectedTargetID, edgeTypeID, rejectedBy, reason string) error {
	if s.db == nil {
		return nil
	}

	if tenantID == "" {
		tenantID, _ = s.GetGoldTenantID(ctx)
		if tenantID == "" {
			tenantID = "00000000-0000-0000-0000-000000000000"
		}
	}
	if rejectedBy == "" {
		rejectedBy = "user"
	}
	if edgeTypeID == "" {
		edgeTypeID = "3be9d6ae-1598-4628-a3dd-b606921a9193" // Default mapping edge type
	}

	// Resolve node UUIDs if names were supplied
	if !isUUIDString(sourceNodeID) {
		var foundID string
		_ = s.db.GetContext(ctx, &foundID, "SELECT id FROM catalog_node WHERE UPPER(node_name) = UPPER($1) LIMIT 1", sourceNodeID)
		if foundID != "" {
			sourceNodeID = foundID
		}
	}
	if !isUUIDString(rejectedTargetID) {
		var foundID string
		_ = s.db.GetContext(ctx, &foundID, "SELECT id FROM catalog_node WHERE UPPER(node_name) = UPPER($1) LIMIT 1", rejectedTargetID)
		if foundID != "" {
			rejectedTargetID = foundID
		}
	}

	if !isUUIDString(sourceNodeID) || !isUUIDString(rejectedTargetID) {
		logging.GetLogger().Sugar().Warnf("Skipping rejection persistence for non-resolvable node: %s -> %s", sourceNodeID, rejectedTargetID)
		return nil
	}

	query := `
		INSERT INTO catalog_edge_rejection_store (
			rejection_id, tenant_id, source_node_id, rejected_target_id, edge_type_id, rejected_by, reason, created_at
		) VALUES (
			gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, NOW()
		)
		ON CONFLICT (tenant_id, source_node_id, rejected_target_id, edge_type_id)
		DO UPDATE SET
			rejected_by = EXCLUDED.rejected_by,
			reason = EXCLUDED.reason,
			created_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query, tenantID, sourceNodeID, rejectedTargetID, edgeTypeID, rejectedBy, reason)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed recording edge rejection in DB: %v", err)
	}
	return nil
}

// ListRejections returns all active edge rejections for a tenant.
func (s *TermRelationshipService) ListRejections(ctx context.Context, tenantID string) ([]RejectionRecord, error) {
	if s.db == nil {
		return nil, nil
	}

	var records []RejectionRecord
	query := `
		SELECT rejection_id, tenant_id, source_node_id, rejected_target_id, edge_type_id, rejected_by, COALESCE(reason, '') as reason, created_at
		FROM catalog_edge_rejection_store
		WHERE tenant_id = $1::uuid
		ORDER BY created_at DESC
	`
	err := s.db.SelectContext(ctx, &records, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed listing edge rejections: %w", err)
	}
	return records, nil
}

// DeleteRejection deletes a rejection entry, allowing the suggestion to reappear.
func (s *TermRelationshipService) DeleteRejection(ctx context.Context, tenantID, rejectionID string) error {
	if s.db == nil {
		return nil
	}

	query := `DELETE FROM catalog_edge_rejection_store WHERE rejection_id = $1::uuid AND tenant_id = $2::uuid`
	_, err := s.db.ExecContext(ctx, query, rejectionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed deleting edge rejection: %w", err)
	}
	return nil
}

// nodeToRelatedTermInfo formats raw catalog node data into a strongly typed RelatedTermInfo.
func (s *TermRelationshipService) nodeToRelatedTermInfo(
	id, name, path, desc string,
	props map[string]interface{},
	role, edgeType string,
	isGoldCopy bool,
	confidence float64,
) RelatedTermInfo {
	return RelatedTermInfo{
		TermID:               id,
		TermName:             name,
		QualifiedPath:        path,
		Category:             getRelStringProp(props, "category", "General"),
		DataType:             getRelStringProp(props, "data_type", "string"),
		Domain:               getRelStringProp(props, "domain", "Core"),
		Role:                 role,
		RelationshipType:     edgeType,
		DifferentiationNotes: getRelStringProp(props, "differentiator_notes", desc),
		FormatPattern:        getRelStringProp(props, "format_pattern", ""),
		Standard:             getRelStringProp(props, "standard", ""),
		Confidence:           confidence,
		IsGoldCopy:           isGoldCopy,
		Properties:           props,
	}
}

// buildFallbackDisambiguation builds static knowledge heuristics for common financial/enterprise families.
func (s *TermRelationshipService) buildFallbackDisambiguation(termName string) *TermDisambiguation {
	norm := strings.ToUpper(strings.TrimSpace(termName))

	// 1. Account Family
	if strings.Contains(norm, "ACCOUNT") || strings.Contains(norm, "ACCT") {
		primary := RelatedTermInfo{
			TermName:             "Account Code",
			Category:             "Account & Entity",
			Domain:               "Accounting & Finance",
			Role:                 "Parent Concept",
			DifferentiationNotes: "General umbrella identifier representing a financial, institutional, or retail account.",
			Confidence:           0.95,
		}

		related := []RelatedTermInfo{
			{
				TermName:             "Allocation Account Code",
				Category:             "Account & Entity",
				Domain:               "Trading & Portfolio Management",
				Role:                 "Specialization (Sub-type)",
				RelationshipType:     EdgeTypeIsSpecializationOf,
				DifferentiationNotes: "Sub-account for post-trade block execution fills and sleeve apportionment.",
				Confidence:           0.92,
			},
			{
				TermName:             "Custodial Account Code",
				Category:             "Account & Entity",
				Domain:               "Custody & Settlement",
				Role:                 "Specialization (Sub-type)",
				RelationshipType:     EdgeTypeIsSpecializationOf,
				DifferentiationNotes: "External bank depository safekeeping account (e.g. State Street, BNY Mellon) where assets legally reside.",
				Confidence:           0.90,
			},
			{
				TermName:             "GL Account Code",
				Category:             "Account & Entity",
				Domain:               "Financial Reporting",
				Role:                 "Specialization (Sub-type)",
				RelationshipType:     EdgeTypeIsSpecializationOf,
				DifferentiationNotes: "General Ledger chart-of-accounts number for double-entry trial balance & financial statements.",
				Confidence:           0.88,
			},
			{
				TermName:             "Client Account Number",
				Category:             "Account & Entity",
				Domain:               "Client Servicing",
				Role:                 "Specialization (Sub-type)",
				RelationshipType:     EdgeTypeIsSpecializationOf,
				DifferentiationNotes: "Client master identity account for client statements and KYC.",
				Confidence:           0.85,
			},
		}

		return &TermDisambiguation{
			PrimaryTerm:           primary,
			RelatedTerms:          related,
			DifferentiatorSummary: "• Account Code is the general parent.\n• Allocation Account Code routes post-trade block trade fills.\n• Custodial Account Code identifies external bank depositories.\n• GL Account Code is for accounting ledger entries.",
			DomainScope:           "Accounting & Finance",
		}
	}

	// 2. Symbology Family (ISIN, CUSIP, SEDOL, FIGI, LEI)
	if strings.Contains(norm, "ISIN") || strings.Contains(norm, "CUSIP") || strings.Contains(norm, "SEDOL") || strings.Contains(norm, "FIGI") || strings.Contains(norm, "LEI") {
		primary := RelatedTermInfo{
			TermName:             "ISIN",
			Category:             "Symbology & Identifiers",
			Domain:               "Capital Markets",
			Role:                 "Global Standard (ISO 6166)",
			Standard:             "ISO 6166",
			FormatPattern:        "^[A-Z]{2}[A-Z0-9]{9}[0-9]$",
			DifferentiationNotes: "12-character international security identifier with 2-char country prefix + 9-char local code + 1 check digit.",
			Confidence:           0.98,
		}

		related := []RelatedTermInfo{
			{
				TermName:             "CUSIP",
				Category:             "Symbology & Identifiers",
				Domain:               "Capital Markets",
				Role:                 "Peer Symbology",
				Standard:             "ANSI X9.6",
				FormatPattern:        "^[0-9A-Z]{9}$",
				RelationshipType:     EdgeTypeIsPeerIdentifierOf,
				DifferentiationNotes: "9-character North American security identifier. Embeds directly into US/CA ISINs.",
				Confidence:           0.95,
			},
			{
				TermName:             "SEDOL",
				Category:             "Symbology & Identifiers",
				Domain:               "Capital Markets",
				Role:                 "Peer Symbology",
				Standard:             "LSEG SEDOL",
				FormatPattern:        "^[0-9B-DF-HJ-NP-TV-Z]{6}[0-9]$",
				RelationshipType:     EdgeTypeIsPeerIdentifierOf,
				DifferentiationNotes: "7-character UK/Irish security identifier issued by London Stock Exchange.",
				Confidence:           0.93,
			},
			{
				TermName:             "FIGI",
				Category:             "Symbology & Identifiers",
				Domain:               "Capital Markets",
				Role:                 "Peer Symbology",
				Standard:             "OMG FIGI",
				FormatPattern:        "^BBG[0-9A-Z]{9}$",
				RelationshipType:     EdgeTypeIsPeerIdentifierOf,
				DifferentiationNotes: "12-character open persistent identifier from Bloomberg Open Symbology starting with BBG.",
				Confidence:           0.90,
			},
			{
				TermName:             "LEI",
				Category:             "Entities & Counterparties",
				Domain:               "Regulatory Compliance",
				Role:                 "Entity Identifier (ISO 17442)",
				Standard:             "ISO 17442",
				FormatPattern:        "^[0-9A-Z]{18}[0-9]{2}$",
				RelationshipType:     EdgeTypeDifferentiatedFrom,
				DifferentiationNotes: "Identifies the corporate legal entity (issuer/broker), NOT the security/instrument itself.",
				Confidence:           0.88,
			},
		}

		return &TermDisambiguation{
			PrimaryTerm:           primary,
			RelatedTerms:          related,
			DifferentiatorSummary: "• ISIN (12-char global ISO 6166), CUSIP (9-char North America), and SEDOL (7-char UK/IE) identify instruments.\n• LEI (20-char ISO 17442) identifies legal companies/issuers, NOT securities.",
			DomainScope:           "Capital Markets",
		}
	}

	return &TermDisambiguation{
		PrimaryTerm: RelatedTermInfo{
			TermName:   termName,
			Category:   "General",
			Confidence: 0.70,
		},
		RelatedTerms: nil,
	}
}

// EvaluateSemanticBoundary analyzes the semantic relationship and differentiation between two terms.
// Returns the differentiation note and whether the terms are considered interchangeable synonyms.
func (s *TermRelationshipService) EvaluateSemanticBoundary(termA, termB string) (string, bool) {
	normA := strings.ToUpper(strings.TrimSpace(termA))
	normB := strings.ToUpper(strings.TrimSpace(termB))

	// Account hierarchy differentiation
	if strings.Contains(normA, "CUSTODIAL") && strings.Contains(normB, "ALLOCATION") {
		return "Custodial Account Code identifies external bank depository safekeeping accounts (e.g. State Street, BNY Mellon) where assets legally reside; Allocation Account Code is a trading sub-account for post-trade block execution fills and sleeve apportionment.", false
	}
	if strings.Contains(normA, "ALLOCATION") && strings.Contains(normB, "CUSTODIAL") {
		return "Allocation Account Code routes post-trade sleeve apportionment; Custodial Account Code identifies external depository safekeeping accounts.", false
	}

	// Symbology differentiation
	if strings.Contains(normA, "ISIN") && strings.Contains(normB, "CUSIP") {
		return "ISIN is a 12-character global standard (ISO 6166); CUSIP is a 9-character North American security identifier.", false
	}

	if normA == normB {
		return "Terms are identical synonyms.", true
	}

	return fmt.Sprintf("Terms '%s' and '%s' represent distinct semantic concepts in the domain taxonomy.", termA, termB), false
}

func (s *TermRelationshipService) buildFallbackAIContext(domain string) *AIContextPayload {
	if domain == "" {
		domain = "Enterprise & Financial Ontology"
	}

	terms := []AITermDefinition{
		{
			TermName:               "Account Code",
			QualifiedPath:          "business_term/Account Code",
			Domain:                 "Accounting & Finance",
			Category:               "Account & Entity",
			DataType:               "string",
			Definition:             "Generic account identifier representing a financial, institutional, or retail account entity.",
			DifferentiatorNotes:    "General umbrella identifier for any account hierarchy level.",
			SpecializedSubTerms:    []string{"Allocation Account Code", "Custodial Account Code", "GL Account Code", "Client Account Number"},
			DisambiguationGuidance: "Use as base parent term when specific sub-role is not designated.",
		},
		{
			TermName:               "Allocation Account Code",
			QualifiedPath:          "business_term/Allocation Account Code",
			Domain:                 "Trading & Portfolio Management",
			Category:               "Account & Entity",
			DataType:               "string",
			Definition:             "Sub-account identifier used for allocating executed block trades across individual portfolio sleeves or funds.",
			DifferentiatorNotes:    "Specific to post-trade execution and block allocation; distinct from custodial safekeeping accounts.",
			ParentTerm:             "Account Code",
			DisambiguationGuidance: "Routes post-trade sleeve executions; not where assets legally settle.",
		},
		{
			TermName:               "Custodial Account Code",
			QualifiedPath:          "business_term/Custodial Account Code",
			Domain:                 "Custody & Settlement",
			Category:               "Account & Entity",
			DataType:               "string",
			Definition:             "Account number assigned by a third-party depository or custodian bank holding client securities and cash.",
			DifferentiatorNotes:    "Identifies the external bank custodian safekeeping account (e.g. BNY Mellon, State Street, Euroclear).",
			ParentTerm:             "Account Code",
			DisambiguationGuidance: "Identifies external depository holding account.",
		},
		{
			TermName:               "ISIN",
			QualifiedPath:          "business_term/ISIN",
			Domain:                 "Capital Markets",
			Category:               "Symbology & Identifiers",
			DataType:               "string",
			Standard:               "ISO 6166",
			FormatPattern:          "^[A-Z]{2}[A-Z0-9]{9}[0-9]$",
			Definition:             "International Securities Identification Number (ISO 6166), 12-character alphanumeric global instrument identifier.",
			DifferentiatorNotes:    "12-character global standard with 2-char country prefix, 9-char NSIN (e.g. CUSIP/SEDOL base), and 1 check digit.",
			PeerIdentifiers:        []string{"CUSIP", "SEDOL", "FIGI"},
			DisambiguationGuidance: "Primary global security identifier.",
		},
		{
			TermName:               "CUSIP",
			QualifiedPath:          "business_term/CUSIP",
			Domain:                 "Capital Markets",
			Category:               "Symbology & Identifiers",
			DataType:               "string",
			Standard:               "ANSI X9.6",
			FormatPattern:          "^[0-9A-Z]{9}$",
			Definition:             "Committee on Uniform Securities Identification Procedures, 9-character North American security identifier.",
			DifferentiatorNotes:    "9-character identifier for US and Canadian securities. Embeds inside US/CA ISINs as the NSIN.",
			PeerIdentifiers:        []string{"ISIN", "SEDOL", "FIGI"},
			DisambiguationGuidance: "North American 9-character security identifier.",
		},
		{
			TermName:               "SEDOL",
			QualifiedPath:          "business_term/SEDOL",
			Domain:                 "Capital Markets",
			Category:               "Symbology & Identifiers",
			DataType:               "string",
			Standard:               "LSEG SEDOL",
			FormatPattern:          "^[0-9B-DF-HJ-NP-TV-Z]{6}[0-9]$",
			Definition:             "Stock Exchange Daily Official List, 7-character UK and European security identifier issued by LSEG.",
			DifferentiatorNotes:    "7-character UK/Ireland identifier (6 alphanumeric characters without vowels + 1 check digit).",
			PeerIdentifiers:        []string{"ISIN", "CUSIP", "FIGI"},
			DisambiguationGuidance: "UK & European 7-character security identifier.",
		},
	}

	edges := []AIGraphEdge{
		{
			SourceTerm:  "Allocation Account Code",
			Predicate:   "IS_SPECIALIZATION_OF",
			TargetTerm:  "Account Code",
			Explanation: "Allocation Account Code specifically routes post-trade sleeve fills.",
		},
		{
			SourceTerm:  "Custodial Account Code",
			Predicate:   "IS_SPECIALIZATION_OF",
			TargetTerm:  "Account Code",
			Explanation: "Custodial Account Code identifies external safekeeping custodian banks.",
		},
		{
			SourceTerm:  "ISIN",
			Predicate:   "IS_PEER_IDENTIFIER_OF",
			TargetTerm:  "CUSIP",
			Explanation: "US/CA CUSIPs embed directly into ISINs with US/CA prefix.",
		},
		{
			SourceTerm:  "ISIN",
			Predicate:   "IS_PEER_IDENTIFIER_OF",
			TargetTerm:  "SEDOL",
			Explanation: "UK/IE SEDOLs embed into GB/IE ISINs.",
		},
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString("### SEMANTIC ONTOLOGY & NODE DIFFERENTIATION GUIDE\n")
	promptBuilder.WriteString("Use these precise definitions and relationship rules to distinguish and map business concepts correctly:\n\n")

	for _, t := range terms {
		promptBuilder.WriteString(fmt.Sprintf("#### Term: `%s`\n", t.TermName))
		promptBuilder.WriteString(fmt.Sprintf("- **Definition**: %s\n", t.Definition))
		promptBuilder.WriteString(fmt.Sprintf("- **Domain/Category**: %s / %s (Data Type: %s)\n", t.Domain, t.Category, t.DataType))
		if t.Standard != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Standard / Format**: %s (`%s`)\n", t.Standard, t.FormatPattern))
		}
		if t.ParentTerm != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Parent Concept**: `%s`\n", t.ParentTerm))
		}
		if len(t.SpecializedSubTerms) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Specialized Variants**: %s\n", strings.Join(t.SpecializedSubTerms, ", ")))
		}
		if len(t.PeerIdentifiers) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Peer Symbologies**: %s\n", strings.Join(t.PeerIdentifiers, ", ")))
		}
		if t.DifferentiatorNotes != "" {
			promptBuilder.WriteString(fmt.Sprintf("- **Differentiator & Usage**: %s\n", t.DifferentiatorNotes))
		}
		promptBuilder.WriteString("\n")
	}

	directives := []string{
		"Do NOT substitute custodial_account_code when allocation_account_code is requested in trade allocation contexts.",
		"When security symbology is ambiguous, inspect IS_PEER_IDENTIFIER_OF edges to resolve CUSIP/SEDOL cross-references.",
		"Enforce strict distinction between trade_date (market agreement) and settlement_date (cash delivery).",
		"LEI identifies corporate legal entity issuers/counterparties, NOT specific traded instruments (ISIN/CUSIP/SEDOL).",
	}

	return &AIContextPayload{
		Version:            "1.0.0",
		GeneratedAt:        time.Now().UTC(),
		TenantID:           "default",
		Domain:             domain,
		TermCount:          len(terms),
		Terms:              terms,
		TaxonomyEdges:      edges,
		ActiveDirectives:   directives,
		PromptContextBlock: promptBuilder.String(),
		JSONLDSchema: map[string]interface{}{
			"@context": map[string]interface{}{
				"schema": "http://schema.org/",
				"fibo":   "https://spec.edmcouncil.org/fibo/ontology/",
			},
			"@graph": terms,
		},
	}
}

func getRelStringProp(m map[string]interface{}, key, defaultVal string) string {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

// L3ClassificationInfo represents a Tier-3 taxonomy node with its full breadcrumb path.
type L3ClassificationInfo struct {
	ID            string `json:"id" db:"id"`
	Name          string `json:"name" db:"name"`
	QualifiedPath string `json:"qualified_path" db:"qualified_path"`
	CategoryName  string `json:"category_name" db:"category_name"`
	DomainName    string `json:"domain_name" db:"domain_name"`
	Breadcrumb    string `json:"breadcrumb" db:"breadcrumb"`
	Description   string `json:"description" db:"description"`
}

// ListL3Classifications returns all Tier-3 classifications with resolved L1 > L2 > L3 breadcrumbs.
func (s *TermRelationshipService) ListL3Classifications(ctx context.Context, tenantID string) ([]L3ClassificationInfo, error) {
	if s.db == nil {
		return s.getFallbackL3Classifications(), nil
	}

	goldTenantID, _ := s.GetGoldTenantID(ctx)

	query := `
		SELECT 
			l3.id,
			l3.node_name as name,
			l3.qualified_path,
			COALESCE(l2.node_name, 'General') as category_name,
			COALESCE(l1.node_name, 'Enterprise Core') as domain_name,
			COALESCE(
				l3.properties->>'breadcrumb',
				COALESCE(l1.node_name, 'Enterprise Core') || ' > ' || COALESCE(l2.node_name, 'General') || ' > ' || l3.node_name
			) as breadcrumb,
			COALESCE(l3.description, '') as description
		FROM catalog_node l3
		LEFT JOIN catalog_node l2 ON l3.parent_id = l2.id
		LEFT JOIN catalog_node l1 ON l2.parent_id = l1.id
		WHERE l3.node_type_id = '55555555-5555-5555-5555-555555550003'
		  AND (l3.tenant_id = $1 OR l3.tenant_id = $2 OR l3.tenant_id = '00000000-0000-0000-0000-000000000000')
		ORDER BY l1.node_name, l2.node_name, l3.node_name ASC
	`

	var results []L3ClassificationInfo
	err := s.db.SelectContext(ctx, &results, query, tenantID, goldTenantID)
	if err != nil || len(results) == 0 {
		return s.getFallbackL3Classifications(), nil
	}
	return results, nil
}

// SuggestL3Classification maps a term name or column name to one of the 16 canonical L3 classifications.
func (s *TermRelationshipService) SuggestL3Classification(termName, columnName string) *L3ClassificationInfo {
	target := strings.ToUpper(strings.TrimSpace(termName))
	if target == "" {
		target = strings.ToUpper(strings.TrimSpace(columnName))
	}

	all := s.getFallbackL3Classifications()
	matchName := ""

	// 1. Account & Custody
	if strings.Contains(target, "CUSTOD") {
		matchName = "Custodial Safekeeping"
	} else if strings.Contains(target, "STATUS") || strings.Contains(target, "TYPE_CODE") || strings.Contains(target, "MANAGER") || strings.Contains(target, "ACTION_TYPE") {
		matchName = "Account Governance"
	} else if strings.Contains(target, "ACCOUNT") || strings.Contains(target, "ACCT") {
		if strings.Contains(target, "ALLOC") {
			matchName = "Trade Allocation"
		} else {
			matchName = "Account Identification"
		}
	}

	// 2. Orders & Execution
	if matchName == "" {
		if strings.Contains(target, "ALLOC") {
			matchName = "Trade Allocation"
		} else if strings.Contains(target, "EXEC") {
			matchName = "Trade Execution"
		} else if strings.Contains(target, "ORDER") || strings.Contains(target, "DIRECTION") || strings.Contains(target, "SIDE") {
			matchName = "Order Lifecycle"
		}
	}

	// 3. FIX Protocol
	if matchName == "" {
		if strings.Contains(target, "FIX") {
			if strings.Contains(target, "MSG") || strings.Contains(target, "MESSAGE") || strings.Contains(target, "DEFAULT") {
				matchName = "FIX Message Payload"
			} else {
				matchName = "FIX Session & Connectivity"
			}
		}
	}

	// 4. Broker / Counterparty
	if matchName == "" {
		if strings.Contains(target, "BROKER") || strings.Contains(target, "BKR") || strings.Contains(target, "ISSUER") {
			matchName = "Broker & Counterparty"
		}
	}

	// 5. Positions & Amounts
	if matchName == "" {
		if strings.Contains(target, "POSITION") || strings.Contains(target, "HOLDING") || strings.Contains(target, "SHARE") || strings.Contains(target, "BASE_COST") {
			matchName = "Position & Lot Balances"
		} else if strings.Contains(target, "AMOUNT") || strings.Contains(target, "CURRENCY") || strings.Contains(target, "CRRN") || strings.Contains(target, "REVENUE") || strings.Contains(target, "PRICE") {
			if strings.Contains(target, "LAST") || strings.Contains(target, "CURVE") || strings.Contains(target, "EXCH") {
				matchName = "Market Quotes & Pricing"
			} else {
				matchName = "Monetary Amounts"
			}
		}
	}

	// 6. Corporate Actions
	if matchName == "" {
		if strings.Contains(target, "CORP") || strings.Contains(target, "ACTION") || strings.Contains(target, "DIVIDEND") || strings.Contains(target, "EX_DATE") {
			matchName = "Corporate Action Event"
		}
	}

	// 7. Symbology & Market Ref
	if matchName == "" {
		if strings.Contains(target, "ISIN") || strings.Contains(target, "CUSIP") || strings.Contains(target, "SEDOL") || strings.Contains(target, "FIGI") || strings.Contains(target, "LEI") || strings.Contains(target, "SECURITY") {
			matchName = "Instrument Symbology"
		} else if strings.Contains(target, "CALENDAR") || strings.Contains(target, "HOLIDAY") {
			matchName = "Exchange Calendar"
		} else if strings.Contains(target, "ALERT") || strings.Contains(target, "RULE") {
			matchName = "Monitoring & Rules"
		} else if strings.Contains(target, "DATE") || strings.Contains(target, "TIME") || strings.Contains(target, "AUDIT") || strings.Contains(target, "CODE") || strings.Contains(target, "CATEGORY") {
			matchName = "Audit & Temporal Metadata"
		}
	}

	if matchName == "" {
		matchName = "Account Identification"
	}

	for _, item := range all {
		if item.Name == matchName {
			res := item
			return &res
		}
	}
	return &all[0]
}

// ClassifyTerm links a business term or semantic term to an L3 classification node via CLASSIFIED_BY.
func (s *TermRelationshipService) ClassifyTerm(ctx context.Context, tenantID, termID, l3NodeID string) error {
	if s.db == nil {
		return nil
	}

	query := `
		INSERT INTO catalog_edge (
			tenant_id, source_id, target_id, edge_type_name, edge_type_id, properties, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'CLASSIFIED_BY', '66666666-6666-6666-6666-666666660001', '{"tier": "L3"}'::jsonb, NOW(), NOW()
		)
		ON CONFLICT (tenant_id, source_id, target_id, edge_type_name)
		DO UPDATE SET
			updated_at = NOW()
	`
	_, err := s.db.ExecContext(ctx, query, tenantID, termID, l3NodeID)
	return err
}

func (s *TermRelationshipService) getFallbackL3Classifications() []L3ClassificationInfo {
	return []L3ClassificationInfo{
		{
			ID:            "a3000000-0000-0000-0000-000000000001",
			Name:          "Order Lifecycle",
			QualifiedPath: "classification/order_lifecycle",
			CategoryName:  "Orders & Allocations",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Orders & Allocations > Order Lifecycle",
			Description:   "Top-level order attributes, side, state, and tracking.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000002",
			Name:          "Trade Allocation",
			QualifiedPath: "classification/trade_allocation",
			CategoryName:  "Orders & Allocations",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Orders & Allocations > Trade Allocation",
			Description:   "Block splitting, child shares, allocation accounts.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000003",
			Name:          "Trade Execution",
			QualifiedPath: "classification/trade_execution",
			CategoryName:  "Orders & Allocations",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Orders & Allocations > Trade Execution",
			Description:   "Fills, execution timestamps, venues, and prices.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000004",
			Name:          "FIX Session & Connectivity",
			QualifiedPath: "classification/fix_session_connectivity",
			CategoryName:  "Financial Protocols",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Financial Protocols > FIX Session & Connectivity",
			Description:   "FIX session states, sender/target comp IDs.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000005",
			Name:          "FIX Message Payload",
			QualifiedPath: "classification/fix_message_payload",
			CategoryName:  "Financial Protocols",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Financial Protocols > FIX Message Payload",
			Description:   "Protocol message types, raw tags, and defaults.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000006",
			Name:          "Broker & Counterparty",
			QualifiedPath: "classification/broker_counterparty",
			CategoryName:  "Intermediaries",
			DomainName:    "Trading & Execution (OMS/EMS)",
			Breadcrumb:    "Trading & Execution (OMS/EMS) > Intermediaries > Broker & Counterparty",
			Description:   "Executing brokers, clearing firms, broker status.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000007",
			Name:          "Account Identification",
			QualifiedPath: "classification/account_identification",
			CategoryName:  "Account Master",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Account Master > Account Identification",
			Description:   "Internal account codes, account numbers, names.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000008",
			Name:          "Custodial Safekeeping",
			QualifiedPath: "classification/custodial_safekeeping",
			CategoryName:  "Account Master",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Account Master > Custodial Safekeeping",
			Description:   "Depository accounts, custodian bank identifiers.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000009",
			Name:          "Account Governance",
			QualifiedPath: "classification/account_governance",
			CategoryName:  "Account Master",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Account Master > Account Governance",
			Description:   "Account types, tax treatments, lifecycle states.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000010",
			Name:          "Position & Lot Balances",
			QualifiedPath: "classification/position_lot_balances",
			CategoryName:  "Positions & Valuation",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Positions & Valuation > Position & Lot Balances",
			Description:   "Shares held, base cost, position instances.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000011",
			Name:          "Monetary Amounts",
			QualifiedPath: "classification/monetary_amounts",
			CategoryName:  "Positions & Valuation",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Positions & Valuation > Monetary Amounts",
			Description:   "Financial consideration, fees, currencies.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000012",
			Name:          "Corporate Action Event",
			QualifiedPath: "classification/corporate_action_event",
			CategoryName:  "Asset Servicing",
			DomainName:    "Portfolio, Accounting & Custody",
			Breadcrumb:    "Portfolio, Accounting & Custody > Asset Servicing > Corporate Action Event",
			Description:   "Ex/record/pay dates, corporate action types.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000013",
			Name:          "Instrument Symbology",
			QualifiedPath: "classification/instrument_symbology",
			CategoryName:  "Security Master",
			DomainName:    "Securities & Market Data",
			Breadcrumb:    "Securities & Market Data > Security Master > Instrument Symbology",
			Description:   "CUSIP, ISIN, SEDOL, FIGI, LEI, Tickers.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000014",
			Name:          "Market & Currency Reference",
			QualifiedPath: "classification/market_currency_reference",
			CategoryName:  "Security Master",
			DomainName:    "Securities & Market Data",
			Breadcrumb:    "Securities & Market Data > Security Master > Market & Currency Reference",
			Description:   "ISO currency codes, curve categories.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000015",
			Name:          "Market Quotes & Pricing",
			QualifiedPath: "classification/market_quotes_pricing",
			CategoryName:  "Pricing & Analytics",
			DomainName:    "Securities & Market Data",
			Breadcrumb:    "Securities & Market Data > Pricing & Analytics > Market Quotes & Pricing",
			Description:   "Last price, bid/ask, closing valuations.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000016",
			Name:          "Audit & Temporal Metadata",
			QualifiedPath: "classification/audit_temporal_metadata",
			CategoryName:  "Governance & Ops",
			DomainName:    "Platform Ops & Data Lake",
			Breadcrumb:    "Platform Ops & Data Lake > Governance & Ops > Audit & Temporal Metadata",
			Description:   "Effective dates, created timestamps, as-of dates.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000017",
			Name:          "Exchange Calendar",
			QualifiedPath: "classification/exchange_calendar",
			CategoryName:  "Governance & Ops",
			DomainName:    "Platform Ops & Data Lake",
			Breadcrumb:    "Platform Ops & Data Lake > Governance & Ops > Exchange Calendar",
			Description:   "Market holidays, exchange settlement schedules.",
		},
		{
			ID:            "a3000000-0000-0000-0000-000000000018",
			Name:          "Monitoring & Rules",
			QualifiedPath: "classification/monitoring_rules",
			CategoryName:  "Governance & Ops",
			DomainName:    "Platform Ops & Data Lake",
			Breadcrumb:    "Platform Ops & Data Lake > Governance & Ops > Monitoring & Rules",
			Description:   "Business rules, alert messages, threshold breaks.",
		},
	}
}

// LensType defines projection lens for the cognitive graph
type LensType string

const (
	LensSemanticCalculationMesh LensType = "SEMANTIC_CALCULATION_MESH"
	LensPhysicalERD             LensType = "PHYSICAL_ERD"
	LensSubtypeAndPeers         LensType = "SUBTYPE_AND_PEERS"
	LensTaxonomyHierarchy       LensType = "TAXONOMY_HIERARCHY"
	LensPipelineImpact          LensType = "PIPELINE_IMPACT"
)

type VisualizeLensRequest struct {
	TenantID        string   `json:"tenant_id"`
	NodeName        string   `json:"node_name,omitempty"`
	LensType        LensType `json:"lens_type"`
	Depth           int      `json:"depth"`
	IncludeIndirect bool     `json:"include_indirect"`
}

type LensGraphNode struct {
	ID         string                 `json:"id"`
	NodeKey    string                 `json:"node_key"`
	NodeName   string                 `json:"node_name"`
	NodeType   string                 `json:"node_type"`
	IsFocal    bool                   `json:"is_focal"`
	SourceTier string                 `json:"source_tier"` // CORE or CUSTOM
	Properties map[string]interface{} `json:"properties"`
}

type LensGraphEdge struct {
	ID         string                 `json:"id"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	EdgeType   string                 `json:"edge_type"`
	Properties map[string]interface{} `json:"properties"`
}

type LensContainers struct {
	ConnectedBusinessObjectsCount int `json:"connected_business_objects_count"`
	PhysicalTableBindingsCount    int `json:"physical_table_bindings_count"`
	SiblingSubtypesCount          int `json:"sibling_subtypes_count"`
	PeerSymbologiesCount          int `json:"peer_symbologies_count"`
}

type VisualizeLensResponse struct {
	FocalNodeID    string          `json:"focal_node_id"`
	ActiveLens     LensType        `json:"active_lens"`
	BreadcrumbPath string          `json:"breadcrumb_path"`
	Nodes          []LensGraphNode `json:"nodes"`
	Edges          []LensGraphEdge `json:"edges"`
	Containers     LensContainers  `json:"containers"`
}

// VisualizeLens builds projection graph data for the 5 Cognitive Studio Lenses.
func (s *TermRelationshipService) VisualizeLens(ctx context.Context, tenantID, nodeIDOrName string, req VisualizeLensRequest) (*VisualizeLensResponse, error) {
	if tenantID == "" {
		tenantID, _ = s.GetGoldTenantID(ctx)
		if tenantID == "" {
			tenantID = "00000000-0000-0000-0000-000000000000"
		}
	}
	if req.LensType == "" {
		req.LensType = LensSubtypeAndPeers
	}

	// 1. Resolve focal node
	focalName := strings.TrimSpace(nodeIDOrName)
	if req.NodeName != "" {
		focalName = strings.TrimSpace(req.NodeName)
	}
	focalID := strings.TrimSpace(nodeIDOrName)
	focalType := "business_term"
	var focalProps map[string]interface{}
	focalQualifiedPath := ""
	focalDesc := ""

	if s.db != nil {
		var row struct {
			ID            string         `db:"id"`
			NodeName      sql.NullString `db:"node_name"`
			Name          sql.NullString `db:"name"`
			NodeType      string         `db:"node_type"`
			QualifiedPath sql.NullString `db:"qualified_path"`
			Description   sql.NullString `db:"description"`
			PropsJSON     []byte         `db:"properties"`
		}
		query := `
			SELECT cn.id, cn.node_name, cn.name, COALESCE(cnt.type_name, 'business_term') as node_type,
			       cn.qualified_path, cn.description, cn.properties
			FROM catalog_node cn
			LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
			WHERE (cn.id::text = $1 OR UPPER(COALESCE(NULLIF(cn.node_name, ''), NULLIF(cn.name, ''), '')) = UPPER($1))
			ORDER BY (cn.tenant_id::text = $2) DESC, cn.created_at DESC
			LIMIT 1
		`
		err := s.db.GetContext(ctx, &row, query, nodeIDOrName, tenantID)
		if err != nil {
			// Fallback search without tenant restriction
			fallbackQuery := `
				SELECT cn.id, cn.node_name, cn.name, COALESCE(cnt.type_name, 'business_term') as node_type,
				       cn.qualified_path, cn.description, cn.properties
				FROM catalog_node cn
				LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
				WHERE (cn.id::text = $1 OR UPPER(COALESCE(NULLIF(cn.node_name, ''), NULLIF(cn.name, ''), '')) = UPPER($1))
				ORDER BY cn.created_at DESC
				LIMIT 1
			`
			err = s.db.GetContext(ctx, &row, fallbackQuery, nodeIDOrName)
		}
		if err == nil {
			focalID = row.ID
			nameVal := ""
			if row.NodeName.Valid && strings.TrimSpace(row.NodeName.String) != "" {
				nameVal = strings.TrimSpace(row.NodeName.String)
			} else if row.Name.Valid && strings.TrimSpace(row.Name.String) != "" {
				nameVal = strings.TrimSpace(row.Name.String)
			}
			if nameVal != "" {
				focalName = nameVal
			}
			focalType = row.NodeType
			focalQualifiedPath = row.QualifiedPath.String
			focalDesc = row.Description.String
			if len(row.PropsJSON) > 0 {
				_ = json.Unmarshal(row.PropsJSON, &focalProps)
			}
		}
	}

	if focalProps == nil {
		focalProps = make(map[string]interface{})
	}
	if focalQualifiedPath != "" && focalProps["qualified_path"] == nil {
		focalProps["qualified_path"] = focalQualifiedPath
	}

	// Resolve breadcrumb
	breadcrumb := "Enterprise Core > General > " + focalName
	sugL3 := s.SuggestL3Classification(focalName, "")
	if sugL3 != nil && sugL3.Breadcrumb != "" {
		breadcrumb = sugL3.Breadcrumb
	}

	nodes := make([]LensGraphNode, 0)
	edges := make([]LensGraphEdge, 0)
	containers := LensContainers{}

	focalKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(focalName, " ", "_"), "-", "_"))

	focalNode := LensGraphNode{
		ID:         focalID,
		NodeKey:    focalKey,
		NodeName:   focalName,
		NodeType:   focalType,
		IsFocal:    true,
		SourceTier: "CORE",
		Properties: focalProps,
	}

	domainStr := ""
	if d, ok := focalProps["domain"].(string); ok && d != "" {
		domainStr = d
	} else if sugL3 != nil && sugL3.DomainName != "" {
		domainStr = sugL3.DomainName
	} else {
		domainStr = "Enterprise Core"
	}

	categoryStr := ""
	if c, ok := focalProps["category"].(string); ok && c != "" {
		categoryStr = c
	} else if sugL3 != nil && sugL3.CategoryName != "" {
		categoryStr = sugL3.CategoryName
	} else {
		categoryStr = "General"
	}

	switch req.LensType {
	case LensSubtypeAndPeers:
		// Subtype, Symbology & Peer Lens
		// 1. Try querying real relationship edges from DB
		type DBEdgeRow struct {
			EdgeID       string         `db:"id"`
			SourceID     string         `db:"source_id"`
			TargetID     string         `db:"target_id"`
			EdgeTypeName string         `db:"edge_type_name"`
			Properties   []byte         `db:"properties"`
			OtherNodeID  string         `db:"other_node_id"`
			OtherName    string         `db:"other_name"`
			OtherType    string         `db:"other_type"`
			OtherPath    string         `db:"other_path"`
			OtherDesc    sql.NullString `db:"other_desc"`
			OtherProps   []byte         `db:"other_props"`
			IsOutgoing   bool           `db:"is_outgoing"`
		}

		var dbEdges []DBEdgeRow
		if s.db != nil {
			edgeQuery := `
				SELECT ce.id, ce.source_id, ce.target_id, ce.edge_type_name, ce.properties,
				       cn.id as other_node_id, 
				       COALESCE(NULLIF(cn.node_name, ''), NULLIF(cn.name, ''), '') as other_name, 
				       COALESCE(cnt.type_name, 'business_term') as other_type,
				       cn.qualified_path as other_path, cn.description as other_desc, cn.properties as other_props,
				       (ce.source_id::text = $1) as is_outgoing
				FROM catalog_edge ce
				JOIN catalog_node cn ON (CASE WHEN ce.source_id::text = $1 THEN ce.target_id = cn.id ELSE ce.source_id = cn.id END)
				LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
				WHERE (ce.source_id::text = $1 OR ce.target_id::text = $1)
				  AND (ce.edge_type_name IN ('IS_SPECIALIZATION_OF', 'IS_GENERALIZATION_OF', 'IS_PEER_IDENTIFIER_OF', 'DIFFERENTIATED_FROM', 'RELATES_TO'))
				  AND (ce.tenant_id::text = $2 OR ce.tenant_id::text = '00000000-0000-0000-0000-000000000000' OR $2 = '')
				  AND ce.is_active = true
			`
			_ = s.db.SelectContext(ctx, &dbEdges, edgeQuery, focalID, tenantID)
		}

		if len(dbEdges) > 0 {
			nodes = append(nodes, focalNode)
			for _, row := range dbEdges {
				var p map[string]interface{}
				if len(row.OtherProps) > 0 {
					_ = json.Unmarshal(row.OtherProps, &p)
				}
				if p == nil {
					p = make(map[string]interface{})
				}
				if row.OtherDesc.Valid && row.OtherDesc.String != "" {
					p["description"] = row.OtherDesc.String
				}
				role := "Related Term"
				switch row.EdgeTypeName {
				case EdgeTypeIsSpecializationOf:
					if row.IsOutgoing {
						role = "Supertype Umbrella Parent"
					} else {
						role = "Specialized Subtype"
					}
				case EdgeTypeIsGeneralizationOf:
					if row.IsOutgoing {
						role = "Specialized Subtype"
					} else {
						role = "Supertype Umbrella Parent"
					}
				case EdgeTypeIsPeerIdentifierOf:
					role = "Peer Symbology / Alternate ID"
				case EdgeTypeDifferentiatedFrom:
					role = "Differentiated Alternative"
				}
				p["role"] = role

				n := LensGraphNode{
					ID:         row.OtherNodeID,
					NodeKey:    strings.ToLower(strings.ReplaceAll(row.OtherName, " ", "_")),
					NodeName:   row.OtherName,
					NodeType:   row.OtherType,
					SourceTier: "CORE",
					Properties: p,
				}
				nodes = append(nodes, n)

				var edgeProps map[string]interface{}
				if len(row.Properties) > 0 {
					_ = json.Unmarshal(row.Properties, &edgeProps)
				}

				edges = append(edges, LensGraphEdge{
					ID:         row.EdgeID,
					SourceID:   row.SourceID,
					TargetID:   row.TargetID,
					EdgeType:   row.EdgeTypeName,
					Properties: edgeProps,
				})
			}
			containers.SiblingSubtypesCount = len(nodes) - 1
		} else {
			// Dynamic fallback disambiguation based on focal term
			disambig := s.buildFallbackDisambiguation(focalName)
			if disambig != nil && len(disambig.RelatedTerms) > 0 {
				isParent := true
				normFocal := strings.ToUpper(focalName)
				for _, r := range disambig.RelatedTerms {
					if strings.ToUpper(r.TermName) == normFocal {
						isParent = false
						break
					}
				}

				focalNode.Properties["role"] = disambig.PrimaryTerm.Role
				if focalNode.Properties["role"] == nil || focalNode.Properties["role"] == "" {
					if isParent {
						focalNode.Properties["role"] = "Supertype Concept"
					} else {
						focalNode.Properties["role"] = "Specialized Subtype"
					}
				}
				if focalDesc != "" {
					focalNode.Properties["description"] = focalDesc
				}
				nodes = append(nodes, focalNode)

				for _, rel := range disambig.RelatedTerms {
					if strings.ToUpper(rel.TermName) == normFocal {
						continue
					}
					relID := rel.TermID
					if relID == "" {
						relID = uuid.New().String()
					}
					props := rel.Properties
					if props == nil {
						props = make(map[string]interface{})
					}
					if rel.Role != "" {
						props["role"] = rel.Role
					}
					if rel.DifferentiationNotes != "" {
						props["differentiation"] = rel.DifferentiationNotes
					}
					if rel.Standard != "" {
						props["standard"] = rel.Standard
					}
					if rel.FormatPattern != "" {
						props["format_pattern"] = rel.FormatPattern
					}

					rn := LensGraphNode{
						ID:         relID,
						NodeKey:    strings.ToLower(strings.ReplaceAll(rel.TermName, " ", "_")),
						NodeName:   rel.TermName,
						NodeType:   "business_term",
						SourceTier: "CORE",
						Properties: props,
					}
					nodes = append(nodes, rn)

					eType := rel.RelationshipType
					if eType == "" {
						if strings.Contains(rel.Role, "Peer") {
							eType = EdgeTypeIsPeerIdentifierOf
						} else if strings.Contains(rel.Role, "Specialization") || strings.Contains(rel.Role, "Sub-type") {
							eType = EdgeTypeIsSpecializationOf
						} else {
							eType = EdgeTypeRelatesTo
						}
					}

					if eType == EdgeTypeIsSpecializationOf {
						// Child points to Parent
						if isParent {
							edges = append(edges, LensGraphEdge{
								ID:       uuid.New().String(),
								SourceID: rn.ID,
								TargetID: focalNode.ID,
								EdgeType: EdgeTypeIsSpecializationOf,
							})
						} else {
							edges = append(edges, LensGraphEdge{
								ID:       uuid.New().String(),
								SourceID: focalNode.ID,
								TargetID: rn.ID,
								EdgeType: EdgeTypeIsSpecializationOf,
							})
						}
					} else {
						edges = append(edges, LensGraphEdge{
							ID:       uuid.New().String(),
							SourceID: focalNode.ID,
							TargetID: rn.ID,
							EdgeType: eType,
						})
					}
				}
				containers.SiblingSubtypesCount = len(nodes) - 1
			} else if focalType == "business_object" {
				// Business Object: Generate entity subtypes
				focalNode.Properties["role"] = "Enterprise Business Object"
				if focalDesc != "" {
					focalNode.Properties["description"] = focalDesc
				}
				nodes = append(nodes, focalNode)

				st1 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "standard_" + focalKey,
					NodeName:   focalName + " (Standard)",
					NodeType:   "business_object",
					SourceTier: "CORE",
					Properties: map[string]interface{}{
						"role":            "Specialized Subtype",
						"differentiation": "Base operational execution contract for " + focalName + ".",
					},
				}
				st2 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "allocated_" + focalKey,
					NodeName:   focalName + " (Allocated)",
					NodeType:   "business_object",
					SourceTier: "CORE",
					Properties: map[string]interface{}{
						"role":            "Specialized Subtype",
						"differentiation": "Multi-sleeve or fund apportionment subtype for " + focalName + ".",
					},
				}
				nodes = append(nodes, st1, st2)
				edges = append(edges,
					LensGraphEdge{ID: uuid.New().String(), SourceID: st1.ID, TargetID: focalNode.ID, EdgeType: "IS_SPECIALIZATION_OF"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: st2.ID, TargetID: focalNode.ID, EdgeType: "IS_SPECIALIZATION_OF"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: st1.ID, TargetID: st2.ID, EdgeType: "DIFFERENTIATED_FROM", Properties: map[string]interface{}{"reason": "Standard single-entity vs Multi-sleeve allocation"}},
				)
				containers.SiblingSubtypesCount = 2
			} else {
				// Term has no static family - generate contextual supertype & specialized subtypes
				focalNode.Properties["role"] = "Parent Concept"
				if focalDesc != "" {
					focalNode.Properties["description"] = focalDesc
				}
				nodes = append(nodes, focalNode)

				c1 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "direct_" + focalKey,
					NodeName:   "Direct " + focalName,
					NodeType:   focalType,
					SourceTier: "CORE",
					Properties: map[string]interface{}{
						"role":            "Specialized Subtype",
						"differentiation": "Primary direct source partition for " + focalName + " without adjustments.",
					},
				}
				c2 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "allocated_" + focalKey,
					NodeName:   "Allocated " + focalName,
					NodeType:   focalType,
					SourceTier: "CORE",
					Properties: map[string]interface{}{
						"role":            "Specialized Subtype",
						"differentiation": "Apportioned or downstream allocated representation of " + focalName + ".",
					},
				}
				c3 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "external_" + focalKey,
					NodeName:   "External " + focalName,
					NodeType:   focalType,
					SourceTier: "CORE",
					Properties: map[string]interface{}{
						"role":            "Specialized Subtype",
						"differentiation": "Counterparty or third-party depository reported " + focalName + ".",
					},
				}
				nodes = append(nodes, c1, c2, c3)

				edges = append(edges,
					LensGraphEdge{ID: uuid.New().String(), SourceID: c1.ID, TargetID: focalNode.ID, EdgeType: "IS_SPECIALIZATION_OF"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: c2.ID, TargetID: focalNode.ID, EdgeType: "IS_SPECIALIZATION_OF"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: c3.ID, TargetID: focalNode.ID, EdgeType: "IS_SPECIALIZATION_OF"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: c1.ID, TargetID: c2.ID, EdgeType: "DIFFERENTIATED_FROM", Properties: map[string]interface{}{"reason": "Direct execution vs Allocated split"}},
				)
				containers.SiblingSubtypesCount = 3
			}
		}

	case LensTaxonomyHierarchy:
		// 3-Tier Enterprise Taxonomy Lens: Domain (L1) -> Category (L2) -> Classification (L3) <- CLASSIFIED_BY <- Focal Node
		l1Name := domainStr
		l2Name := categoryStr
		l3Name := focalName

		if sugL3 != nil {
			if sugL3.DomainName != "" {
				l1Name = sugL3.DomainName
			}
			if sugL3.CategoryName != "" {
				l2Name = sugL3.CategoryName
			}
			if sugL3.Name != "" {
				l3Name = sugL3.Name
			}
		}

		l1ID := uuid.New().String()
		l2ID := uuid.New().String()
		l3ID := uuid.New().String()
		if sugL3 != nil && sugL3.ID != "" {
			l3ID = sugL3.ID
		}

		l1Node := LensGraphNode{
			ID:         l1ID,
			NodeKey:    "domain_" + strings.ToLower(strings.ReplaceAll(l1Name, " ", "_")),
			NodeName:   l1Name,
			NodeType:   "Classification_L1",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"tier": "Domain (L1)"},
		}
		l2Node := LensGraphNode{
			ID:         l2ID,
			NodeKey:    "cat_" + strings.ToLower(strings.ReplaceAll(l2Name, " ", "_")),
			NodeName:   l2Name,
			NodeType:   "Classification_L2",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"tier": "Category (L2)"},
		}
		l3Node := LensGraphNode{
			ID:         l3ID,
			NodeKey:    "class_" + strings.ToLower(strings.ReplaceAll(l3Name, " ", "_")),
			NodeName:   l3Name,
			NodeType:   "Classification_L3",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"tier": "Classification (L3)"},
		}

		nodes = append(nodes, l1Node, l2Node, l3Node, focalNode)
		edges = append(edges,
			LensGraphEdge{ID: uuid.New().String(), SourceID: l1Node.ID, TargetID: l2Node.ID, EdgeType: "PARENT_OF"},
			LensGraphEdge{ID: uuid.New().String(), SourceID: l2Node.ID, TargetID: l3Node.ID, EdgeType: "PARENT_OF"},
			LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: l3Node.ID, EdgeType: "CLASSIFIED_BY"},
		)

	case LensSemanticCalculationMesh:
		// Semantic Calculation Mesh for Business Objects, Tables, Measures & Dimensions
		if focalType == "business_object" {
			focalNode.Properties["role"] = "ENTERPRISE BUSINESS OBJECT"
			nodes = append(nodes, focalNode)

			f1 := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "field_id_" + focalKey,
				NodeName:   focalName + " ID",
				NodeType:   "semantic_term",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"data_type": "UUID", "role": "Primary Key Field"},
			}
			f2 := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "field_status_" + focalKey,
				NodeName:   "Status Code",
				NodeType:   "semantic_term",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"data_type": "VARCHAR(32)", "role": "Lifecycle Dimension"},
			}
			f3 := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "field_amount_" + focalKey,
				NodeName:   "Monetary Value",
				NodeType:   "semantic_term",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"data_type": "NUMERIC(18,4)", "role": "Measure Field", "ast_expression": "SUM(unit_price * quantity)"},
			}
			nodes = append(nodes, f1, f2, f3)

			consumer := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "endpoint_" + focalKey,
				NodeName:   focalName + " REST API & GraphQL Data Product",
				NodeType:   "consumer_endpoint",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"protocol": "HTTPS / JSON:API", "status": "ACTIVE"},
			}
			nodes = append(nodes, consumer)

			edges = append(edges,
				LensGraphEdge{ID: uuid.New().String(), SourceID: f1.ID, TargetID: focalNode.ID, EdgeType: "HAS_FIELD"},
				LensGraphEdge{ID: uuid.New().String(), SourceID: f2.ID, TargetID: focalNode.ID, EdgeType: "HAS_FIELD"},
				LensGraphEdge{ID: uuid.New().String(), SourceID: f3.ID, TargetID: focalNode.ID, EdgeType: "HAS_MEASURE"},
				LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: consumer.ID, EdgeType: "SERVES_CONSUMER"},
			)
			containers.ConnectedBusinessObjectsCount = 1
		} else if focalType == "database_table" {
			focalNode.Properties["role"] = "STORAGE TABLE"
			nodes = append(nodes, focalNode)

			c1 := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "col_pk_" + focalKey,
				NodeName:   "🔑 " + focalKey + "_id",
				NodeType:   "table_column",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"data_type": "UUID", "role": "Primary Key"},
			}
			c2 := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "col_data_" + focalKey,
				NodeName:   "🔹 " + focalKey + "_name",
				NodeType:   "table_column",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"data_type": "VARCHAR(128)", "role": "Attribute Column"},
			}
			nodes = append(nodes, c1, c2)

			st := LensGraphNode{
				ID:         uuid.New().String(),
				NodeKey:    "sem_term_" + focalKey,
				NodeName:   focalName + " Semantic Entity",
				NodeType:   "semantic_term",
				SourceTier: "CORE",
				Properties: map[string]interface{}{"role": "Semantic Model Entity"},
			}
			nodes = append(nodes, st)

			edges = append(edges,
				LensGraphEdge{ID: uuid.New().String(), SourceID: c1.ID, TargetID: focalNode.ID, EdgeType: "CONTAINS_COLUMN"},
				LensGraphEdge{ID: uuid.New().String(), SourceID: c2.ID, TargetID: focalNode.ID, EdgeType: "CONTAINS_COLUMN"},
				LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: st.ID, EdgeType: "MAPS_TO"},
			)
			containers.ConnectedBusinessObjectsCount = 1
		} else {
			// Measures and Dimensions
			astExpr := ""
			if ast, ok := focalProps["ast_expression"].(string); ok && ast != "" {
				astExpr = ast
			} else if f, ok := focalProps["formula"].(string); ok && f != "" {
				astExpr = f
			}

			norm := strings.ToUpper(focalName)
			isMeasure := strings.Contains(norm, "AMOUNT") || strings.Contains(norm, "TOTAL") ||
				strings.Contains(norm, "PRICE") || strings.Contains(norm, "RATE") ||
				strings.Contains(norm, "COST") || strings.Contains(norm, "VALUE") ||
				strings.Contains(norm, "BALANCE") || strings.Contains(norm, "RETURN") ||
				strings.Contains(norm, "PNL") || strings.Contains(norm, "FEE") ||
				strings.Contains(norm, "MARGIN") || strings.Contains(norm, "VOLUME") ||
				strings.Contains(norm, "COUNT") || strings.Contains(norm, "QUANTITY") ||
				astExpr != ""

			if isMeasure {
				if astExpr == "" {
					astExpr = "SUM(base_metric * allocation_factor)"
				}
				focalNode.Properties["data_type"] = "NUMERIC(18,4)"
				focalNode.Properties["role"] = "CALCULATED MEASURE"
				focalNode.Properties["ast_expression"] = astExpr
				nodes = append(nodes, focalNode)

				u1 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "base_metric_" + focalKey,
					NodeName:   "Base " + focalName,
					NodeType:   "semantic_term",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"data_type": "NUMERIC(18,4)", "role": "Input Metric"},
				}
				u2 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "allocation_factor_" + focalKey,
					NodeName:   "Allocation Factor",
					NodeType:   "semantic_term",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"data_type": "NUMERIC(8,6)", "role": "Weight Multiplier"},
				}
				u3 := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "adjustment_delta_" + focalKey,
					NodeName:   "Adjustment Delta",
					NodeType:   "semantic_term",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"data_type": "NUMERIC(18,4)", "role": "Recon Adjustment"},
				}
				mid := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "unadjusted_" + focalKey,
					NodeName:   "Unadjusted " + focalName,
					NodeType:   "semantic_term",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"ast_expression": "base_metric * allocation_factor", "data_type": "NUMERIC(18,4)"},
				}
				nodes = append(nodes, u1, u2, u3, mid)

				edges = append(edges,
					LensGraphEdge{ID: uuid.New().String(), SourceID: u1.ID, TargetID: mid.ID, EdgeType: "USES_INPUT"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: u2.ID, TargetID: mid.ID, EdgeType: "USES_INPUT"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: u3.ID, TargetID: focalNode.ID, EdgeType: "USES_INPUT"},
					LensGraphEdge{ID: uuid.New().String(), SourceID: mid.ID, TargetID: focalNode.ID, EdgeType: "DERIVED_FROM", Properties: map[string]interface{}{"aggregation": "SUM"}},
				)

				bo := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "bo_" + strings.ToLower(strings.ReplaceAll(categoryStr, " ", "_")),
					NodeName:   categoryStr + " Business Object",
					NodeType:   "business_object",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"entity_type": strings.ToUpper(strings.ReplaceAll(categoryStr, " ", "_"))},
				}
				nodes = append(nodes, bo)
				edges = append(edges, LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: bo.ID, EdgeType: "HAS_FIELD"})
				containers.ConnectedBusinessObjectsCount = 1
			} else {
				// Dimension / Identifier calculation mesh
				focalNode.Properties["role"] = "SEMANTIC DIMENSION"
				if focalNode.Properties["data_type"] == nil {
					focalNode.Properties["data_type"] = "VARCHAR(64)"
				}
				nodes = append(nodes, focalNode)

				src := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "raw_" + focalKey,
					NodeName:   "Raw " + focalName,
					NodeType:   "semantic_term",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"data_type": focalNode.Properties["data_type"], "role": "Source Ingestion Attribute"},
				}
				nodes = append(nodes, src)
				edges = append(edges, LensGraphEdge{ID: uuid.New().String(), SourceID: src.ID, TargetID: focalNode.ID, EdgeType: "TRANSFORMS_TO"})

				bo := LensGraphNode{
					ID:         uuid.New().String(),
					NodeKey:    "bo_" + strings.ToLower(strings.ReplaceAll(categoryStr, " ", "_")),
					NodeName:   categoryStr + " Business Object",
					NodeType:   "business_object",
					SourceTier: "CORE",
					Properties: map[string]interface{}{"entity_type": strings.ToUpper(strings.ReplaceAll(categoryStr, " ", "_"))},
				}
				nodes = append(nodes, bo)
				edges = append(edges, LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: bo.ID, EdgeType: "HAS_DIMENSION"})
				containers.ConnectedBusinessObjectsCount = 1
			}
		}

	case LensPhysicalERD:
		nodes = append(nodes, focalNode)
		entityTable := strings.ToLower(strings.ReplaceAll(categoryStr, " ", "_"))
		if entityTable == "" || entityTable == "general" {
			entityTable = strings.ToLower(strings.ReplaceAll(focalName, " ", "_"))
		}
		colName := strings.ToLower(strings.ReplaceAll(focalName, " ", "_"))
		colType := "VARCHAR(64)"
		if dt, ok := focalProps["data_type"].(string); ok && dt != "" {
			colType = strings.ToUpper(dt)
		}

		t1 := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "tbl_public_" + entityTable,
			NodeName:   "public." + entityTable,
			NodeType:   "database_table",
			SourceTier: "CORE",
			Properties: map[string]interface{}{
				"engine":  "PostgreSQL",
				"schema":  "public",
				"columns": []string{"🔑 " + entityTable + "_id [UUID]", "🔹 " + colName + " [" + colType + "]", "🔹 client_id [UUID]", "🔹 is_active [BOOLEAN]", "🔹 updated_at [TIMESTAMP]"},
			},
		}
		t2 := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "tbl_lakehouse_" + entityTable + "_analytics",
			NodeName:   "lakehouse." + entityTable + "_fact",
			NodeType:   "database_table",
			SourceTier: "CORE",
			Properties: map[string]interface{}{
				"engine":  "StarRocks",
				"schema":  "lakehouse",
				"columns": []string{"🔑 fact_id [BIGINT]", "🔗 " + entityTable + "_id [UUID]", "🔹 " + colName + " [" + colType + "]", "🔹 business_date [DATE]"},
			},
		}
		nodes = append(nodes, t1, t2)
		edges = append(edges,
			LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: t1.ID, EdgeType: "MAPS_TO", Properties: map[string]interface{}{"column": colName}},
			LensGraphEdge{ID: uuid.New().String(), SourceID: t1.ID, TargetID: t2.ID, EdgeType: "JOINS_TO", Properties: map[string]interface{}{"predicate": entityTable + "." + entityTable + "_id = " + entityTable + "_fact." + entityTable + "_id"}},
		)
		containers.PhysicalTableBindingsCount = 2

	case LensPipelineImpact:
		// 4-Tier Pipeline & Impact Lens tailored to domain
		ingestProtocol := "Kafka CDC Event Stream"
		if strings.Contains(strings.ToUpper(domainStr), "TRADING") || strings.Contains(strings.ToUpper(domainStr), "OMS") {
			ingestProtocol = "CRIMS / FIX Order Stream (Kafka CDC)"
		} else if strings.Contains(strings.ToUpper(domainStr), "MARKET") || strings.Contains(strings.ToUpper(domainStr), "SECURITIES") {
			ingestProtocol = "Bloomberg / Market Data Feed (WebSocket / Tick Stream)"
		} else if strings.Contains(strings.ToUpper(domainStr), "CUSTODY") || strings.Contains(strings.ToUpper(domainStr), "ACCOUNTING") {
			ingestProtocol = "SWIFT MT5xx / Bank Depository Ingestion Stream"
		}

		ingest := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "stream_ingest_" + focalKey,
			NodeName:   domainStr + " Ingestion Feed",
			NodeType:   "pipeline_source",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"protocol": ingestProtocol, "rate": "3,800 msg/sec"},
		}
		lakehouse := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "lakehouse_store_" + focalKey,
			NodeName:   "Storage Seam (StarRocks Hot Partition / Apache Iceberg Cold)",
			NodeType:   "database_table",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"hot_tier": "StarRocks >= 30d", "cold_tier": "Apache Iceberg Parquet"},
		}
		contract := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "contract_" + focalKey,
			NodeName:   categoryStr + " Semantic Contract",
			NodeType:   "business_object",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"version": "v1.0", "target_term": focalName},
		}
		consumer := LensGraphNode{
			ID:         uuid.New().String(),
			NodeKey:    "consumer_endpoint_" + focalKey,
			NodeName:   categoryStr + " REST API & Executive BI Dashboard",
			NodeType:   "consumer_endpoint",
			SourceTier: "CORE",
			Properties: map[string]interface{}{"status": "ACTIVE", "impact_risk": "MEDIUM-HIGH"},
		}

		nodes = append(nodes, ingest, lakehouse, focalNode, contract, consumer)
		edges = append(edges,
			LensGraphEdge{ID: uuid.New().String(), SourceID: ingest.ID, TargetID: lakehouse.ID, EdgeType: "INGESTS_TO"},
			LensGraphEdge{ID: uuid.New().String(), SourceID: lakehouse.ID, TargetID: focalNode.ID, EdgeType: "FEEDS_TERM"},
			LensGraphEdge{ID: uuid.New().String(), SourceID: focalNode.ID, TargetID: contract.ID, EdgeType: "COMPOSED_INTO"},
			LensGraphEdge{ID: uuid.New().String(), SourceID: contract.ID, TargetID: consumer.ID, EdgeType: "SERVES_CONSUMER"},
		)
	}

	return &VisualizeLensResponse{
		FocalNodeID:    focalID,
		ActiveLens:     req.LensType,
		BreadcrumbPath: breadcrumb,
		Nodes:          nodes,
		Edges:          edges,
		Containers:     containers,
	}, nil
}

