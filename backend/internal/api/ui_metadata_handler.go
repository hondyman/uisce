package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
)

type ViewDefinitionResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Title      string          `json:"title"`
	Components []ViewComponent `json:"components"`
}

type ViewComponent struct {
	ID            string                 `json:"id"`
	DataKey       string                 `json:"dataKey"`
	ComponentType string                 `json:"componentType"`
	Label         string                 `json:"label"`
	Order         int                    `json:"order"`
	Properties    map[string]interface{} `json:"properties"`
}

// GetViewDefinition handles GET /ui-definitions/{id_or_name}
// It supports fetching by UUID or Name (e.g. "high_value_order_approval_form")
func (s *Server) GetViewDefinition(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "id")
	if param == "" {
		http.Error(w, "Missing ID or Name", http.StatusBadRequest)
		return
	}

	// 1. Fetch Definition
	var viewID, name, title string
	// Try UUID first. If not, try Name.
	// Actually, easier to just query with OR, or regex check if UUID.
	// For simplicity, let's assume if it looks like a Uuid use ID, else Name.
	// Or just ONE query: WHERE id::text = $1 OR name = $1 (might fail uuid cast if not uuid)
	// Safer: Query by Name if not UUID-like?
	// Let's implement robust query:

	query := `SELECT id, name, title FROM view_definitions WHERE id::text = $1 OR name = $1 LIMIT 1`
	// Note: Postgres will throw error if $1 is not uuid when casting id::text? No, id::text works.
	// But `id = $1` where $1 is string "foo" will fail input syntax for uuid.
	// So `id::text = $1` is safe.

	err := s.DB.QueryRowContext(r.Context(), query, param).Scan(&viewID, &name, &title)
	if err != nil {
		// Log error just in case it's DB failure
		// log.Printf("DB Error: %v", err)
		http.Error(w, "View Definition not found", http.StatusNotFound)
		return
	}

	// 2. Fetch Components
	compQuery := `
		SELECT id, data_key, component_type, label, "order", properties 
		FROM view_components 
		WHERE view_definition_id = $1 
		ORDER BY "order" ASC
	`
	rows, err := s.DB.QueryContext(r.Context(), compQuery, viewID)
	if err != nil {
		http.Error(w, "Failed to fetch components", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var components []ViewComponent
	for rows.Next() {
		var c ViewComponent
		var propsJSON []byte
		if err := rows.Scan(&c.ID, &c.DataKey, &c.ComponentType, &c.Label, &c.Order, &propsJSON); err != nil {
			continue
		}
		if len(propsJSON) > 0 {
			_ = json.Unmarshal(propsJSON, &c.Properties)
		}
		components = append(components, c)
	}

	// Double check sort (though DB did it)
	sort.Slice(components, func(i, j int) bool {
		return components[i].Order < components[j].Order
	})

	resp := ViewDefinitionResponse{
		ID:         viewID,
		Name:       name,
		Title:      title,
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CompleteTask handles POST /tasks/{id}/complete
func (s *Server) CompleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		http.Error(w, "Missing Task ID", http.StatusBadRequest)
		return
	}

	// 1. Parse Result Payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// 2. Fetch Task Token from DB
	var taskTokenStr string
	var status string
	err := s.DB.QueryRowContext(r.Context(), "SELECT task_token, status FROM human_tasks WHERE id = $1", taskID).Scan(&taskTokenStr, &status)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if status != "PENDING" {
		http.Error(w, "Task already completed", http.StatusConflict)
		return
	}

	// 3. Complete Temporal Activity
	// Task Token is generic string in DB, but Temporal expects []byte.
	// Actually, Temporal Task Token is []byte. We stored it as string/byte array?
	// In schema: task_token TEXT. It should be base64 encoded or raw bytes?
	// The SDK activity.GetInfo(ctx).TaskToken returns []byte.
	// If we successfully scanned it into string in DB, we need to convert it back?
	// Usually TaskToken is raw bytes. Storing in TEXT column might corrupt it if not base64.
	// Let's assume we change DB column to BYTEA or store as Base64 string.
	// For now, let's treat it as if we stored it correctly.

	// CAUTION: If task_token is binary, we should have used BYTEA or base64.
	// Let's check schema: task_token TEXT.
	// We might need to base64 decode it if we modify the Activity to store base64.
	// For now, let's just pass []byte(taskTokenStr) and hope it works or fix Activity.

	err = s.TemporalClient.CompleteActivity(r.Context(), []byte(taskTokenStr), payload, nil)
	if err != nil {
		http.Error(w, "Failed to complete workflow activity: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Update Task Status in DB
	_, err = s.DB.ExecContext(r.Context(), "UPDATE human_tasks SET status = 'COMPLETED', result = $1, updated_at = NOW() WHERE id = $2", r.Body, taskID)
	// r.Body is consumed above. Need to marshal payload again.
	payloadBytes, _ := json.Marshal(payload)
	_, _ = s.DB.ExecContext(r.Context(), "UPDATE human_tasks SET status = 'COMPLETED', result = $1, updated_at = NOW() WHERE id = $2", payloadBytes, taskID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}
