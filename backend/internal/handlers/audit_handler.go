package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
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
	datasourceId := r.URL.Query().Get("datasourceId")

	var tenantID string
	var isGlobalAdmin bool
	var userTenantIDs []string

	if authInfo, ok := security.AuthInfoFromContext(r.Context()); ok && authInfo.UserID != "" {
		if len(authInfo.TenantIDs) > 0 {
			tenantID = authInfo.TenantIDs[0]
		}
		isGlobalAdmin = authInfo.IsGlobalAdmin
		userTenantIDs = authInfo.TenantIDs
	} else if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.UserID != "" {
		tenantID = claims.TenantID
		isGlobalAdmin = claims.IsCoreAdmin
		userTenantIDs = append([]string{claims.TenantID}, claims.TenantIDs...)
	} else {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Allow the caller to request audit logs for a specific tenant (e.g. an
	// admin viewing a tenant detail page that isn't their own default tenant),
	// but only if the JWT/Auth context proves they actually have access to it.
	if requestedTenantID := r.URL.Query().Get("tenantId"); requestedTenantID != "" {
		if !isGlobalAdmin {
			allowed := false
			for _, tid := range userTenantIDs {
				if tid == requestedTenantID {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "Forbidden: not authorized for this tenant", http.StatusForbidden)
				return
			}
		}
		tenantID = requestedTenantID
	}

	if tenantID == "" {
		http.Error(w, "Unauthorized: missing tenant context", http.StatusUnauthorized)
		return
	}

	// startDateStr/endDateStr/datasourceId are interpolated directly into a raw
	// SQL string below (DataFusion has no parameterized query support), so they
	// must be strictly validated first to prevent SQL injection.
	if startDateStr != "" {
		if _, err := time.Parse(time.RFC3339, startDateStr); err != nil {
			http.Error(w, "Invalid startDate: must be RFC3339", http.StatusBadRequest)
			return
		}
	}
	if endDateStr != "" {
		if _, err := time.Parse(time.RFC3339, endDateStr); err != nil {
			http.Error(w, "Invalid endDate: must be RFC3339", http.StatusBadRequest)
			return
		}
	}
	if datasourceId != "" {
		if _, err := uuid.Parse(datasourceId); err != nil {
			http.Error(w, "Invalid datasourceId: must be a UUID", http.StatusBadRequest)
			return
		}
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		http.Error(w, "Invalid tenant context", http.StatusBadRequest)
		return
	}

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
	if datasourceId != "" {
		whereClause += fmt.Sprintf(" AND datasource_id = '%s'", datasourceId)
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
		"SELECT id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details FROM audit_logs%s ORDER BY %s %s LIMIT %d OFFSET %d",
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
	for rowIdx := 0; rowIdx < resp.RowCount; rowIdx++ {
		colCount := len(resp.Records)
		if colCount < 9 {
			continue
		}
		e := AuditLogEntry{
			ID:           fmt.Sprintf("%v", resp.Records[0][rowIdx]),
			TenantID:     fmt.Sprintf("%v", resp.Records[1][rowIdx]),
			UserName:     fmt.Sprintf("%v", resp.Records[3][rowIdx]),
			UserEmail:    fmt.Sprintf("%v", resp.Records[4][rowIdx]),
			Action:       fmt.Sprintf("%v", resp.Records[5][rowIdx]),
			Resource:     fmt.Sprintf("%v", resp.Records[6][rowIdx]),
			ResourceType: fmt.Sprintf("%v", resp.Records[7][rowIdx]),
		}
		if tsStr := fmt.Sprintf("%v", resp.Records[2][rowIdx]); tsStr != "" {
			if parsedTs, err := time.Parse(time.RFC3339, tsStr); err == nil {
				e.Timestamp = parsedTs
			}
		}
		if detailsStr := fmt.Sprintf("%v", resp.Records[8][rowIdx]); detailsStr != "" {
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
