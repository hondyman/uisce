package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/autonomous_runtime/capacity"
	errorclustering "github.com/hondyman/semlayer/backend/internal/autonomous_runtime/error_clustering"
	"github.com/hondyman/semlayer/backend/internal/autonomous_runtime/incidents"
	preagggeneration "github.com/hondyman/semlayer/backend/internal/autonomous_runtime/preagg_generation"
	sloprevention "github.com/hondyman/semlayer/backend/internal/autonomous_runtime/slo_prevention"
)

type AutonomousRuntimeHandler struct {
	sloPredictor     *sloprevention.SLOPredictor
	capacityPlanner  *capacity.CapacityPlanner
	incidentReporter *incidents.IncidentReporter
	errorClusterer   *errorclustering.ErrorClusterer
	preAggGenerator  *preagggeneration.PreAggGenerator
}

func NewAutonomousRuntimeHandler(
	slo *sloprevention.SLOPredictor,
	cap *capacity.CapacityPlanner,
	inc *incidents.IncidentReporter,
	err *errorclustering.ErrorClusterer,
	pre *preagggeneration.PreAggGenerator,
) *AutonomousRuntimeHandler {
	return &AutonomousRuntimeHandler{
		sloPredictor:     slo,
		capacityPlanner:  cap,
		incidentReporter: inc,
		errorClusterer:   err,
		preAggGenerator:  pre,
	}
}

func (h *AutonomousRuntimeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// SLO Prevention
	r.Get("/slo-predictions", h.GetSLOPredictions)
	r.Post("/prevent-violation", h.PreventViolation)

	// Capacity Planning
	r.Get("/capacity-forecast/{tenantId}", h.GetCapacityForecast)

	// Incident Reporting
	r.Post("/incidents/generate/{incidentId}", h.GenerateIncidentReport)

	// Error Clustering
	r.Get("/error-clusters", h.GetErrorClusters)
	r.Post("/error-clusters/{clusterId}/heal", h.AutoHealCluster)

	// Pre-Agg Generation
	r.Get("/preagg/suggestions", h.GetPreAggSuggestions)
	r.Post("/preagg/generate-changeset", h.GeneratePreAggChangeSet)

	return r
}

func (h *AutonomousRuntimeHandler) GetSLOPredictions(w http.ResponseWriter, r *http.Request) {
	predictions, _ := h.sloPredictor.PredictViolations(r.Context())
	json.NewEncoder(w).Encode(predictions)
}

func (h *AutonomousRuntimeHandler) PreventViolation(w http.ResponseWriter, r *http.Request) {
	var prediction sloprevention.ViolationPrediction
	if err := json.NewDecoder(r.Body).Decode(&prediction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actions, _ := h.sloPredictor.PreventViolation(r.Context(), &prediction)
	json.NewEncoder(w).Encode(actions)
}

func (h *AutonomousRuntimeHandler) GetCapacityForecast(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	forecasts, _ := h.capacityPlanner.Forecast(r.Context(), tenantID)
	recommendations, _ := h.capacityPlanner.Recommend(r.Context(), forecasts)

	response := map[string]interface{}{
		"forecasts":       forecasts,
		"recommendations": recommendations,
	}
	json.NewEncoder(w).Encode(response)
}

func (h *AutonomousRuntimeHandler) GenerateIncidentReport(w http.ResponseWriter, r *http.Request) {
	incidentID, _ := uuid.Parse(chi.URLParam(r, "incidentId"))
	report, _ := h.incidentReporter.Generate(r.Context(), incidentID)
	json.NewEncoder(w).Encode(report)
}

func (h *AutonomousRuntimeHandler) GetErrorClusters(w http.ResponseWriter, r *http.Request) {
	clusters, _ := h.errorClusterer.ClusterErrors(r.Context())
	json.NewEncoder(w).Encode(clusters)
}

func (h *AutonomousRuntimeHandler) AutoHealCluster(w http.ResponseWriter, r *http.Request) {
	clusterID, _ := uuid.Parse(chi.URLParam(r, "clusterId"))

	// Find cluster and attempt healing
	clusters, _ := h.errorClusterer.ClusterErrors(r.Context())
	for _, cluster := range clusters {
		if cluster.ClusterID == clusterID {
			err := h.errorClusterer.AutoHeal(r.Context(), &cluster)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	http.Error(w, "cluster not found", http.StatusNotFound)
}

func (h *AutonomousRuntimeHandler) GetPreAggSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, _ := h.preAggGenerator.Suggest(r.Context())
	json.NewEncoder(w).Encode(suggestions)
}

func (h *AutonomousRuntimeHandler) GeneratePreAggChangeSet(w http.ResponseWriter, r *http.Request) {
	var suggestion preagggeneration.PreAggSuggestion
	if err := json.NewDecoder(r.Body).Decode(&suggestion); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	changeset, _ := h.preAggGenerator.GenerateChangeSet(r.Context(), &suggestion)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(changeset))
}
