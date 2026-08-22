package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DataPeekRequest struct {
	TenantID  uuid.UUID `json:"tenantId"`
	TableName string    `json:"tableName"`
}

type DataPeekResponse struct {
	TableName    string                   `json:"tableName"`
	Columns      []string                 `json:"columns"`
	Rows         []map[string]interface{} `json:"rows"`
	LatencyMs    int64                    `json:"latencyMs"`
	SampledCount int                      `json:"sampledCount"`
}

type DataPeekHandler struct {
	db *sqlx.DB
}

func NewDataPeekHandler(db *sqlx.DB) *DataPeekHandler {
	return &DataPeekHandler{db: db}
}

func (h *DataPeekHandler) HandleDataPeek(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	start := time.Now()

	var req DataPeekRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	// Rule 7 Guard
	if req.TenantID == uuid.Nil || req.TableName == "" {
		http.Error(w, `{"error":"tenantId and tableName are mandatory"}`, http.StatusBadRequest)
		return
	}

	sanitizedTable := strings.ReplaceAll(req.TableName, ";", "")

	if h.db == nil {
		json.NewEncoder(w).Encode(DataPeekResponse{
			TableName:    sanitizedTable,
			Columns:      []string{"account_id", "security_id", "trade_date", "quantity", "base_cost"},
			Rows:         []map[string]interface{}{{"account_id": "CUST_ACC_01", "security_id": "US0378331005", "quantity": 1000.0}},
			LatencyMs:    time.Since(start).Milliseconds(),
			SampledCount: 1,
		})
		return
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s 
		WHERE tenant_id = $1 AND is_deleted = FALSE 
		LIMIT 5;
	`, sanitizedTable)

	rows, err := h.db.QueryxContext(r.Context(), query, req.TenantID.String())
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"peek failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	resultRows := make([]map[string]interface{}, 0)

	for rows.Next() {
		rowMap := make(map[string]interface{})
		if err := rows.MapScan(rowMap); err == nil {
			// Convert bytes to string for JSON serialization
			for k, v := range rowMap {
				if b, ok := v.([]byte); ok {
					rowMap[k] = string(b)
				}
			}
			resultRows = append(resultRows, rowMap)
		}
	}

	json.NewEncoder(w).Encode(DataPeekResponse{
		TableName:    sanitizedTable,
		Columns:      cols,
		Rows:         resultRows,
		LatencyMs:    time.Since(start).Milliseconds(),
		SampledCount: len(resultRows),
	})
}
