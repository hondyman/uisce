package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP requests for MDM source scoring and vendor displacement.
type Handler struct {
	service         *Service
	lastValidBundle atomic.Pointer[OptimalBundleResult]
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
		r.Get("/dimensions", h.HandleGetDimensions)
		r.Get("/shadow-eval", h.HandleGetShadowValidation)
		r.Get("/trends", h.HandleGetTrends)
		r.Get("/health", h.HandleHealthCheck)
		r.Post("/optimize-bundle", h.HandleOptimizeBundle)
		r.Post("/simulate-profiles", h.HandleSimulateProfiles)
		r.Get("/weight-profiles", h.HandleGetWeightProfiles)
		r.Post("/weight-profiles", h.HandleSaveWeightProfile)
		r.Post("/displacement", h.HandleSimulateDisplacement)
		r.Post("/displacement-multi", h.HandleMultiDisplacement)
		r.Post("/sync-mart", h.HandleSyncMart)
		r.Post("/run-ingest", h.HandleRunIngest)
		r.Post("/spend", h.HandleUpdateSpend)
		r.Put("/spend", h.HandleUpdateSpend)
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

	entityDomain := r.URL.Query().Get("entity_domain")

	report, err := h.service.GetScorecardReport(r.Context(), asOf, universeSize, entityDomain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// HandleUpdateSpend updates the annual spend for a vendor overall or per entity domain.
func (h *Handler) HandleUpdateSpend(w http.ResponseWriter, r *http.Request) {
	var req UpdateSpendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.VendorID == "" {
		http.Error(w, "vendor_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateVendorCost(r.Context(), req.VendorID, req.AnnualCost, req.EntityDomain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "SUCCESS",
		"vendor_id":     req.VendorID,
		"annual_cost":   req.AnnualCost,
		"entity_domain": req.EntityDomain,
		"message":       "Vendor annual spend updated successfully",
	})
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

// HandleSyncMart manually triggers a recalculation and flush to StarRocks hot mart.
func (h *Handler) HandleSyncMart(w http.ResponseWriter, r *http.Request) {
	asOfStr := r.URL.Query().Get("as_of")
	var asOf time.Time
	if asOfStr != "" {
		if parsed, err := time.Parse("2006-01-02", asOfStr); err == nil {
			asOf = parsed
		}
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}

	report, err := h.service.SyncMart(r.Context(), asOf, 42000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":          "COMPLETED",
		"as_of":           report.AsOfDate,
		"records_synced":  len(report.SubstitutionMatrix),
		"annual_spend":    report.AnnualSpendTotal,
		"frontier_points": len(report.FrontierPoints),
	})
}

// RunIngestRequest models an ingest and scoring pipeline trigger request.
type RunIngestRequest struct {
	UniverseSize int    `json:"universe_size"`
	PipelineID   string `json:"pipeline_id"`
	AsOf         string `json:"as_of"`
}

// HandleRunIngest executes the multi-vendor ingestion & scoring pipeline on demand.
func (h *Handler) HandleRunIngest(w http.ResponseWriter, r *http.Request) {
	var req RunIngestRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	universeSize := req.UniverseSize
	if universeSize <= 0 {
		universeSize = 42000
	}

	var asOf time.Time
	if req.AsOf != "" {
		if parsed, err := time.Parse("2006-01-02", req.AsOf); err == nil {
			asOf = parsed
		}
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}

	report, err := h.service.SyncMart(r.Context(), asOf, universeSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("ingest failed: %v", err), http.StatusInternalServerError)
		return
	}

	runID := "run-mdm-" + time.Now().Format("20060102150405")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"run_id":           runID,
		"status":           "COMPLETED",
		"as_of":            report.AsOfDate,
		"records_ingested": universeSize,
		"records_scored":   len(report.SubstitutionMatrix),
		"frontier_points":  len(report.FrontierPoints),
		"annual_spend":     report.AnnualSpendTotal,
		"message":          "MDM Multi-Vendor Ingestion, Iceberg export, Staging load, Mastering & Scoring completed.",
	})
}

// HandleGetDimensions returns granular 6-pillar vendor telemetry profiles.
func (h *Handler) HandleGetDimensions(w http.ResponseWriter, r *http.Request) {
	asOfStr := r.URL.Query().Get("as_of")
	var asOf time.Time
	if asOfStr != "" {
		if parsed, err := time.Parse("2006-01-02", asOfStr); err == nil {
			asOf = parsed
		}
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}

	profiles, err := h.service.GetDimensionProfiles(r.Context(), asOf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(profiles)
}

// HandleHealthCheck returns StarRocks mart freshness, feed gap alerts, and solver partial event counts.
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := h.service.GetPipelineHealth(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health)
}

// HandleOptimizeBundle executes the weighted set cover portfolio optimization solver.
func (h *Handler) HandleOptimizeBundle(w http.ResponseWriter, r *http.Request) {
	var req OptimalBundleRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
	defer cancel()

	res, err := h.service.OptimizeBundle(ctx, req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded") {
			// Record alert metric
			h.service.RecordPartialSolver()

			// Check if we have a cached last valid bundle
			if cached := h.lastValidBundle.Load(); cached != nil {
				clone := *cached
				clone.SolverPartial = true
				clone.SolverExecutionMs = 50.0
				clone.SolverStrategy = "CACHED_LAST_VALID"
				res = &clone
			} else {
				// Gracefully return partial baseline with solver_partial flag
				res = &OptimalBundleResult{
					SolverPartial:       true,
					SolverExecutionMs:   50.0,
					SolverStrategy:      "TIMEOUT_FALLBACK_BASELINE",
					SelectedVendors:     []string{"BBG", "RFT"},
					SelectedVendorNames: []string{"Bloomberg", "Refinitiv (LSEG)"},
					TotalAnnualCost:     3320000,
				}
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if res != nil && !res.SolverPartial {
		h.lastValidBundle.Store(res)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleMultiDisplacement simulates multi-vendor exit and replacement scenarios.
func (h *Handler) HandleMultiDisplacement(w http.ResponseWriter, r *http.Request) {
	var req MultiVendorDisplacementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.DroppedVendorIDs) == 0 {
		http.Error(w, "at least one dropped vendor ID is required", http.StatusBadRequest)
		return
	}

	res, err := h.service.EvaluateMultiDisplacement(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

// HandleGetWeightProfiles lists registered weight governance profiles.
func (h *Handler) HandleGetWeightProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.GetWeightProfiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(profiles)
}

// HandleSaveWeightProfile validates and persists a weight profile.
func (h *Handler) HandleSaveWeightProfile(w http.ResponseWriter, r *http.Request) {
	var profile WeightProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if profile.ProfileName == "" {
		http.Error(w, "profile_name is required", http.StatusBadRequest)
		return
	}

	saved, err := h.service.SaveWeightProfile(r.Context(), profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(saved)
}

// HandleGetShadowValidation returns the side-by-side legacy vs 6-pillar comparator report and recent audit logs.
func (h *Handler) HandleGetShadowValidation(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
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

	report, err := h.service.GetShadowValidationReport(r.Context(), asOf, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// HandleGetTrends handles GET /api/mdm/scoring/trends with watermark-based query routing.
func (h *Handler) HandleGetTrends(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	dimStr := r.URL.Query().Get("dimension")
	if dimStr == "" {
		dimStr = "COMPOSITE"
	}

	dimension, err := ValidateDimension(dimStr)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	var vendorIDs []string
	if vParam := r.URL.Query().Get("vendors"); vParam != "" {
		for _, part := range strings.Split(vParam, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				vendorIDs = append(vendorIDs, trimmed)
			}
		}
	} else if vParam := r.URL.Query().Get("vendor_ids"); vParam != "" {
		for _, part := range strings.Split(vParam, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				vendorIDs = append(vendorIDs, trimmed)
			}
		}
	}

	var dateFrom, dateTo time.Time
	if fromStr := r.URL.Query().Get("date_from"); fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			dateFrom = parsed
		}
	}
	if toStr := r.URL.Query().Get("date_to"); toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			dateTo = parsed
		}
	}

	entityDomain := r.URL.Query().Get("entity_domain")

	req := TrendQueryRequest{
		TenantID:     tenantID,
		VendorIDs:    vendorIDs,
		Dimension:    dimension,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		EntityDomain: entityDomain,
	}

	report, err := h.service.QueryHistoricalTrends(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

// HandleSimulateProfiles performs A/B sensitivity comparison between two weight profiles.
func (h *Handler) HandleSimulateProfiles(w http.ResponseWriter, r *http.Request) {
	tenantID := "default"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	} else if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
		tenantID = tid
	}

	var req ProfileSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	res, err := h.service.SimulateProfiles(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must sum to") || strings.Contains(err.Error(), "must specify") {
			status = http.StatusBadRequest
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}




