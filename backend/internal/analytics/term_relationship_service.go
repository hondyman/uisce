package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

	// Aliases are the physical column names (across tables/datasources) that
	// resolve to this term, so an MCP client can map a raw column back to the
	// concept it represents without a separate lookup.
	Aliases []string `json:"aliases,omitempty"`
	// Synonyms are alternate business names for the same concept: curated via
	// catalog_node.properties["synonyms"], plus any RELATES_TO peer whose edge
	// is tagged properties.relationship_subtype = "synonym".
	Synonyms []string `json:"synonyms,omitempty"`
	// RelatedTerms is the full relationship graph for this term (role,
	// relationship type, confidence, differentiation notes) — richer than the
	// flattened ParentTerm/PeerIdentifiers/SpecializedSubTerms fields above,
	// which stay for prompt-block/backward-compat use.
	RelatedTerms []RelatedTermInfo `json:"related_terms,omitempty"`
}

// AIGraphEdge represents an associative or hierarchical connection in the AI context.
type AIGraphEdge struct {
	SourceTerm  string `json:"source_term" db:"source_term"`
	Predicate   string `json:"predicate" db:"predicate"`
	TargetTerm  string `json:"target_term" db:"target_term"`
	Explanation string `json:"explanation,omitempty" db:"explanation"`
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

	rejectionStoreOnce   sync.Once
	rejectionStoreExists bool
}

// hasRejectionStore reports whether catalog_edge_rejection_store exists in
// this database. It's optional infrastructure (backend/db/migrations/20260821_
// cognitive_graph_edge_taxonomy.sql, a directory this deployment's migration
// runner may not scan — confirmed absent against the dev instance), so
// queries that filter against it must check first rather than assume it's
// there: referencing a missing table fails the whole query at parse time,
// not just the subquery, even inside a conditional.
func (s *TermRelationshipService) hasRejectionStore(ctx context.Context) bool {
	s.rejectionStoreOnce.Do(func() {
		if s.db == nil {
			return
		}
		var regclass sql.NullString
		if err := s.db.GetContext(ctx, &regclass, `SELECT to_regclass('public.catalog_edge_rejection_store')::text`); err == nil {
			s.rejectionStoreExists = regclass.Valid && regclass.String != ""
		}
	})
	return s.rejectionStoreExists
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
			  AND (tenant_id = $2 OR tenant_id = $3 OR tenant_id = '00000000-0000-0000-0000-000000000000' OR $2::text = '')
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
			  AND (tenant_id = $2 OR tenant_id = $3 OR tenant_id = '00000000-0000-0000-0000-000000000000' OR $2::text = '')
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

	// NOTE: catalog_edge stores relationships via source_node_id/target_node_id
	// and a UUID edge_type_id (FK into catalog_edge_types); it has no
	// edge_type_name or source_id/target_id columns of its own. edge_type_name
	// is resolved by joining catalog_edge_type, the back-compat view over
	// catalog_edge_types (see resolveOrCreateEdgeType below for the same join).
	rejectionFilter := ""
	if s.hasRejectionStore(ctx) {
		rejectionFilter = `
			  AND NOT EXISTS (
				  SELECT 1 FROM catalog_edge_rejection_store r
				  WHERE ($2::text != '' AND r.tenant_id = CAST($2 AS UUID))
				    AND ( (r.source_node_id = ce.source_node_id AND r.rejected_target_id = ce.target_node_id) OR (r.source_node_id = ce.target_node_id AND r.rejected_target_id = ce.source_node_id) )
			  )`
	}
	edgeQuery := `
		WITH combined_edges AS (
			SELECT
				ce.id, ce.source_node_id, ce.target_node_id, cet.edge_type_name, ce.properties, ce.tenant_id,
				ROW_NUMBER() OVER (
					PARTITION BY ce.source_node_id, ce.target_node_id, ce.edge_type_id
					ORDER BY CASE WHEN ce.tenant_id = $2 THEN 1 ELSE 2 END
				) AS precedence_rank
			FROM catalog_edge ce
			JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
			WHERE (ce.source_node_id = $1 OR ce.target_node_id = $1)
			  AND (ce.tenant_id = $2 OR ce.tenant_id = $3 OR ce.tenant_id = '00000000-0000-0000-0000-000000000000' OR $2::text = '')
			  AND ce.is_active = true` + rejectionFilter + `
		)
		SELECT
			ce.id, ce.source_node_id AS source_id, ce.target_node_id AS target_id, ce.edge_type_name, ce.properties,
			cn.id as other_node_id, cn.node_name as other_name, cn.qualified_path as other_path,
			cn.description as other_desc, cn.properties as other_props, cn.tenant_id as other_tenant,
			(ce.source_node_id = $1) as is_outgoing
		FROM combined_edges ce
		JOIN catalog_node cn ON (
			CASE
				WHEN ce.source_node_id = $1 THEN ce.target_node_id = cn.id
				ELSE ce.source_node_id = cn.id
			END
		)
		WHERE ce.precedence_rank = 1
		  AND (cn.tenant_id = $2 OR cn.tenant_id = $3 OR cn.tenant_id = '00000000-0000-0000-0000-000000000000' OR $2::text = '')
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
			Synonyms:             getRelStringSliceProp(props, "synonyms"),
			Aliases:              s.fetchTermAliases(ctx, tenantID, goldTenantID, node.ID),
		}

		// Query relationship context for this node
		disambig, _ := s.GetRelatedTerms(ctx, tenantID, datasourceID, node.ID)
		if disambig != nil {
			termDef.RelatedTerms = disambig.RelatedTerms
			for _, r := range disambig.RelatedTerms {
				if r.Role == "Parent (Generalization)" {
					termDef.ParentTerm = r.TermName
				} else if r.Role == "Peer Symbology / Alternate ID" {
					termDef.PeerIdentifiers = append(termDef.PeerIdentifiers, r.TermName)
				} else if strings.Contains(r.Role, "Specialization") {
					termDef.SpecializedSubTerms = append(termDef.SpecializedSubTerms, r.TermName)
				}
				// A RELATES_TO edge explicitly tagged as a synonym relationship
				// contributes the peer's name to Synonyms, not just RelatedTerms.
				if r.RelationshipType == EdgeTypeRelatesTo {
					if sub, ok := r.Properties["relationship_subtype"].(string); ok && strings.EqualFold(sub, "synonym") {
						termDef.Synonyms = append(termDef.Synonyms, r.TermName)
					}
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
		if len(termDef.Aliases) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Physical Column Aliases**: %s\n", strings.Join(termDef.Aliases, ", ")))
		}
		if len(termDef.Synonyms) > 0 {
			promptBuilder.WriteString(fmt.Sprintf("- **Synonyms**: %s\n", strings.Join(termDef.Synonyms, ", ")))
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
			cet.edge_type_name as predicate,
			c2.node_name as target_term,
			COALESCE(ce.properties->>'differentiation', ce.properties->>'key_distinction', '') as explanation
		FROM catalog_edge ce
		JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
		JOIN catalog_node c1 ON ce.source_node_id = c1.id
		JOIN catalog_node c2 ON ce.target_node_id = c2.id
		WHERE (ce.tenant_id = $1 OR ce.tenant_id = $2)
		  AND ce.is_active = true
	`
	if err := s.db.SelectContext(ctx, &edges, edgeQuery, tenantID, goldTenantID); err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch taxonomy edges for AI context: %v", err)
	}

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

	// NOTE: catalog_edge has no edge_type_name column of its own — edge_type_id
	// (resolved above) is the only source of truth; edge_type_name is a
	// property of catalog_edge_type/catalog_edge_types, not of the edge row.
	//
	// catalog_edge is partitioned with PRIMARY KEY (id, created_at) only — there
	// is no unique constraint on (tenant_id, source_node_id, target_node_id,
	// edge_type_id), so ON CONFLICT against that tuple errors at runtime
	// ("no unique or exclusion constraint matching the ON CONFLICT
	// specification"), confirmed against the live instance. Do the
	// find-then-insert-or-update explicitly instead.
	var existingID string
	err = s.db.GetContext(ctx, &existingID, `
		SELECT id FROM catalog_edge
		WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
		LIMIT 1
	`, tenantID, sourceTermID, targetTermID, edgeTypeID)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to look up existing relationship edge: %w", err)
	}

	if existingID != "" {
		_, err = s.db.ExecContext(ctx, `
			UPDATE catalog_edge
			SET properties = $2::jsonb, is_active = true, updated_at = NOW()
			WHERE id = $1
		`, existingID, string(propsJSON))
		if err != nil {
			return "", fmt.Errorf("failed to update existing relationship edge: %w", err)
		}
		return existingID, nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (
			id, tenant_id, tenant_datasource_id,
			source_node_id, target_node_id,
			edge_type_id,
			properties, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6,
			$7::jsonb, true, NOW(), NOW()
		)
	`, edgeID, tenantID, nullDsID, sourceTermID, targetTermID, edgeTypeID, string(propsJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create relationship edge: %w", err)
	}
	return edgeID, nil
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
	return resolveOrCreateEdgeTypeID(ctx, s.db, tenantID, edgeTypeName)
}

// resolveOrCreateEdgeTypeID looks up the UUID of a catalog_edge_type row by
// name (preferring a tenant-scoped row, falling back to any tenant's, e.g. a
// shared/gold-copy definition), creating a tenant-scoped row if the name is
// unknown anywhere. Shared by TermRelationshipService and
// SemanticMappingService so both write through the same edge-type vocabulary
// instead of each maintaining their own resolution logic.
func resolveOrCreateEdgeTypeID(ctx context.Context, db *sqlx.DB, tenantID, edgeTypeName string) (string, error) {
	if db == nil {
		return uuid.Nil.String(), nil
	}

	var edgeTypeID string
	if err := db.GetContext(ctx, &edgeTypeID, `
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

	if err := db.GetContext(ctx, &edgeTypeID, `
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

// getRelStringSliceProp reads a curated string-array property (e.g.
// properties["synonyms"] or properties["aliases"]), tolerating both a real
// JSON array and, defensively, a single comma-separated string.
func getRelStringSliceProp(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		var out []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// fetchTermAliases returns the distinct physical column/node names linked to
// termNodeID via a column-mapping edge (has_semantic, provides_context_to, or
// the STI exact-name IS_CLASSIFIED_AS link), in either direction. These are
// the raw names an MCP client or column scanner would actually see, so a
// lookup from a physical column name can resolve straight back to this term.
func (s *TermRelationshipService) fetchTermAliases(ctx context.Context, tenantID, goldTenantID, termNodeID string) []string {
	if s.db == nil {
		return nil
	}
	var names []string
	query := `
		SELECT DISTINCT cn.node_name
		FROM catalog_edge ce
		JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id
		JOIN catalog_node cn ON cn.id = (CASE WHEN ce.source_node_id = $1 THEN ce.target_node_id ELSE ce.source_node_id END)
		WHERE (ce.source_node_id = $1 OR ce.target_node_id = $1)
		  AND ce.is_active = true
		  AND (ce.tenant_id = $2 OR ce.tenant_id = $3 OR ce.tenant_id = '00000000-0000-0000-0000-000000000000' OR $2::text = '')
		  AND cet.edge_type_name IN ('has_semantic', 'provides_context_to', 'IS_CLASSIFIED_AS')
		  AND cn.node_name IS NOT NULL AND cn.node_name != ''
		ORDER BY cn.node_name
		LIMIT 50
	`
	if err := s.db.SelectContext(ctx, &names, query, termNodeID, tenantID, goldTenantID); err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch aliases for term %s: %v", termNodeID, err)
		return nil
	}
	return names
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

// classifiedByEdgeType is the edge type actually seeded in the live catalog
// for linking a business term to its classification node. The literal UUID
// '66666666-6666-6666-6666-666666660001' this function used to hardcode
// (from backend/migrations/20260821_3tier_taxonomy_and_classified_by.sql,
// named 'CLASSIFIED_BY', targeting a Tier-3 node) does not exist in the live
// instance — that migration's tenant/node-type lookups evidently missed and
// it no-op'd. The edge type that IS actually seeded and bound to
// business_term as its source is 'classified_by_l4', targeting Classification_L4
// (confirmed against the running instance), so despite this function's
// original "L3" naming, it links to a Tier-4 classification node in practice.
const classifiedByEdgeType = "classified_by_l4"

// ClassifyTerm links a business term (or semantic term) to a classification
// node via the tenant's classifiedByEdgeType.
func (s *TermRelationshipService) ClassifyTerm(ctx context.Context, tenantID, termID, l3NodeID string) error {
	if s.db == nil {
		return nil
	}

	edgeTypeID, err := resolveOrCreateEdgeTypeID(ctx, s.db, tenantID, classifiedByEdgeType)
	if err != nil {
		return fmt.Errorf("failed to resolve %s edge type: %w", classifiedByEdgeType, err)
	}

	// `id` has no default and is NOT NULL (confirmed against the live
	// instance), and there is no unique constraint on (tenant_id,
	// source_node_id, target_node_id, edge_type_id) to ON CONFLICT against —
	// catalog_edge is partitioned with PRIMARY KEY (id, created_at) only. Do
	// an explicit find-then-insert-or-touch instead.
	var existingID string
	err = s.db.GetContext(ctx, &existingID, `
		SELECT id FROM catalog_edge
		WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4
		LIMIT 1
	`, tenantID, termID, l3NodeID, edgeTypeID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to look up existing %s edge: %w", classifiedByEdgeType, err)
	}
	if existingID != "" {
		_, err = s.db.ExecContext(ctx, `UPDATE catalog_edge SET updated_at = NOW() WHERE id = $1`, existingID)
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (
			id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, '{"tier": "L4"}'::jsonb, true, NOW(), NOW()
		)
	`, uuid.New().String(), tenantID, termID, l3NodeID, edgeTypeID)
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
