package semanticmatch

import (
	"encoding/json"
	"net/http"
)

// SuggestHandler: POST /api/v1/semantic-match/suggest
// Body: {"database":"prod","schema":"public","table":"ts_order_broker",
//        "column":"uw_exp","data_type":"NUMERIC(18,9) NULL",
//        "sample_values":["1250.00"]}
func SuggestHandler(p *Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ColumnMeta
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Table == "" || req.Column == "" {
			http.Error(w, "table and column are required", http.StatusBadRequest)
			return
		}
		cands, outcome, err := p.Suggest(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"column":     req,
			"candidates": cands,
			"decision":   outcome,
		})
	}
}
