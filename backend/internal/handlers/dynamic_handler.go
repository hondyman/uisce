package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/dynamic"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/backend/models"
)

// DynamicQueryHandler handles dynamic query requests
type DynamicQueryHandler struct {
	dynamicEngine *dynamic.DynamicQueryEngine
	templateMgr   *query.QueryTemplateManager
}

// NewDynamicQueryHandler creates a new dynamic query handler
func NewDynamicQueryHandler(dynamicEngine *dynamic.DynamicQueryEngine, templateMgr *query.QueryTemplateManager) *DynamicQueryHandler {
	return &DynamicQueryHandler{
		dynamicEngine: dynamicEngine,
		templateMgr:   templateMgr,
	}
}

// RegisterRoutes registers the routes for DynamicQueryHandler.
func (dqh *DynamicQueryHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/dynamic", func(r chi.Router) {
		r.Post("/query", dqh.HandleDynamicQuery)
		r.Post("/suggest-measures", dqh.HandleDynamicMeasureSuggestion)
		r.Post("/validate-parameters", dqh.HandleParameterValidation)
		r.Post("/generate-cube-config", dqh.HandleCubeConfigGeneration)
	})
}

// HandleDynamicQuery processes dynamic query requests
func (dqh *DynamicQueryHandler) HandleDynamicQuery(w http.ResponseWriter, r *http.Request) {
	var req DynamicQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Convert to internal format
	dynamicReq := &dynamic.DynamicQueryRequest{
		BaseQuery: &models.Query{
			Metrics:    req.Metrics,
			Dimensions: req.Dimensions,
			Filters:    convertFilters(req.Filters),
			TableName:  req.TableName,
		},
		Parameters:      req.Parameters,
		DynamicMeasures: req.DynamicMeasures,
		TimeRange:       req.TimeRange,
		Context:         req.Context,
	}

	// Resolve parameters
	ctx := r.Context()
	resolved, err := dqh.dynamicEngine.ResolveParameters(ctx, dynamicReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Parameter resolution failed",
			"details": err.Error(),
		})
		return
	}

	// Generate SQL
	sql, args := resolved.BuildSQL()

	// Execute query (you would integrate with your existing DB layer)
	result := map[string]interface{}{
		"query_id":         generateQueryID(),
		"sql":              sql,
		"parameters":       args,
		"resolved_params":  resolved.Parameters,
		"dynamic_measures": resolved.Metrics,
		"execution_time":   "pending", // Would be populated after execution
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleDynamicMeasureSuggestion suggests dynamic measures based on context
func (dqh *DynamicQueryHandler) HandleDynamicMeasureSuggestion(w http.ResponseWriter, r *http.Request) {
	var req MeasureSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Analyze query context and suggest measures
	suggestions := dqh.suggestDynamicMeasures(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
		"context":     req.Context,
	})
}

// HandleParameterValidation validates dynamic parameters
func (dqh *DynamicQueryHandler) HandleParameterValidation(w http.ResponseWriter, r *http.Request) {
	var req ParameterValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate parameters
	validationResult := dqh.validateParameters(req.Parameters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validationResult)
}

// HandleCubeConfigGeneration generates enhanced Cube.js configuration
func (dqh *DynamicQueryHandler) HandleCubeConfigGeneration(w http.ResponseWriter, r *http.Request) {
	var req CubeConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Generate Cube.js configuration
	cubeEnhancer := dynamic.NewCubeDynamicEnhancer(nil) // Would use your existing cube
	config, err := cubeEnhancer.GenerateCubeJSConfig(req.Parameters, req.DynamicMeasures)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Configuration generation failed",
			"details": err.Error(),
		})
		return
	}

	// Generate parameter schema
	schema, err := cubeEnhancer.GenerateParameterSchema(req.Parameters)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Schema generation failed",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cube_config":      config,
		"parameter_schema": schema,
		"generated_at":     "now",
	})
}

// Helper functions

func convertFilters(filters []Filter) []models.Filter {
	result := make([]models.Filter, len(filters))
	for i, f := range filters {
		result[i] = models.Filter{
			Field:  f.Field,
			Op:     f.Operator,
			Values: f.Values,
		}
	}
	return result
}

func (dqh *DynamicQueryHandler) suggestDynamicMeasures(req MeasureSuggestionRequest) []dynamic.DynamicMeasure {
	suggestions := []dynamic.DynamicMeasure{}

	// Revenue per user
	if dqh.hasMetrics(req.Metrics, "revenue", "users") {
		suggestions = append(suggestions, dynamic.DynamicMeasure{
			Name: "Revenue per User",
			Type: "ratio",
			SQL:  "SUM(revenue) / COUNT(DISTINCT user_id)",
		})
	}

	// Growth rate
	if dqh.hasTimeDimensions(req.Dimensions) {
		suggestions = append(suggestions, dynamic.DynamicMeasure{
			Name: "Period Growth Rate",
			Type: "percentage",
			SQL:  "((current_value - previous_value) / previous_value) * 100",
		})
	}

	// Conversion rate
	if dqh.hasMetrics(req.Metrics, "conversions", "visitors") {
		suggestions = append(suggestions, dynamic.DynamicMeasure{
			Name: "Conversion Rate",
			Type: "percentage",
			SQL:  "(SUM(conversions) / SUM(visitors)) * 100",
		})
	}

	return suggestions
}

func (dqh *DynamicQueryHandler) hasMetrics(metrics []string, required ...string) bool {
	metricSet := make(map[string]bool)
	for _, m := range metrics {
		metricSet[m] = true
	}

	for _, req := range required {
		if !metricSet[req] {
			return false
		}
	}
	return true
}

func (dqh *DynamicQueryHandler) hasTimeDimensions(dimensions []string) bool {
	timeDims := []string{"date", "month", "quarter", "year", "period"}
	for _, dim := range dimensions {
		for _, timeDim := range timeDims {
			if dim == timeDim {
				return true
			}
		}
	}
	return false
}

func (dqh *DynamicQueryHandler) validateParameters(params []dynamic.DynamicParameter) map[string]interface{} {
	result := map[string]interface{}{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
	}

	for _, param := range params {
		// Check required parameters
		if param.Required && param.Value == nil && param.DefaultValue == nil {
			result["valid"] = false
			result["errors"] = append(result["errors"].([]string), "Required parameter "+param.Name+" not provided")
		}

		// Validate options
		if len(param.Options) > 0 && param.Value != nil {
			validOption := false
			for _, option := range param.Options {
				if param.Value == option {
					validOption = true
					break
				}
			}
			if !validOption {
				result["errors"] = append(result["errors"].([]string), "Invalid value for parameter "+param.Name)
			}
		}
	}

	return result
}

func generateQueryID() string {
	return "dq_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Request/Response types

type DynamicQueryRequest struct {
	Metrics         []string                   `json:"metrics"`
	Dimensions      []string                   `json:"dimensions"`
	Filters         []Filter                   `json:"filters"`
	TableName       string                     `json:"table_name"`
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
	TimeRange       *query.TimeRange           `json:"time_range,omitempty"`
	Context         map[string]interface{}     `json:"context,omitempty"`
}

type Filter struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type MeasureSuggestionRequest struct {
	Metrics    []string               `json:"metrics"`
	Dimensions []string               `json:"dimensions"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

type ParameterValidationRequest struct {
	Parameters []dynamic.DynamicParameter `json:"parameters"`
}

type CubeConfigRequest struct {
	Parameters      []dynamic.DynamicParameter `json:"parameters"`
	DynamicMeasures []dynamic.DynamicMeasure   `json:"dynamic_measures"`
}
