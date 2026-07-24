package temporal

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	workflowservice "go.temporal.io/api/workflowservice/v1"
)

var TemporalUp = promauto.NewGauge(prometheus.GaugeOpts{Name: "temporal_up", Help: "Temporal service reachable (1 = up)"})

// HealthHandler returns an HTTP handler that performs a lightweight admin RPC
// (ListNamespaces) to verify the Temporal server is reachable.
func HealthHandler(admin *AdminClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if admin == nil || admin.svc == nil {
			TemporalUp.Set(0)
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{"status": "down", "error": "admin client not available"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// ListNamespaces is a lightweight RPC that verifies admin API responsiveness.
		req := &workflowservice.ListNamespacesRequest{}
		if _, err := admin.svc.ListNamespaces(ctx, req); err != nil {
			TemporalUp.Set(0)
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{"status": "down", "error": err.Error()})
			return
		}

		TemporalUp.Set(1)
		json.NewEncoder(w).Encode(map[string]any{"status": "up"})
	}
}
