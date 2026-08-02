package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/logging"
)

// AuditLogEntry represents an audit log record
type AuditLogEntry struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenantId"`
	Timestamp    time.Time              `json:"timestamp"`
	UserName     string                 `json:"userName"`
	UserEmail    string                 `json:"userEmail"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	ResourceType string                 `json:"resourceType"`
	Details      map[string]interface{} `json:"details"`
}

// AuditLogResponse is the API response structure
type AuditLogResponse struct {
	Entries []AuditLogEntry `json:"entries"`
	Total   int             `json:"total"`
}

// HandleGetAuditLogs returns audit log entries for a tenant/datasource via DataFusion
func HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")
	tenantID := r.URL.Query().Get("tenantId")

	limit := 10
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	datafusionURL := os.Getenv("DATAFUSION_URL")
	if datafusionURL == "" {
		datafusionURL = "http://100.84.50.65:8555"
	}
	dfClient := boresolver.NewDataFusionClient(datafusionURL)

	// Build Query for DataFusion / Apache Iceberg catalog
	whereClause := " WHERE 1=1"
	if startDateStr != "" {
		whereClause += fmt.Sprintf(" AND timestamp >= '%s'", startDateStr)
	}
	if endDateStr != "" {
		whereClause += fmt.Sprintf(" AND timestamp <= '%s'", endDateStr)
	}
	if tenantID != "" {
		whereClause += fmt.Sprintf(" AND tenant_id = '%s'", tenantID)
	}

	sortBy := r.URL.Query().Get("sortBy")
	sortOrder := r.URL.Query().Get("sortOrder")

	allowedSort := map[string]bool{
		"timestamp": true,
		"user_name": true,
		"action":    true,
		"resource":  true,
	}
	if !allowedSort[sortBy] {
		sortBy = "timestamp"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	sqlQuery := fmt.Sprintf(
		"SELECT id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details FROM iceberg.audit.audit_logs%s ORDER BY %s %s LIMIT %d OFFSET %d",
		whereClause, sortBy, sortOrder, limit, offset,
	)

	ctx := r.Context()
	resp, err := dfClient.ExecuteQuery(ctx, tenantID, sqlQuery)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("DataFusion audit query failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuditLogResponse{Entries: []AuditLogEntry{}, Total: 0})
		return
	}

	var entries []AuditLogEntry
	for _, record := range resp.Records {
		if len(record) < 9 {
			continue
		}
		e := AuditLogEntry{
			ID:           fmt.Sprintf("%v", record[0]),
			TenantID:     fmt.Sprintf("%v", record[1]),
			UserName:     fmt.Sprintf("%v", record[3]),
			UserEmail:    fmt.Sprintf("%v", record[4]),
			Action:       fmt.Sprintf("%v", record[5]),
			Resource:     fmt.Sprintf("%v", record[6]),
			ResourceType: fmt.Sprintf("%v", record[7]),
		}
		if tsStr := fmt.Sprintf("%v", record[2]); tsStr != "" {
			if parsedTs, err := time.Parse(time.RFC3339, tsStr); err == nil {
				e.Timestamp = parsedTs
			}
		}
		if detailsStr := fmt.Sprintf("%v", record[8]); detailsStr != "" {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err == nil {
				e.Details = detailsMap
			} else {
				e.Details = map[string]interface{}{"raw": detailsStr}
			}
		}
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuditLogResponse{
		Entries: entries,
		Total:   resp.RowCount,
	})
}
