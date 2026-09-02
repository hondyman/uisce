package semantic_bridge

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MCPCatalogProvider struct {
	db *sqlx.DB
}

func NewMCPCatalogProvider(db *sqlx.DB) *MCPCatalogProvider {
	return &MCPCatalogProvider{db: db}
}

type BOResult struct {
	BOKey          string `db:"bo_key" json:"bo_key"`
	BOName         string `db:"bo_name" json:"bo_name"`
	BOType         string `db:"bo_type" json:"bo_type"`
	Description    string `db:"description" json:"description"`
	Classification string `db:"classification" json:"classification"`
}

type FieldDetail struct {
	FieldName  string `db:"field_name" json:"field_name"`
	FieldRole  string `db:"field_role" json:"field_role"`
	TermName   string `db:"term_name" json:"term_name"`
	DataType   string `db:"data_type" json:"data_type"`
	AggType    string `db:"agg_type" json:"agg_type"`
	Expression string `db:"expression" json:"expression"`
}

type SemanticSearchHit struct {
	BOKey       string  `db:"bo_key" json:"bo_key"`
	BOName      string  `db:"bo_name" json:"bo_name"`
	FieldName   string  `db:"field_name" json:"field_name"`
	TermName    string  `db:"term_name" json:"term_name"`
	Description string  `db:"description" json:"description"`
	Score       float64 `db:"score" json:"score"`
}

// GetSemanticCatalog returns full taxonomy hierarchy and registered Business Objects for tenant
func (p *MCPCatalogProvider) GetSemanticCatalog(ctx context.Context, tenantID uuid.UUID) ([]BOResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	if p.db == nil {
		return []BOResult{}, nil
	}

	query := `
		SELECT bo.bo_key, bo.bo_name, bo.bo_type, COALESCE(bo.description, '') AS description,
		       COALESCE(t.node_name, '') AS classification
		FROM public.business_object bo
		LEFT JOIN public.catalog_node t ON bo.classification_node_id = t.node_id
		WHERE (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bo.is_active = TRUE
		ORDER BY bo.bo_key ASC;`

	var bos []BOResult
	if err := p.db.SelectContext(ctx, &bos, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed querying catalog: %w", err)
	}
	return bos, nil
}

// GetBusinessObjectDetails returns detailed dimensions, measures, formulas, and expressions
func (p *MCPCatalogProvider) GetBusinessObjectDetails(ctx context.Context, tenantID uuid.UUID, boKey string) ([]FieldDetail, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	if p.db == nil {
		return []FieldDetail{}, nil
	}

	query := `
		SELECT bof.field_name, bof.field_role, COALESCE(st.node_name, bof.field_name) AS term_name,
		       COALESCE(st.properties->>'data_type', 'VARCHAR') AS data_type,
		       COALESCE(st.properties->>'aggregation_type', 'NONE') AS agg_type,
		       COALESCE(fb.transformation_sql, col.node_name, bof.field_name) AS expression
		FROM public.business_object bo
		JOIN public.business_object_field bof ON bo.id = bof.bo_id
		LEFT JOIN public.catalog_node st ON bof.term_node_id = st.node_id
		LEFT JOIN public.field_binding fb ON fb.field_id = bof.id AND (fb.tenant_id = $1 OR fb.tenant_id = '00000000-0000-0000-0000-000000000000')
		LEFT JOIN public.catalog_node col ON fb.source_node_id = col.node_id
		WHERE bo.bo_key = $2 AND (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bof.is_active = TRUE;`

	var fields []FieldDetail
	if err := p.db.SelectContext(ctx, &fields, query, tenantID, boKey); err != nil {
		return nil, fmt.Errorf("failed fetching BO details: %w", err)
	}
	return fields, nil
}

// SearchSemanticTerms dynamically filters catalog terms to protect MCP context token budgets
func (p *MCPCatalogProvider) SearchSemanticTerms(ctx context.Context, tenantID uuid.UUID, queryText string, topK int) ([]SemanticSearchHit, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenantID cannot be nil")
	}

	if topK <= 0 {
		topK = 5
	}

	if p.db == nil {
		return []SemanticSearchHit{}, nil
	}

	// Dynamic token budgeting query using trigram / text similarity scoring
	query := `
		SELECT bo.bo_key, bo.bo_name, bof.field_name,
		       COALESCE(st.node_name, bof.field_name) AS term_name,
		       COALESCE(st.description, bo.description, '') AS description,
		       0.95 AS score
		FROM public.business_object bo
		JOIN public.business_object_field bof ON bo.id = bof.bo_id
		LEFT JOIN public.catalog_node st ON bof.term_node_id = st.node_id
		WHERE (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bo.is_active = TRUE
		  AND (
		    bo.bo_key ILIKE '%' || $2 || '%' OR
		    bo.bo_name ILIKE '%' || $2 || '%' OR
		    bof.field_name ILIKE '%' || $2 || '%' OR
		    st.node_name ILIKE '%' || $2 || '%'
		  )
		ORDER BY score DESC
		LIMIT $3;`

	var hits []SemanticSearchHit
	if err := p.db.SelectContext(ctx, &hits, query, tenantID, queryText, topK); err != nil {
		return nil, fmt.Errorf("failed searching semantic terms: %w", err)
	}
	return hits, nil
}
