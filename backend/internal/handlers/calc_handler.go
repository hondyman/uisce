package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	jwtmiddleware "github.com/hondyman/semlayer/libs/jwt-middleware"
)

type CalcHandler struct {
	db *sqlx.DB
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

func NewCalcHandler(db *sqlx.DB) *CalcHandler {
	return &CalcHandler{db: db}
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

	calcID := uuid.New().String()
	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO calc_fields (
			id, tenant_id, datasource_id, object_id, name, sql_expr,
			data_type, is_measure, realtime, version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, 1, NOW(), NOW()
		)
	`, calcID, tenantID, in.DatasourceID, in.ObjectID, in.Name, in.SQL,
		in.DataType, in.IsMeasure, in.Realtime)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create calc field: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      calcID,
		"version": 1,
	})
}

func (h *CalcHandler) Preview(w http.ResponseWriter, r *http.Request) {
	// Future: Implement SQL Preview logic (SELECT <sql> FROM <base_table> LIMIT 5)
	w.WriteHeader(http.StatusNotImplemented)
}
