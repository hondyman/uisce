package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/models"
)

type QueryHandler struct {
	service      *analytics.QueryService
	securityDeps SecurityContextDeps
}

func NewQueryHandler(s *analytics.QueryService, deps SecurityContextDeps) *QueryHandler {
	return &QueryHandler{service: s, securityDeps: deps}
}

// HandleExecuteQuery is updated to use the new explorer request/response.
func (h *QueryHandler) HandleExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req models.ExplorerQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// --- 1. AuthN/Z & Security Context ---
	// In a real app, this would come from auth middleware (e.g., JWT claims)
	secCtx, ctx, err := SecurityContextFromRequest(r, req.DatasourceID, req.Region, h.securityDeps)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	req.Region = secCtx.Region
	req.DatasourceID = secCtx.DatasourceID

	// --- 2. Caps & Rate Limiting (Placeholder) ---
	// if err := enforceCaps(req); err != nil {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusUnprocessableEntity)
	// 	json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
	// 	return
	// }
	// if !rateLimiter.Allow(secCtx.UserID) {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusTooManyRequests)
	// 	json.NewEncoder(w).Encode(map[string]interface{}{"error": "rate limit exceeded"})
	// 	return
	// }

	// --- 3 & 4. Compile & Execute ---
	result, err := h.service.ExecuteQuery(ctx, *secCtx, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to execute query", "details": err.Error()})
		return
	}

	// --- 5. Audit Logging ---
	// And update last run stats if it's a saved query
	// The SavedID is a pointer to a string, so we need to check for nil first.
	if req.SavedID != nil && *req.SavedID != "" {
		// This now also computes and stores the diff
		go func() { _ = h.service.LogAndDiffRun(r.Context(), *req.SavedID, req, result) }()
		// This can be removed if LogAndDiffRun handles all stats updates
		go func() {
			_ = h.service.UpdateLastRunStats(r.Context(), *req.SavedID, result.DurationMs, len(result.Rows))
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleCompileQuery handles requests to compile a query without executing it.
func (h *QueryHandler) HandleCompileQuery(w http.ResponseWriter, r *http.Request) {
	var req models.ExplorerQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// --- Security Context ---
	secCtx, ctx, err := SecurityContextFromRequest(r, req.DatasourceID, req.Region, h.securityDeps)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	req.Region = secCtx.Region
	req.DatasourceID = secCtx.DatasourceID

	result, err := h.service.CompileQuery(ctx, *secCtx, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to compile query", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleListHistory retrieves the user's query history.
func (h *QueryHandler) HandleListHistory(w http.ResponseWriter, r *http.Request) {
	// In a real app, user ID would come from auth context.
	userID := "user-123"
	history, err := h.service.ListHistory(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve history", "details": err.Error()})
		return
	}
	if history == nil {
		history = []analytics.SavedQueryResponse{} // Ensure empty array instead of null
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// HandleExportQuery handles requests to export query results to CSV.
func (h *QueryHandler) HandleExportQuery(w http.ResponseWriter, r *http.Request) {
	var req models.ExplorerQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// --- 6. Export Logic ---
	// In a real app, you would check row counts and decide between sync/async.
	// For now, we'll just perform a synchronous export.

	// For exports, we don't want pagination from the request.
	req.Limit = nil
	req.Offset = nil

	secCtx, ctx, err := SecurityContextFromRequest(r, req.DatasourceID, req.Region, h.securityDeps)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	req.Region = secCtx.Region
	req.DatasourceID = secCtx.DatasourceID

	result, err := h.service.ExecuteQuery(ctx, *secCtx, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to execute query for export", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="export.csv"`)

	writer := csv.NewWriter(w)
	var headers []string
	for _, col := range result.Columns {
		headers = append(headers, col.Name)
	}
	_ = writer.Write(headers)
	for _, row := range result.Rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			if val, ok := row[h]; ok {
				record[i] = fmt.Sprint(val)
			}
		}
		_ = writer.Write(record)
	}
	writer.Flush()
}
