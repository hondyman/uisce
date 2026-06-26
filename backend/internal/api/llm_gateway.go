package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// LLMGateway orchestrates: NL → Planner LLM → Semantic Query → Executor LLM → SQL → DB
type LLMGateway struct {
	server *Server
}

// LLMProvider abstracts LLM providers used by the gateway (Gemini, etc.)
type LLMProvider interface {
	GenerateSemanticQuery(ctx context.Context, bundle *SemanticBundle, userPrompt string, mode string, region string) (*SemanticQuery, error)
	GenerateSQL(ctx context.Context, bundle *SemanticBundle, q *SemanticQuery) (string, error)
}

// NewLLMGateway creates a new gateway
func NewLLMGateway(srv *Server) *LLMGateway {
	return &LLMGateway{server: srv}
}

// ProcessQuery orchestrates the full NL → SQL → Rows flow
func (gw *LLMGateway) ProcessQuery(
	ctx context.Context,
	tenantID string,
	region string,
	req *SemanticQueryRequest,
) (*SemanticQueryResponse, error) {
	baseResp := &SemanticQueryResponse{
		Datasource: req.Datasource,
		Version:    req.Version,
		Rows:       []interface{}{},
		Count:      0,
	}

	// Step 1: Load semantic bundle (cached)
	bundle, err := gw.loadSemanticBundle(ctx, tenantID, req.Datasource, region, req.Version)
	if err != nil {
		baseResp.Error = fmt.Sprintf("failed to load semantic bundle: %v", err)
		return baseResp, err
	}

	baseResp.Version = bundle.Version

	// Step 2: Call Planner LLM: NL → SemanticQuery JSON
	semQuery, err := gw.callPlannerLLM(ctx, bundle, req.Prompt, req.Mode, region)
	if err != nil {
		baseResp.Error = fmt.Sprintf("planner LLM error: %v", err)
		return baseResp, err
	}

	// Enforce region presence / match per region spec
	if strings.TrimSpace(semQuery.Region) == "" {
		baseResp.Error = "region is required for all semantic operations."
		return baseResp, fmt.Errorf("planner did not include region in semantic query")
	}
	if !strings.EqualFold(strings.TrimSpace(semQuery.Region), strings.TrimSpace(region)) {
		baseResp.Error = fmt.Sprintf("planner returned mismatched region '%s' (expected '%s')", semQuery.Region, region)
		return baseResp, fmt.Errorf("planner returned mismatched region")
	}

	// Step 3: Validate semantic query against bundle
	if err := gw.server.ValidateSemanticQuery(bundle, semQuery); err != nil {
		baseResp.Error = fmt.Sprintf("semantic query validation failed: %v", err)
		return baseResp, err
	}

	// Marshal the semantic query for response
	semQJSON, _ := json.MarshalIndent(semQuery, "", "  ")
	baseResp.SemanticSQL = string(semQJSON)

	// Step 4: Call Executor (LLM): SemanticQuery + bundle → SQL
	sql, err := gw.callExecutorLLM(ctx, bundle, semQuery)
	if err != nil {
		baseResp.Error = fmt.Sprintf("executor LLM error: %v", err)
		return baseResp, err
	}

	baseResp.GeneratedSQL = sql

	// Step 5: Execute SQL against database
	rows, err := gw.executeSQL(ctx, sql)
	if err != nil {
		baseResp.Error = fmt.Sprintf("SQL execution failed: %v", err)
		return baseResp, err
	}

	baseResp.Rows = rows
	baseResp.Count = len(rows)

	return baseResp, nil
}

// loadSemanticBundle retrieves a semantic bundle by datasource name and optional version
func (gw *LLMGateway) loadSemanticBundle(
	ctx context.Context,
	tenantID string,
	datasourceName string,
	region string,
	version string,
) (*SemanticBundle, error) {
	// Query to get the business object
	var boID, boName, dsID, drivingTable string
	var boVersion int

	query := `
		SELECT 
			bo.id, 
			bo.name, 
			bo.datasource_id, 
			bo.driving_table,
			COALESCE(bo.version, 1)
		FROM business_objects bo
		WHERE bo.name = $1 AND bo.tenant_id = $2 AND (bo.region IS NULL OR bo.region = $3)
		LIMIT 1
	`

	row := gw.server.DB.QueryRowContext(ctx, query, datasourceName, tenantID, region)
	if err := row.Scan(&boID, &boName, &dsID, &drivingTable, &boVersion); err != nil {
		return nil, fmt.Errorf("business object not found: %s", datasourceName)
	}

	// Build bundle struct
	bundle := &SemanticBundle{
		BusinessObjectID:   boID,
		BusinessObjectName: boName,
		DatasourceID:       dsID,
		DrivingTable:       drivingTable,
		Version:            fmt.Sprintf("v%d", boVersion),
		Fields:             []SemanticField{},
		Relationships:      []SemanticRelationship{},
	}

	// Query fields for this BO
	fieldsQuery := `
		SELECT 
			id, 
			name, 
			display_name, 
			semantic_term,
			datasource_id,
			table_name,
			column_name
		FROM bo_fields
		WHERE business_object_id = $1 AND tenant_id = $2 AND is_active = true
		ORDER BY name
	`

	fieldRows, err := gw.server.DB.QueryContext(ctx, fieldsQuery, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fields: %w", err)
	}
	defer fieldRows.Close()

	for fieldRows.Next() {
		var field SemanticField
		var displayName, semanticTerm *string

		if err := fieldRows.Scan(
			&field.FieldID,
			&field.Name,
			&displayName,
			&semanticTerm,
			&field.Physical.DatasourceID,
			&field.Physical.Table,
			&field.Physical.Column,
		); err != nil {
			return nil, fmt.Errorf("failed to scan field: %w", err)
		}

		if displayName != nil {
			field.DisplayName = *displayName
		}
		if semanticTerm != nil {
			field.SemanticTerm = *semanticTerm
		}

		bundle.Fields = append(bundle.Fields, field)
	}

	if err = fieldRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fields: %w", err)
	}

	// Load latest semantic snapshot for tenant+region (required by region spec)
	snap, err := gw.loadLatestSnapshot(ctx, tenantID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to load semantic snapshot: %w", err)
	}
	bundle.Snapshot = snap

	return bundle, nil
}

// callPlannerLLM calls the planner LLM to convert NL → semantic query JSON
func (gw *LLMGateway) callPlannerLLM(
	ctx context.Context,
	bundle *SemanticBundle,
	prompt string,
	mode string,
	region string,
) (*SemanticQuery, error) {
	// Use Gemini client if available
	if gw.server.GeminiClient != nil {
		return gw.server.GeminiClient.GenerateSemanticQuery(ctx, bundle, prompt, mode, region)
	}

	// Fallback error if no LLM client configured
	return nil, fmt.Errorf("no LLM client configured (Gemini or other LLM provider required)")
}

// callExecutorLLM calls the executor LLM to convert semantic query → SQL
func (gw *LLMGateway) callExecutorLLM(
	ctx context.Context,
	bundle *SemanticBundle,
	q *SemanticQuery,
) (string, error) {
	// Use Gemini client if available
	if gw.server.GeminiClient != nil {
		return gw.server.GeminiClient.GenerateSQL(ctx, bundle, q)
	}

	// Fallback error if no LLM client configured
	return "", fmt.Errorf("no LLM client configured (Gemini or other LLM provider required)")
}

// executeSQL executes a SQL query and returns rows as generic interface{} slice
func (gw *LLMGateway) executeSQL(
	ctx context.Context,
	sql string,
) ([]interface{}, error) {
	rows, err := gw.server.DB.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan rows into generic format
	var result []interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map for cleaner JSON representation
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			rowMap[col] = values[i]
		}

		result = append(result, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ParseLLMResponse attempts to parse an LLM response as either SemanticQuery, SQL, or error
func ParseLLMResponse(raw string) (*SemanticQuery, string, error) {
	raw = strings.TrimSpace(raw)

	// Try to parse as JSON (SemanticQuery or error object)
	if strings.HasPrefix(raw, "{") {
		var q SemanticQuery
		if err := json.Unmarshal([]byte(raw), &q); err == nil {
			// Successfully parsed as SemanticQuery
			return &q, "", nil
		}

		// Try to parse as error object
		var errObj map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &errObj); err == nil {
			if errMsg, ok := errObj["error"].(string); ok {
				return nil, "", fmt.Errorf("LLM returned error: %s", errMsg)
			}
		}
	}

	// If it looks like SQL (starts with SELECT, INSERT, etc.), treat it as SQL
	upper := strings.ToUpper(strings.TrimSpace(raw))
	if strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "INSERT") ||
		strings.HasPrefix(upper, "UPDATE") ||
		strings.HasPrefix(upper, "DELETE") ||
		strings.HasPrefix(upper, "WITH") {
		return nil, raw, nil
	}

	return nil, "", fmt.Errorf("unparseable LLM response: %s", raw)
}

// loadLatestSnapshot finds the most recent semantic_snapshot for tenant+region in the catalog
func (gw *LLMGateway) loadLatestSnapshot(ctx context.Context, tenantID, region string) (*audit.SemanticSnapshot, error) {
	query := `
		SELECT properties->>'snapshot_id',
		       properties->>'semantic_term_id',
		       properties->>'business_term_id',
		       properties->>'definition',
		       properties->>'version',
		       properties->>'metadata',
		       properties->>'compliance',
		       properties->>'lineage',
		       created_at
		FROM public.catalog_node
		WHERE qualified_path LIKE 'audit/semantic_snapshot/%'
		  AND tenant_id = $1
		  AND properties->>'region' = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	row := gw.server.DB.QueryRowContext(ctx, query, tenantID, region)
	var snapshotID, semanticTermID, businessTermID, definition, versionStr, metadataStr, complianceStr, lineageStr string
	var createdAt sql.NullTime
	if err := row.Scan(&snapshotID, &semanticTermID, &businessTermID, &definition, &versionStr, &metadataStr, &complianceStr, &lineageStr, &createdAt); err != nil {
		return nil, fmt.Errorf("semantic snapshot not found for tenant '%s' region '%s'", tenantID, region)
	}

	var ss audit.SemanticSnapshot
	ss.SnapshotID = snapshotID
	ss.SemanticTermID = semanticTermID
	ss.BusinessTermID = businessTermID
	ss.TenantID = tenantID
	ss.Region = region
	ss.Definition = definition
	// parse version
	if versionStr != "" {
		if v, err := strconv.Atoi(versionStr); err == nil {
			ss.Version = v
		}
	}
	// metadata/compliance/lineage remain as raw strings stored in Metadata/Compliance/Lineage
	if metadataStr != "" {
		ss.Metadata = json.RawMessage(metadataStr)
	}
	if complianceStr != "" {
		ss.Compliance = json.RawMessage(complianceStr)
	}
	if lineageStr != "" {
		ss.Lineage = json.RawMessage(lineageStr)
	}
	if createdAt.Valid {
		ss.Timestamp = createdAt.Time
	}

	return &ss, nil
}
