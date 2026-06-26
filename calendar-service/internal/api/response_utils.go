package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Cannot write error to response if header already written, just log or ignore
		// In a real app we might log this
	}
}
