package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/boresolver"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SQLGenerationRequest wrapper to match what's expected from frontend if needed
// reusing boresolver type directly for now

// GenerateSQLHandler generates SQL from a BO-based query definition.
//
// Request body must include:
//   - business_object_id: UUID of the business object
//   - selected_fields: Array of Field UUIDs (NOT field names or semantic term codes)
//     Get field IDs from: GET /api/business-objects/{id}/fields
//   - filters: Array of filter clauses (using field UUIDs in fieldId)
//   - limit: Maximum number of rows to return
//
// Example:
//
//	{
//	  "business_object_id": "be7b9e37-5b9b-41fe-ac6e-58465387eb7c",
//	  "selected_fields": ["fdbd3543-9ca2-41f4-927e-a283a00c0d08"],
//	  "filters": [],
//	  "limit": 100,
//	  "datasource_id": "ds-postgres"
//	}
func (s *Server) GenerateSQLHandler(w http.ResponseWriter, r *http.Request) {
	var req boresolver.SQLGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Initialize Generator
	// In a real app, we should reuse a singleton or factory from the Server struct
	// For now, we instantiate on demand.
	// We need logic to pass the correct dialect (e.g. from Datasource config)
	// For MVP, defaulting to Postgres

	// dependency injection: repository
	repo := boresolver.NewPostgresBORepository(sqlx.NewDb(s.DB, "postgres"))

	// Quick validation: ensure selected_fields contain valid BO field IDs
	invalid, err := boresolver.ValidateSelectedFields(repo, req.BusinessObjectID, req.SelectedFields)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to load BO definition for validation: %v", err)
		http.Error(w, "Failed to load BO definition: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(invalid) > 0 {
		logging.GetLogger().Sugar().Warnf("GenerateSQLHandler: invalid selected_fields %v for BO %s", invalid, req.BusinessObjectID)
		http.Error(w, "Invalid selected_fields: "+strings.Join(invalid, ","), http.StatusBadRequest)
		return
	}

	generator, err := boresolver.NewBOSQLGenerator(repo, "postgres")
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create SQL generator: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sql, err := generator.GenerateSQL(req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("SQL Generation failed: %v", err)
		http.Error(w, "Failed to generate SQL: "+err.Error(), http.StatusBadRequest) // 400 because it might be invalid BO/Field
		return
	}

	resp := boresolver.SQLGenerationResponse{
		SQL: sql,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
	}
}

// GenerateSQLFromSemanticHandler generates SQL from a semantic query definition.
//
// This is the new human-friendly API that uses semantic names instead of UUIDs.
//
// Request body example:
//
//	{
//	  "datasource": "customers",
//	  "select": [
//	    {"term": "id", "label": "ID"},
//	    {"term": "address", "label": "Address"}
//	  ],
//	  "filters": [
//	    {"term": "country", "op": "=", "value": "USA"}
//	  ],
//	  "limit": 100
//	}
func (s *Server) GenerateSQLFromSemanticHandler(w http.ResponseWriter, r *http.Request) {
	var req boresolver.SemanticSQLGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get tenant and datasource context from headers
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}
	if datasourceID == "" {
		http.Error(w, "Missing X-Tenant-Datasource-ID header", http.StatusBadRequest)
		return
	}

	// Initialize Generator
	repo := boresolver.NewPostgresBORepository(sqlx.NewDb(s.DB, "postgres"))
	generator, err := boresolver.NewBOSQLGenerator(repo, "postgres")
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create SQL generator: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sql, err := generator.GenerateSQLFromSemantic(&req, tenantID, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Semantic SQL Generation failed: %v", err)
		http.Error(w, "Failed to generate SQL: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := boresolver.SQLGenerationResponse{
		SQL: sql,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
	}
}

// ExecuteSQLRequest wraps SQL execution request
type ExecuteSQLRequest struct {
	SQL              string `json:"sql"`
	Limit            int    `json:"limit"`
	BusinessObjectID string `json:"business_object_id,omitempty"` // Optional: if provided, will auto-route based on BO's datasource
	DatasourceID     string `json:"datasource_id,omitempty"`      // Optional: manual override; if provided, takes precedence over BO lookup
}

// ExecuteResult represents the result of executing generated SQL
type ExecuteResult struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
	Rows []map[string]interface{} `json:"rows"`
	Page struct {
		Limit   int  `json:"limit"`
		Offset  int  `json:"offset"`
		HasNext bool `json:"hasNext"`
	} `json:"page"`
	RowCount   int    `json:"row_count"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// ExecuteSQLHandler executes the generated SQL against the datasource
//
// Request body must include:
//   - sql: The SQL query to execute
//   - limit: Maximum number of rows to return
//   - business_object_id: (Optional) UUID of the business object; will auto-route based on BO's configured datasource
//   - datasource_id: (Optional) Manual datasource override; takes precedence if provided
//
// Routing logic:
// 1. If datasource_id is provided, use it (manual override)
// 2. If business_object_id is provided, look up BO's datasource from metadata and use that
// 3. Otherwise, default to current operating scope (s.DB)
//
// Example:
//
//	{
//	  "sql": "SELECT customer_id, company_name FROM public.customers LIMIT 100",
//	  "limit": 100,
//	  "business_object_id": "be7b9e37-5b9b-41fe-ac6e-58465387eb7c"
//	}
func (s *Server) ExecuteSQLHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecuteSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SQL == "" {
		http.Error(w, "SQL query is required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	// Get tenant ID from header (for audit/logging purposes)
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		logging.GetLogger().Sugar().Warnf("ExecuteSQLHandler: X-Tenant-ID header not provided")
	}

	// Determine which database connection to use
	// Priority: manual datasource_id > BO auto-lookup > default (alpha)
	db := s.DB
	datasourceTarget := "default (alpha)"

	// First, check for manual datasource_id override
	if req.DatasourceID != "" {
		logging.GetLogger().Sugar().Infof("ExecuteSQLHandler: Using manual datasource override: %s", req.DatasourceID)
		if req.DatasourceID == "northwinds" && s.AggregatesDB != nil {
			db = s.AggregatesDB
			datasourceTarget = "northwinds"
		}
	} else if req.BusinessObjectID != "" {
		// Look up BO's datasource from metadata
		logging.GetLogger().Sugar().Infof("ExecuteSQLHandler: Looking up datasource for BO: %s", req.BusinessObjectID)

		// Query the alpha database for the BO's datasource_id and join with source_name
		var boDatasourceID sql.NullString
		var sourceName sql.NullString
		query := `
			SELECT bo.datasource_id::text, tpd.source_name
			FROM public.business_objects bo
			LEFT JOIN public.tenant_product_datasource tpd ON bo.datasource_id = tpd.id
			WHERE bo.id = $1::uuid
			LIMIT 1
		`
		err := s.DB.QueryRowContext(r.Context(), query, req.BusinessObjectID).Scan(&boDatasourceID, &sourceName)
		if err != nil && err != sql.ErrNoRows {
			logging.GetLogger().Sugar().Warnf("ExecuteSQLHandler: Failed to look up BO datasource for %s: %v", req.BusinessObjectID, err)
			// Fall back to default database
		} else if boDatasourceID.Valid && boDatasourceID.String != "" {
			// Route based on the BO's datasource
			boDatasource := boDatasourceID.String
			dataSourceName := "unknown"
			if sourceName.Valid {
				dataSourceName = sourceName.String
			}
			logging.GetLogger().Sugar().Infof("ExecuteSQLHandler: BO %s is tied to datasource: %s (name: %s)", req.BusinessObjectID, boDatasource, dataSourceName)

			// Route to appropriate database based on source_name or known UUID mappings
			if (sourceName.Valid && strings.Contains(strings.ToLower(sourceName.String), "northwinds")) ||
				boDatasource == "1af891c8-8a5c-4788-8fd2-5ae1f271868f" ||
				strings.Contains(strings.ToLower(boDatasource), "northwinds") {
				if s.AggregatesDB != nil {
					db = s.AggregatesDB
					datasourceTarget = "northwinds"
				}
			}
		}
	}

	logging.GetLogger().Sugar().Infof("ExecuteSQLHandler: Routing query to %s for tenant %s", datasourceTarget, tenantID)

	// Execute the SQL query
	rows, err := db.QueryContext(r.Context(), req.SQL)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("SQL execution failed for tenant %s: %v", tenantID, err)
		errResp := ExecuteResult{Error: fmt.Sprintf("Query execution failed: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errResp)
		return
	}
	defer rows.Close()

	// Get column names and types from the result set
	columns, err := rows.Columns()
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get column names: %v", err)
		errResp := ExecuteResult{Error: fmt.Sprintf("Failed to get columns: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to get column types: %v", err)
		// Continue with "unknown" types
	}

	// Build column metadata
	resultColumns := make([]struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}, len(columns))

	for i, colName := range columns {
		resultColumns[i].Name = colName
		resultColumns[i].Type = "unknown"
		if columnTypes != nil && i < len(columnTypes) {
			dbTypeName := columnTypes[i].DatabaseTypeName()
			// Map database types to friendly names
			switch strings.ToLower(dbTypeName) {
			case "bigint", "int8", "integer", "int4", "smallint", "int2":
				resultColumns[i].Type = "integer"
			case "numeric", "decimal", "real", "double precision":
				resultColumns[i].Type = "decimal"
			case "boolean", "bool":
				resultColumns[i].Type = "boolean"
			case "text", "character varying", "varchar", "char":
				resultColumns[i].Type = "string"
			case "date":
				resultColumns[i].Type = "date"
			case "timestamp", "timestamp with time zone", "timestamp without time zone":
				resultColumns[i].Type = "timestamp"
			case "json", "jsonb":
				resultColumns[i].Type = "json"
			default:
				resultColumns[i].Type = "unknown"
			}
		}
	}

	// Scan rows
	var data []map[string]interface{}
	for rows.Next() {
		// Create a slice of empty interfaces
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to scan row: %v", err)
			continue
		}

		// Build map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert bytes to string for JSON friendliness
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		data = append(data, row)
	}

	if err = rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("Error reading rows: %v", err)
		errResp := ExecuteResult{Error: fmt.Sprintf("Error reading results: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Build response
	result := ExecuteResult{
		Columns:  resultColumns,
		Rows:     data,
		RowCount: len(data),
	}
	result.Page.Limit = req.Limit
	result.Page.Offset = 0
	result.Page.HasNext = false

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
