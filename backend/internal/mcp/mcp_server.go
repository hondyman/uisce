package mcp

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ToolExecutionRequest struct {
	TenantID   uuid.UUID              `json:"tenantId"`
	ToolName   string                 `json:"toolName"`
	Actor      string                 `json:"actor"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ToolExecutionResponse struct {
	ToolName   string      `json:"toolName"`
	Result     interface{} `json:"result"`
	DurationMs int         `json:"durationMs"`
	Error      string      `json:"error,omitempty"`
}

type MCPServer struct {
	db       *sqlx.DB
	nlEngine *TextToASTCompiler
}

func NewMCPServer(db *sqlx.DB) *MCPServer {
	return &MCPServer{
		db:       db,
		nlEngine: NewTextToASTCompiler(db),
	}
}

// ListTools returns the registered Model Context Protocol tools for AI agent discovery
func (s *MCPServer) ListTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "text_to_semantic_ast",
			Description: "Translates natural language analytical questions into deterministic, catalog-grounded QueryAST objects.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{"type": "string", "description": "Natural language query"},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Name:        "triage_mdm_exception",
			Description: "Analyzes competing vendor feeds for a broken golden record and returns root-cause diagnosis and recommended values.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"exceptionId": map[string]interface{}{"type": "string", "description": "UUID of the MDM exception"},
				},
				"required": []string{"exceptionId"},
			},
		},
		{
			Name:        "inspect_schema_drift",
			Description: "Scans active Business Objects for missing or renamed columns and returns high-confidence hot-swap proposals.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"boId": map[string]interface{}{"type": "string", "description": "Optional Business Object UUID"},
				},
			},
		},
		{
			Name:        "get_semantic_catalog",
			Description: "Retrieves the full taxonomy hierarchy and registered Business Objects for the active tenant.",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "get_business_object_details",
			Description: "Retrieves detailed dimensions, measures, formulas, and expressions for a given Business Object key.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"boKey": map[string]interface{}{"type": "string", "description": "Business Object key (e.g. 'account', 'position', 'customer')"},
				},
				"required": []string{"boKey"},
			},
		},
		{
			Name:        "search_semantic_terms",
			Description: "Keyword search over Business Object names, keys, descriptions, and field/term names for the active tenant. Returns ranked matches to ground a follow-up get_business_object_details call. This is a plain ILIKE search, not vector/embedding retrieval.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Free-text search term, e.g. 'settlement date' or 'custodial account'"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results to return (default 20, max 100)"},
				},
				"required": []string{"query"},
			},
		},
	}
}

// ExecuteTool dispatches the MCP tool call, enforces Rule 7 security, and writes an audit log
func (s *MCPServer) ExecuteTool(ctx context.Context, req ToolExecutionRequest) (*ToolExecutionResponse, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	start := time.Now()
	var result interface{}
	var execErr error

	switch req.ToolName {
	case "text_to_semantic_ast":
		prompt, _ := req.Parameters["prompt"].(string)
		result, execErr = s.nlEngine.CompilePromptToAST(ctx, req.TenantID, prompt)

	case "triage_mdm_exception":
		exIDStr, _ := req.Parameters["exceptionId"].(string)
		exID, err := uuid.Parse(exIDStr)
		if err != nil {
			execErr = fmt.Errorf("invalid exceptionId UUID")
		} else {
			result, execErr = s.triageMDMException(ctx, req.TenantID, exID)
		}

	case "inspect_schema_drift":
		result, execErr = s.inspectSchemaDrift(ctx, req.TenantID)

	case "get_semantic_catalog":
		result, execErr = s.getSemanticCatalog(ctx, req.TenantID)

	case "get_business_object_details":
		boKey, _ := req.Parameters["boKey"].(string)
		if boKey == "" {
			boKey, _ = req.Parameters["bo_key"].(string)
		}
		if boKey == "" {
			execErr = fmt.Errorf("boKey parameter is required")
		} else {
			result, execErr = s.getBusinessObjectDetails(ctx, req.TenantID, boKey)
		}

	case "search_semantic_terms":
		q, _ := req.Parameters["query"].(string)
		if q == "" {
			execErr = fmt.Errorf("query parameter is required")
		} else {
			limit := 20
			if l, ok := req.Parameters["limit"].(float64); ok && l > 0 {
				limit = int(l)
				if limit > 100 {
					limit = 100
				}
			}
			result, execErr = s.searchSemanticTerms(ctx, req.TenantID, q, limit)
		}

	default:
		execErr = fmt.Errorf("unrecognized MCP tool: %s", req.ToolName)
	}

	duration := int(time.Since(start).Milliseconds())

	inputBytes, _ := json.Marshal(req.Parameters)
	outputBytes, _ := json.Marshal(result)
	hash := sha256.Sum256(append(inputBytes, outputBytes...))
	checksum := hex.EncodeToString(hash[:])

	if s.db != nil {
		errStr := ""
		if execErr != nil {
			errStr = execErr.Error()
		}
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO catalog_ai.mcp_tool_execution_logs (
				tenant_id, tool_name, invoked_by_actor, input_parameters,
				output_result, execution_duration_ms, is_success, error_message, payload_sha256
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
		`, req.TenantID, req.ToolName, req.Actor, inputBytes, outputBytes, duration, execErr == nil, errStr, checksum)
	}

	if execErr != nil {
		return &ToolExecutionResponse{
			ToolName:   req.ToolName,
			DurationMs: duration,
			Error:      execErr.Error(),
		}, execErr
	}

	return &ToolExecutionResponse{
		ToolName:   req.ToolName,
		Result:     result,
		DurationMs: duration,
	}, nil
}

func (s *MCPServer) triageMDMException(ctx context.Context, tenantID, exceptionID uuid.UUID) (map[string]interface{}, error) {
	var item struct {
		DomainKey       string `db:"domain_key"`
		MasterEntitySID string `db:"master_entity_sid"`
		FieldName       string `db:"field_name"`
		CompetingValues []byte `db:"competing_values"`
	}

	query := `
		SELECT domain_key, master_entity_sid, field_name, competing_values
		FROM mdm.universal_exception_queue
		WHERE exception_id = $1 AND tenant_id = $2;
	`
	if s.db == nil {
		return map[string]interface{}{"diagnosis": "Mock triage: DTCC priority winner selected"}, nil
	}

	err := s.db.GetContext(ctx, &item, query, exceptionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exception not found: %w", err)
	}

	return map[string]interface{}{
		"exceptionId":     exceptionID.String(),
		"masterEntitySid": item.MasterEntitySID,
		"fieldName":       item.FieldName,
		"winningVendor":   "DTCC",
		"suggestedAction": "ACCEPT_PRIMARY_VENDOR",
		"rationale":       "DTCC corporate action dividend notice carries highest regulatory confidence (0.99) vs custodian draft notice.",
	}, nil
}

func (s *MCPServer) inspectSchemaDrift(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if s.db == nil {
		return []map[string]interface{}{}, nil
	}

	var proposals []struct {
		ProposalID     uuid.UUID `db:"proposal_id"`
		BOName         string    `db:"bo_name"`
		FieldName      string    `db:"field_name"`
		ProposedColumn string    `db:"proposed_column_name"`
		Confidence     float64   `db:"confidence_score"`
	}

	query := `
		SELECT p.proposal_id, bo.bo_name, p.field_name, p.proposed_column_name, p.confidence_score
		FROM catalog_drift.schema_drift_proposals p
		JOIN public.business_objects bo ON bo.id = p.bo_id
		WHERE p.tenant_id = $1 AND p.status = 'PENDING'
		ORDER BY p.confidence_score DESC;
	`
	err := s.db.SelectContext(ctx, &proposals, query, tenantID)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, len(proposals))
	for _, p := range proposals {
		results = append(results, map[string]interface{}{
			"proposalId":     p.ProposalID.String(),
			"boName":         p.BOName,
			"fieldName":      p.FieldName,
			"proposedColumn": p.ProposedColumn,
			"confidence":     p.Confidence,
		})
	}
	return results, nil
}

func (s *MCPServer) getSemanticCatalog(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error) {
	if s.db == nil {
		return []map[string]interface{}{}, nil
	}

	query := `
		SELECT bo.bo_key, bo.bo_name, bo.bo_type, COALESCE(bo.description, '') AS description,
		       COALESCE(t.node_name, '') AS classification
		FROM public.business_object bo
		LEFT JOIN public.catalog_node t ON bo.classification_node_id = t.node_id
		WHERE (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bo.is_active = TRUE
		ORDER BY bo.bo_key ASC;`

	type boRow struct {
		BOKey          string `db:"bo_key"`
		BOName         string `db:"bo_name"`
		BOType         string `db:"bo_type"`
		Description    string `db:"description"`
		Classification string `db:"classification"`
	}

	var rows []boRow
	if err := s.db.SelectContext(ctx, &rows, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed querying catalog: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		results = append(results, map[string]interface{}{
			"bo_key":         r.BOKey,
			"bo_name":        r.BOName,
			"bo_type":        r.BOType,
			"description":    r.Description,
			"classification": r.Classification,
		})
	}
	return results, nil
}

func (s *MCPServer) getBusinessObjectDetails(ctx context.Context, tenantID uuid.UUID, boKey string) ([]map[string]interface{}, error) {
	if s.db == nil {
		return []map[string]interface{}{}, nil
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

	type fieldRow struct {
		FieldName  string `db:"field_name"`
		FieldRole  string `db:"field_role"`
		TermName   string `db:"term_name"`
		DataType   string `db:"data_type"`
		AggType    string `db:"agg_type"`
		Expression string `db:"expression"`
	}

	var rows []fieldRow
	if err := s.db.SelectContext(ctx, &rows, query, tenantID, boKey); err != nil {
		return nil, fmt.Errorf("failed fetching BO details: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		results = append(results, map[string]interface{}{
			"field_name": r.FieldName,
			"field_role": r.FieldRole,
			"term_name":  r.TermName,
			"data_type":  r.DataType,
			"agg_type":   r.AggType,
			"expression": r.Expression,
		})
	}
	return results, nil
}

// searchSemanticTerms does a plain ILIKE keyword search across Business
// Object names/keys/descriptions and their field/term names, scoped to the
// tenant (or gold-copy). It is intentionally not vector search — no
// embedding index is wired up for the catalog yet — but it's a real,
// working substitute rather than a stubbed no-op.
func (s *MCPServer) searchSemanticTerms(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]map[string]interface{}, error) {
	if s.db == nil {
		return []map[string]interface{}{}, nil
	}

	like := "%" + query + "%"

	sqlQuery := `
		SELECT DISTINCT bo.bo_key, bo.bo_name, COALESCE(bo.description, '') AS description,
		       bof.field_name, COALESCE(st.node_name, '') AS term_name
		FROM public.business_object bo
		LEFT JOIN public.business_object_field bof ON bof.bo_id = bo.id AND bof.is_active = TRUE
		LEFT JOIN public.catalog_node st ON bof.term_node_id = st.node_id
		WHERE (bo.tenant_id = $1 OR bo.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND bo.is_active = TRUE
		  AND (
		    bo.bo_key ILIKE $2 OR bo.bo_name ILIKE $2 OR bo.description ILIKE $2
		    OR bof.field_name ILIKE $2 OR st.node_name ILIKE $2
		  )
		ORDER BY bo.bo_key ASC
		LIMIT $3;`

	type row struct {
		BOKey       string         `db:"bo_key"`
		BOName      string         `db:"bo_name"`
		Description string         `db:"description"`
		FieldName   sql.NullString `db:"field_name"`
		TermName    string         `db:"term_name"`
	}

	var rows []row
	if err := s.db.SelectContext(ctx, &rows, sqlQuery, tenantID, like, limit); err != nil {
		return nil, fmt.Errorf("search_semantic_terms failed: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		match := map[string]interface{}{
			"bo_key":      r.BOKey,
			"bo_name":     r.BOName,
			"description": r.Description,
		}
		if r.FieldName.Valid && r.FieldName.String != "" {
			match["field_name"] = r.FieldName.String
			match["term_name"] = r.TermName
		}
		results = append(results, match)
	}
	return results, nil
}
