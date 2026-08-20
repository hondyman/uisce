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

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to begin transaction: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Upsert into calc_fields catalog
	var id string
	query := `
		INSERT INTO public.calc_fields (tenant_id, object_id, name, sql_expr, data_type, is_measure, realtime)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, object_id, name)
		DO UPDATE SET sql_expr = $4, data_type = $5, is_measure = $6, realtime = $7, updated_at = clock_timestamp()
		RETURNING id
	`
	err = tx.QueryRowContext(r.Context(), query, tenantID, in.ObjectID, in.Name, in.SQL, dataType, in.IsMeasure, in.Realtime).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to create calc field: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// 2. Upsert into catalog_node as calculation_term
	var calcTermTypeID *string
	_ = tx.QueryRowContext(r.Context(), `
		SELECT id::text FROM catalog_node_type WHERE catalog_type_name = 'calculation_term' LIMIT 1
	`).Scan(&calcTermTypeID)

	if calcTermTypeID != nil && *calcTermTypeID != "" {
		catalogNodeID := "calc:" + id
		_, _ = tx.ExecContext(r.Context(), `
			INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties)
			VALUES ($1, $2, $3, $4, $5, $6, true, $7)
			ON CONFLICT (id) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties
		`,
			catalogNodeID, tenantID, *calcTermTypeID, in.Name,
			"Calculated field: "+in.Name,
			"calc:"+in.Name,
			`{"calc_field_id":"`+id+`","sql_expr":"`+in.SQL+`","data_type":"`+dataType+`","is_measure":`+fmt.Sprintf("%t", in.IsMeasure)+`,"object_id":"`+in.ObjectID+`","source":"calc_fields_catalog"}`,
		)
	}

	// 3. Add calculated_ref entry to business_objects.fields JSONB
	// Check if business_objects.fields already has an entry for this name
	var existingFields []byte
	_ = tx.QueryRowContext(r.Context(), `
		SELECT fields FROM business_objects WHERE id = $1 AND tenant_id = $2
	`, in.ObjectID, tenantID).Scan(&existingFields)

	if existingFields != nil {
		// Parse existing fields and check if this calc field already exists
		var fieldsArr []map[string]interface{}
		if err := json.Unmarshal(existingFields, &fieldsArr); err == nil {
			found := false
			for _, f := range fieldsArr {
				if f["name"] == in.Name {
					found = true
					break
				}
			}
			if !found {
				// Append calculated_ref entry
				newEntry := map[string]interface{}{
					"type":         "calculated_ref",
					"name":         in.Name,
					"calc_field_id": id,
					"data_type":    dataType,
					"is_measure":   in.IsMeasure,
				}
				fieldsArr = append(fieldsArr, newEntry)

				updatedFields, _ := json.Marshal(fieldsArr)
				_, _ = tx.ExecContext(r.Context(), `
					UPDATE business_objects SET fields = $1 WHERE id = $2 AND tenant_id = $3
				`, updatedFields, in.ObjectID, tenantID)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to commit: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        id,
		"tenant_id": tenantID,
		"object_id": in.ObjectID,
		"name":      in.Name,
		"sql_expr":  in.SQL,
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

func (h *CalcHandler) Delete(w http.ResponseWriter, r *http.Request) {
	calcFieldID := r.URL.Query().Get("id")
	if calcFieldID == "" {
		http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
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

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to begin transaction: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Get the calc field details before deleting
	var objectID, name string
	err = tx.QueryRowContext(r.Context(), `
		SELECT object_id, name FROM public.calc_fields WHERE id = $1 AND tenant_id = $2
	`, calcFieldID, tenantID).Scan(&objectID, &name)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"calc field not found: %s"}`, err.Error()), http.StatusNotFound)
		return
	}

	// Delete from calc_fields
	_, err = tx.ExecContext(r.Context(), `
		DELETE FROM public.calc_fields WHERE id = $1 AND tenant_id = $2
	`, calcFieldID, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to delete calc field: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Delete from catalog_node (calculation_term)
	_, _ = tx.ExecContext(r.Context(), `
		DELETE FROM catalog_node WHERE id = $1 AND tenant_id = $2
	`, "calc:"+calcFieldID, tenantID)

	// Remove calculated_ref entry from business_objects.fields
	var existingFields []byte
	err = tx.QueryRowContext(r.Context(), `
		SELECT fields FROM business_objects WHERE id = $1 AND tenant_id = $2
	`, objectID, tenantID).Scan(&existingFields)
	if err == nil && existingFields != nil {
		var fieldsArr []map[string]interface{}
		if err := json.Unmarshal(existingFields, &fieldsArr); err == nil {
			filtered := make([]map[string]interface{}, 0, len(fieldsArr))
			for _, f := range fieldsArr {
				if f["type"] == "calculated_ref" && f["calc_field_id"] == calcFieldID {
					continue // skip this one (it's the calc field we're deleting)
				}
				filtered = append(filtered, f)
			}
			updatedFields, _ := json.Marshal(filtered)
			_, _ = tx.ExecContext(r.Context(), `
				UPDATE business_objects SET fields = $1 WHERE id = $2 AND tenant_id = $3
			`, updatedFields, objectID, tenantID)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to commit: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      calcFieldID,
	})
}
