package scoring

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for MDM source scoring and vendor displacement.
type Handler struct {
	service *Service
}

// NewHandler creates a new MDM scoring HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the scoring API endpoints on a chi.Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/mdm/scoring", func(r chi.Router) {
		r.Get("/scorecard", h.HandleGetScorecard)
		r.Get("/tolerances", h.HandleGetTolerances)
		r.Post("/displacement", h.HandleSimulateDisplacement)
	})
}

// HandleGetScorecard returns the complete vendor quality and displacement scorecard report.
func (h *Handler) HandleGetScorecard(w http.ResponseWriter, r *http.Request) {
	asOfStr := r.URL.Query().Get("as_of")
	var asOf time.Time
	if asOfStr != "" {
		parsed, err := time.Parse("2006-01-02", asOfStr)
		if err == nil {
			asOf = parsed
		}
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}

	universeSize := 42000
	if uStr := r.URL.Query().Get("universe_size"); uStr != "" {
		if parsed, err := strconv.Atoi(uStr); err == nil && parsed > 0 {
			universeSize = parsed
		}
	}

	report, err := h.service.GetScorecardReport(r.Context(), asOf, universeSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// HandleGetTolerances returns all configured attribute tolerances.
func (h *Handler) HandleGetTolerances(w http.ResponseWriter, r *http.Request) {
	tolerances, err := h.service.repo.GetAttributeTolerances(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tolerances)
}

// SimulateDisplacementRequest models a request to simulate vendor removal.
type SimulateDisplacementRequest struct {
	DroppedVendorID   string   `json:"dropped_vendor_id"`
	DroppedVendorName string   `json:"dropped_vendor_name"`
	AnnualCost        float64  `json:"annual_cost"`
	VendorHierarchy   []string `json:"vendor_hierarchy"`
}

// HandleSimulateDisplacement runs a custom vendor exit simulation on demand.
func (h *Handler) HandleSimulateDisplacement(w http.ResponseWriter, r *http.Request) {
	var req SimulateDisplacementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DroppedVendorID == "" {
		http.Error(w, "dropped_vendor_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tolerances, err := h.service.repo.GetAttributeTolerances(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	candidates, err := h.service.repo.GetVendorCandidates(ctx, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	goldenRecords, err := h.service.repo.GetGoldenRecords(ctx, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hierarchy := req.VendorHierarchy
	if len(hierarchy) == 0 {
		hierarchy = []string{"BBG", "RFT", "FDS", "ICE", "SPG"}
	}

	result := h.service.evaluator.SimulateVendorDisplacement(
		req.DroppedVendorID,
		req.DroppedVendorName,
		req.AnnualCost,
		tolerances,
		hierarchy,
		candidates,
		goldenRecords,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
