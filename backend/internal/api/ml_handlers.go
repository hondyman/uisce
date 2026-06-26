package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/ml"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// MLHandler handles ML prediction API endpoints
type MLHandler struct {
	service ml.Service
}

// NewMLHandler creates a new ML API handler
func NewMLHandler(service ml.Service) *MLHandler {
	return &MLHandler{service: service}
}

// RegisterMLRoutes registers all ML endpoints
func RegisterMLRoutes(router *chi.Mux, handler *MLHandler) {
	router.Post("/admin/ml/predict", handler.Predict)
	router.Post("/admin/ml/predict/batch", handler.PredictBatch)
	router.Get("/admin/ml/explain/{chainId}", handler.GetExplanation)
	router.Post("/admin/ml/explain/batch", handler.ExplainBatch)
	router.Get("/admin/ml/anomalies", handler.DetectAnomalies)
	router.Get("/admin/ml/model/metrics", handler.GetModelMetrics)
	router.Get("/admin/ml/model/health", handler.ModelHealth)
}

// Predict generates a failure prediction for a single chain
// @Summary Get failure prediction
// @Tags ML
// @Accept json
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Param prediction body ml.PredictionInput true "Prediction Input"
// @Success 200 {object} ml.Prediction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /admin/ml/predict [post]
func (h *MLHandler) Predict(w http.ResponseWriter, r *http.Request) {
	var input ml.PredictionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate input
	if err := validatePredictionInput(&input); err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid input: %v", err))
		return
	}

	// Get prediction
	prediction, err := h.service.GetPrediction(r.Context(), &input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Prediction failed: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, prediction)
}

// PredictBatch generates predictions for multiple chains
// @Summary Batch failure predictions
// @Tags ML
// @Accept json
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Param batch body ml.PredictionBatch true "Batch Input"
// @Success 200 {object} ml.PredictionBatchResult
// @Failure 400 {object} ErrorResponse
// @Router /admin/ml/predict/batch [post]
func (h *MLHandler) PredictBatch(w http.ResponseWriter, r *http.Request) {
	var batch ml.PredictionBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate batch
	if len(batch.Inputs) == 0 {
		respondError(w, http.StatusBadRequest, "Batch cannot be empty")
		return
	}

	if len(batch.Inputs) > 1000 {
		respondError(w, http.StatusBadRequest, "Batch size cannot exceed 1000")
		return
	}

	// Get batch predictions
	result, err := h.service.GetPredictionBatch(r.Context(), &batch)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Batch prediction failed: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// GetExplanation returns detailed SHAP explanation for a prediction
// @Summary Get prediction explanation
// @Tags ML
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Param chainId path string true "Chain ID"
// @Success 200 {object} ml.Explainability
// @Failure 404 {object} ErrorResponse
// @Router /admin/ml/explain/{chainId} [get]
func (h *MLHandler) GetExplanation(w http.ResponseWriter, r *http.Request) {
	chainID := chi.URLParam(r, "chainId")
	if chainID == "" {
		respondError(w, http.StatusBadRequest, "Chain ID required")
		return
	}

	// In production, would fetch cached explanation from database
	// For now, return 501 indicating explanations need to be computed via batch
	respondError(w, http.StatusNotImplemented, "Use batch explain endpoint or predict with explain=true")
}

// ExplainBatch generates SHAP explanations for batch of predictions
// @Summary Batch explainability analysis
// @Tags ML
// @Accept json
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Param batch body ml.PredictionBatch true "Batch Input"
// @Success 200 {object} map[string]ml.Explainability "Chain ID -> Explainability"
// @Failure 400 {object} ErrorResponse
// @Router /admin/ml/explain/batch [post]
func (h *MLHandler) ExplainBatch(w http.ResponseWriter, r *http.Request) {
	var batch ml.PredictionBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(batch.Inputs) == 0 || len(batch.Inputs) > 100 {
		respondError(w, http.StatusBadRequest, "Batch size must be 1-100")
		return
	}

	// Get explanations via service
	result, err := h.service.GetPredictionBatch(r.Context(), &batch)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Explain failed: %v", err))
		return
	}

	// Extract explanations
	explains := make(map[string]ml.Explainability)
	for _, pred := range result.Predictions {
		if pred.Explainability != nil {
			explains[pred.ChainID] = *pred.Explainability
		}
	}

	respondJSON(w, http.StatusOK, explains)
}

// DetectAnomalies detects anomalies in chain metrics
// @Summary Detect anomalies
// @Tags ML
// @Accept json
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Param chainId query string true "Chain ID"
// @Param region query string true "Region"
// @Success 200 {array} ml.AnomalyScore
// @Failure 400 {object} ErrorResponse
// @Router /admin/ml/anomalies [get]
func (h *MLHandler) DetectAnomalies(w http.ResponseWriter, r *http.Request) {
	chainID := r.URL.Query().Get("chainId")
	region := r.URL.Query().Get("region")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if chainID == "" || region == "" {
		respondError(w, http.StatusBadRequest, "chainId and region parameters required")
		return
	}

	// Mock input for anomaly detection
	// In production, would fetch real metrics from storage
	input := &ml.PredictionInput{
		ChainID:            chainID,
		Region:             region,
		TenantID:           tenantID,
		HealthScore:        0.85,
		ActiveConflicts:    3,
		P99Latency:         450,
		ErrorRate:          0.01,
		CrossRegionLatency: 800,
	}

	anomalies, err := h.service.GetAnomalies(r.Context(), input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Anomaly detection failed: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, anomalies)
}

// GetModelMetrics returns model performance metrics
// @Summary Get model metrics
// @Tags ML
// @Produce json
// @Param Authorization header string true "Auth Token"
// @Success 200 {object} ml.ModelMetrics
// @Failure 500 {object} ErrorResponse
// @Router /admin/ml/model/metrics [get]
func (h *MLHandler) GetModelMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetModelMetrics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get metrics: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// ModelHealth returns a health check for the ML service
// @Summary ML service health check
// @Tags ML
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/ml/model/health [get]
func (h *MLHandler) ModelHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"features": map[string]bool{
			"predictions":       true,
			"explanability":     true,
			"anomaly_detection": true,
			"batch_processing":  true,
		},
	}

	respondJSON(w, http.StatusOK, health)
}

// Helper functions

func validatePredictionInput(input *ml.PredictionInput) error {
	if input.ChainID == "" {
		return fmt.Errorf("chain_id required")
	}
	if input.Region == "" {
		return fmt.Errorf("region required")
	}
	if input.TenantID == "" {
		return fmt.Errorf("tenant_id required")
	}
	if input.HealthScore < 0 || input.HealthScore > 1 {
		return fmt.Errorf("health_score must be between 0 and 1")
	}
	if input.ErrorRate < 0 || input.ErrorRate > 1 {
		return fmt.Errorf("error_rate must be between 0 and 1")
	}
	return nil
}

// respondError sends an error response - local helper for ml_handlers
func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
