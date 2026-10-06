package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// retiredAggregateDemoBody is the stable 410 payload for the dual-mode
// Aggregate Designer demo endpoints (CUBE-0.5). Cubes (/api/cubes + Build →
// Cubes) are the successor authoring surface.
var retiredAggregateDemoBody = map[string]any{
	"error":     "Gone",
	"message":   "The dual-mode Aggregate Designer demo is retired. Author aggregation contracts via Build → Cubes (/api/cubes).",
	"successor": "/api/cubes",
	"ui":        "/build/cubes",
	"adr":       "docs/adr/0002-cubes-supersede-aggregate-designer.md",
}

func writeRetiredAggregateDemoGone(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", `</api/cubes>; rel="successor-version", </build/cubes>; rel="alternate"`)
	w.WriteHeader(http.StatusGone)
	_ = json.NewEncoder(w).Encode(retiredAggregateDemoBody)
}

// HandleRetiredAggregateDemoGone responds 410 Gone for former Aggregate
// Designer demo APIs.
func HandleRetiredAggregateDemoGone(w http.ResponseWriter, _ *http.Request) {
	writeRetiredAggregateDemoGone(w)
}

// RegisterRetiredAggregateDemoRoutes mounts 410 Gone handlers for the demo
// Aggregate Designer endpoints under an /api router.
func RegisterRetiredAggregateDemoRoutes(r chi.Router) {
	r.Route("/analytics", func(r chi.Router) {
		// All methods: the demo never had a real live contract; any call is gone.
		r.Handle("/aggregates", http.HandlerFunc(HandleRetiredAggregateDemoGone))
		r.Handle("/preview", http.HandlerFunc(HandleRetiredAggregateDemoGone))
	})
}
