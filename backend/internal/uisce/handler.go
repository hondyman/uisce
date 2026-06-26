package uisce

import (
	"encoding/json"
	"net/http"
)

// DebugRequest is the payload for the debug endpoint
type DebugRequest struct {
	TradeData map[string]interface{} `json:"tradeData"`
}

// HandleDebug is the HTTP handler for /api/uisce/debug
// It runs the pipeline in trace mode and returns the debug result
func HandleDebug(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req DebugRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Run the debug trace
		result := engine.RunDebug(r.Context(), req.TradeData)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
