package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type QueryPreviewFilter struct {
	FieldKey string   `json:"fieldKey"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type QueryPreviewRequest struct {
	BOKey      string               `json:"boKey"`
	BackendID  *uuid.UUID           `json:"backendId,omitempty"`
	Dimensions []string             `json:"dimensions"`
	Measures   []string             `json:"measures"`
	Filters    []QueryPreviewFilter `json:"filters"`
	Limit      int                  `json:"limit"`
}

type QueryPreviewResponse struct {
	CompiledSQL     string                   `json:"compiledSql"`
	TargetBackend   string                   `json:"targetBackend"`
	EstimatedCost   float64                  `json:"estimatedCostUsd"`
	ComplexityScore int                      `json:"complexityScore"`
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
}

// HandleQueryPreview compiles semantic definitions into dialect SQL and executes live data fetch
func (s *Server) HandleQueryPreview(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID uuid.UUID
	if claims != nil && claims.TenantID != "" {
		tenantID, _ = uuid.Parse(claims.TenantID)
	}
	if tenantID == uuid.Nil {
		tenantHeader := r.Header.Get("X-Tenant-ID")
		if tenantHeader != "" {
			tenantID, _ = uuid.Parse(tenantHeader)
		}
	}
	if tenantID == uuid.Nil {
		http.Error(w, "Rule 7 violation: valid tenant context required", http.StatusUnauthorized)
		return
	}

	var req QueryPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 10000 {
		req.Limit = 10000
	}

	// 1. Synthesize Dialect Pushdown SQL with Tenant RLS Protection
	dimCols := make([]string, 0, len(req.Dimensions))
	for _, d := range req.Dimensions {
		dimCols = append(dimCols, fmt.Sprintf("t.%s", d))
	}

	measureCols := make([]string, 0, len(req.Measures))
	for _, m := range req.Measures {
		measureCols = append(measureCols, fmt.Sprintf("SUM(t.%s) AS %s", m, m))
	}

	var selectClause string
	if len(dimCols) > 0 && len(measureCols) > 0 {
		selectClause = strings.Join(append(dimCols, measureCols...), ", ")
	} else if len(dimCols) > 0 {
		selectClause = strings.Join(dimCols, ", ")
	} else if len(measureCols) > 0 {
		selectClause = strings.Join(measureCols, ", ")
	} else {
		selectClause = "*"
	}

	targetTable := req.BOKey
	if targetTable == "" {
		targetTable = "oms.account"
	}

	var whereClauses []string
	whereClauses = append(whereClauses, fmt.Sprintf("t.tenant_id = '%s'", tenantID.String()))
	whereClauses = append(whereClauses, "t.valid_to IS NULL")

	for _, f := range req.Filters {
		if f.FieldKey != "" && len(f.Values) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("t.%s = '%s'", f.FieldKey, f.Values[0]))
		}
	}

	var compiledSQL string
	if len(dimCols) > 0 && len(measureCols) > 0 {
		compiledSQL = fmt.Sprintf("SELECT %s FROM %s t WHERE %s GROUP BY %s LIMIT %d;",
			selectClause, targetTable, strings.Join(whereClauses, " AND "), strings.Join(dimCols, ", "), req.Limit)
	} else {
		compiledSQL = fmt.Sprintf("SELECT %s FROM %s t WHERE %s LIMIT %d;",
			selectClause, targetTable, strings.Join(whereClauses, " AND "), req.Limit)
	}

	// 2. Fetch live rows from database
	rows := make([]map[string]interface{}, 0)
	if s.SQLXDB != nil {
		dbRows, err := s.SQLXDB.QueryxContext(r.Context(), compiledSQL)
		if err == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				rowMap := make(map[string]interface{})
				if err := dbRows.MapScan(rowMap); err == nil {
					rows = append(rows, rowMap)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(QueryPreviewResponse{
		CompiledSQL:     compiledSQL,
		TargetBackend:   "POSTGRES_HOT",
		EstimatedCost:   0.0015,
		ComplexityScore: 12 + (len(req.Dimensions) * 2) + (len(req.Measures) * 3),
		Columns:         append(req.Dimensions, req.Measures...),
		Rows:            rows,
	})
}
