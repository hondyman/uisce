package api

import (
	"encoding/json"
	"net/http"
)

type ForecastAPIRequest struct {
	Rows         []map[string]interface{} `json:"rows"`
	DimensionCol string                   `json:"dimensionCol"`
	TimeCol      string                   `json:"timeCol"`
	MeasureCol   string                   `json:"measureCol"`
	PeriodsAhead int                      `json:"periodsAhead"`
}

type ForecastAPIHandler struct {
	Service *AdvancedProjectionService
}

func NewForecastAPIHandler(service *AdvancedProjectionService) *ForecastAPIHandler {
	return &ForecastAPIHandler{Service: service}
}

// HandleForecast endpoint wrapper
func (h *ForecastAPIHandler) HandleForecast() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ForecastAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.PeriodsAhead <= 0 {
			req.PeriodsAhead = 6 // Default to 6 periods
		}

		results, err := h.Service.ForecastByDimension(
			req.Rows,
			req.DimensionCol,
			req.TimeCol,
			req.MeasureCol,
			req.PeriodsAhead,
		)
		if err != nil {
			http.Error(w, "Projection calculation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"results": results,
		})
	}
}
