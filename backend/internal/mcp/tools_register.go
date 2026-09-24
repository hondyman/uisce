package mcp

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func (s *Server) registerDefaultTools() {
	s.RegisterTool("get_business_object_contract",
		"Returns fields, formulas, data types, and additivity rules for a governed Business Object. Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"bo_id":  map[string]string{"type": "string"},
				"bo_key": map[string]string{"type": "string"},
			},
		},
		s.path1.getBusinessObjectContract,
	)
	s.RegisterTool("resolve_relationship_path",
		"Finds certified join paths between two Business Objects or catalog nodes via catalog_edge. Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"source_node_id": map[string]string{"type": "string"},
				"target_node_id": map[string]string{"type": "string"},
			},
			"required": []string{"source_node_id", "target_node_id"},
		},
		s.path1.resolveRelationshipPath,
	)
	s.RegisterTool("list_business_objects",
		"Lists published Business Objects for the tenant (bound objects only; not the raw catalog). Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		s.path1.listBusinessObjects,
	)
	s.RegisterTool("get_bo_terms",
		"Returns semantic terms for one Business Object (same fence Page Studio uses). Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"bo_id":  map[string]string{"type": "string"},
				"bo_key": map[string]string{"type": "string"},
			},
		},
		s.path1.getBOTerms,
	)
	s.RegisterTool("list_pages",
		"Lists Page Studio page definitions for the tenant (id, name, slug, status). Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		s.path1.listPages,
	)
	s.RegisterTool("get_page",
		"Returns one page definition (layout, data sources, presentation events). No record data. Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page_id": map[string]string{"type": "string"},
				"slug":    map[string]string{"type": "string"},
			},
		},
		s.path1.getPage,
	)
	s.RegisterTool("compile_semantic_query",
		"Compiles a certified, tenant-fenced SQL query across hot/cold lakehouse tiers (StarRocks/Iceberg). Tenant is extracted from the authenticated JWT.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"effective_start_date": map[string]string{"type": "string"},
				"effective_end_date":   map[string]string{"type": "string"},
				"watermark_date":       map[string]string{"type": "string"},
				"hot_table_name":       map[string]string{"type": "string"},
				"cold_table_name":      map[string]string{"type": "string"},
				"temporal_column":      map[string]string{"type": "string"},
				"business_key_columns": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
				"selected_columns":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			},
			"required": []string{"effective_start_date", "effective_end_date", "watermark_date", "hot_table_name", "cold_table_name"},
		},
		func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error) {
			var req boresolver.BitemporalRangeRequest
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, err
			}
			req.TenantID = tenantID
			return s.path1.compiler.CompileRangeQuery(ctx, req)
		},
	)

	s.registerCatalogTools()
}
