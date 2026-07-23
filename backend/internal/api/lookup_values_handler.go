package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type DynamicLookupValue struct {
	ID          string                 `json:"id"`
	LookupType  string                 `json:"lookup_type"`
	Value       string                 `json:"value"`
	Label       string                 `json:"label"`
	Description *string                `json:"description,omitempty"`
	SortOrder   *int                   `json:"sort_order,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsActive    bool                   `json:"is_active"`
}

// HandleGetLookupValues returns lookup values for a given type
func HandleGetLookupValues(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lookupType := r.URL.Query().Get("type")
		if lookupType == "" {
			http.Error(w, "lookup type is required", http.StatusBadRequest)
			return
		}

		query := `
			SELECT id, lookup_type, value, label, description, sort_order, metadata, is_active
			FROM lookup_values
			WHERE lookup_type = $1 AND is_active = true
			ORDER BY sort_order ASC, label ASC
		`

		rows, err := db.Query(query, lookupType)
		if err != nil {
			http.Error(w, "Failed to fetch lookup values", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var values []DynamicLookupValue
		for rows.Next() {
			var lv DynamicLookupValue
			var metadataJSON []byte

			err := rows.Scan(
				&lv.ID,
				&lv.LookupType,
				&lv.Value,
				&lv.Label,
				&lv.Description,
				&lv.SortOrder,
				&metadataJSON,
				&lv.IsActive,
			)
			if err != nil {
				http.Error(w, "Failed to scan lookup value", http.StatusInternalServerError)
				return
			}

			// Parse metadata JSON
			if len(metadataJSON) > 0 {
				json.Unmarshal(metadataJSON, &lv.Metadata)
			}

			values = append(values, lv)
		}

		if values == nil {
			values = []DynamicLookupValue{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(values)
	}
}
