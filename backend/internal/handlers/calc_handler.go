package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type CalcHandler struct {
	hasura *hasuraclient.HasuraClient
}

type CalcCreateInput struct {
	DatasourceID string `json:"datasource_id"`
	ObjectID     string `json:"object_id"`
	Name         string `json:"name"`
	SQL          string `json:"sql_expr"`
	DataType     string `json:"data_type"`
	IsMeasure    bool   `json:"is_measure"`
	Realtime     bool   `json:"realtime"`
}

func NewCalcHandler(hasura *hasuraclient.HasuraClient) *CalcHandler {
	return &CalcHandler{hasura: hasura}
}

func (h *CalcHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in CalcCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Tenant ID from header (mirrors the proposal's JWT check)
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

	mutation := `
	mutation InsertCalc(
	  $tenant_id: uuid!, $datasource_id: uuid!, $object_id: uuid!,
	  $name: String!, $sql_expr: String!, $data_type: String!,
	  $is_measure: Boolean!, $realtime: Boolean!
	) {
	  insert_calc_fields_one(object: {
	    tenant_id: $tenant_id, datasource_id: $datasource_id,
	    object_id: $object_id, name: $name, sql_expr: $sql_expr,
	    data_type: $data_type, is_measure: $is_measure, realtime: $realtime
	  }) { id version }
	}`

	vars := map[string]interface{}{
		"tenant_id":     tenantID,
		"datasource_id": in.DatasourceID,
		"object_id":     in.ObjectID,
		"name":          in.Name,
		"sql_expr":      in.SQL,
		"data_type":     in.DataType,
		"is_measure":    in.IsMeasure,
		"realtime":      in.Realtime,
	}

	result, err := h.hasura.Mutate(mutation, vars)
	if err != nil {
		http.Error(w, fmt.Sprintf("Hasura mutation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 1. Refresh per-object view
	refreshMutation := `
	mutation Refresh($object_id: uuid!) {
	  refresh_calc_view(args: {p_object_id: $object_id}) { void }
	}`
	_, _ = h.hasura.Mutate(refreshMutation, map[string]interface{}{"object_id": in.ObjectID})

	// 2. Invalidate Cube cache (Regenerate)
	// In a real system, we'd trigger the CubeGenerator or call the existing /api/cube/generate/{boID} endpoint.
	// We'll simulate this by returning the result first.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *CalcHandler) Preview(w http.ResponseWriter, r *http.Request) {
	// Future: Implement SQL Preview logic (SELECT <sql> FROM <base_table> LIMIT 5)
	w.WriteHeader(http.StatusNotImplemented)
}
