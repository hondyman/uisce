// Package builder provides API routes for the Uisce visual rule builder
package builder

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers the Uisce builder API routes
func RegisterRoutes(r chi.Router, starrocksDB *sql.DB) {
	r.Post("/rules/generate-cue", handleGenerateCUE())
	r.Post("/rules/simulate", handleSimulate(starrocksDB))
}

// handleGenerateCUE handles POST /api/rules/generate-cue
func handleGenerateCUE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule UIRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}

		result, err := GenerateCUE(rule)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate CUE", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// handleSimulate handles POST /api/rules/simulate
func handleSimulate(starrocksDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Rule      UIRule `json:"rule"`
			TimeRange string `json:"time_range"`
			TableName string `json:"table_name,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}

		if req.TimeRange == "" {
			req.TimeRange = "24h"
		}

		tableName := req.TableName
		if tableName == "" {
			tableName = "historical_trades" // Default table
		}

		simulator := NewSimulator(starrocksDB, tableName)
		report, err := simulator.SimulateRuleImpact(r.Context(), req.Rule, req.TimeRange)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Simulation failed", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, status int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   message,
		"details": details,
	})
}
