package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type CalcHandler struct {
	db *sqlx.DB
}

type CalcCreateInput struct {
	DatasourceID string `json:"datasource_id"`
	ObjectID    string `json:"object_id"`
	Name        string `json:"name"`
	SQL         string `json:"sql_expr"`
	DataType    string `json:"data_type"`
	IsMeasure   bool   `json:"is_measure"`
	Realtime    bool   `json:"realtime"`
}

type CalcField struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	ObjectID  string `json:"object_id"`
	Name      string `json:"name"`
	SQLExpr   string `json:"sql_expr"`
	DataType  string `json:"data_type"`
	IsMeasure bool   `json:"is_measure"`
	Realtime  bool   `json:"realtime"`
}

type PreviewRequest struct {
	ObjectID string `json:"object_id"`
	SQLExpr  string `json:"sql_expr"`
	Limit    int    `json:"limit"`
}

type PreviewResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func NewCalcHandler(db *sqlx.DB) *CalcHandler {
	return &CalcHandler{db: db}
}

func (h *CalcHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in CalcCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusUnauthorized)
		return
	}

	if in.ObjectID == "" {
		http.Error(w, `{"error":"object_id is required"}`, http.StatusBadRequest)
		return
	}
	if in.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if in.SQL == "" {
		http.Error(w, `{"error":"sql_expr is required"}`, http.StatusBadRequest)
		return
	}

	dataType := in.DataType
	if dataType == "" {
		dataType = "number"
	}

	var id string
	query := `
		INSERT INTO public.calc_fields (tenant_id, object_id, name, sql_expr, data_type, is_measure, realtime)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, object_id, name) 
		DO UPDATE SET sql_expr = $4, data_type = $5, is_measure = $6, realtime = $7, updated_at = clock_timestamp()
		RETURNING id
	`
	err := h.db.QueryRowContext(r.Context(), query, tenantID, in.ObjectID, in.Name, in.SQL, dataType, in.IsMeasure, in.Realtime).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to create calc field: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       id,
		"tenant_id": tenantID,
		"object_id": in.ObjectID,
		"name":     in.Name,
		"sql_expr": in.SQL,
		"data_type": dataType,
		"is_measure": in.IsMeasure,
		"realtime":  in.Realtime,
	})
}

func (h *CalcHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusUnauthorized)
		return
	}

	if req.ObjectID == "" {
		http.Error(w, `{"error":"object_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.SQLExpr == "" {
		http.Error(w, `{"error":"sql_expr is required"}`, http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	var columns []string
	var rows [][]string

	columns = []string{"result"}
	
	query := fmt.Sprintf("SELECT %s as result LIMIT %d", req.SQLExpr, limit)
	
	dbRows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to preview: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer dbRows.Close()

	for dbRows.Next() {
		var result string
		if err := dbRows.Scan(&result); err != nil {
			continue
		}
		rows = append(rows, []string{fmt.Sprintf("%v", result)})
	}

	if rows == nil {
		rows = [][]string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"columns": columns,
		"rows":    rows,
	})
}

func (h *CalcHandler) GetByObjectID(w http.ResponseWriter, r *http.Request) {
	objectID := r.URL.Query().Get("object_id")
	if objectID == "" {
		http.Error(w, `{"error":"object_id is required"}`, http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusUnauthorized)
		return
	}

	var fields []CalcField
	query := `
		SELECT id, tenant_id, object_id, name, sql_expr, data_type, is_measure, realtime
		FROM public.calc_fields
		WHERE tenant_id = $1 AND object_id = $2
		ORDER BY name
	`
	err := h.db.SelectContext(r.Context(), &fields, query, tenantID, objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to get calc fields: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if fields == nil {
		fields = []CalcField{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": fields,
	})
}
