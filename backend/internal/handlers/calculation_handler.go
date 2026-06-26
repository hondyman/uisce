package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/boresolver"
	"github.com/hondyman/semlayer/backend/models"
)

type CalculationHandler struct {
	Service  *analytics.SemanticCalculationService
	SQLCache *boresolver.SQLCache
}

func NewCalculationHandler(service *analytics.SemanticCalculationService) *CalculationHandler {
	return &CalculationHandler{
		Service:  service,
		SQLCache: boresolver.NewSQLCache(1000),
	}
}

func (h *CalculationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListCalculations)
	r.Post("/", h.CreateCalculation)
	r.Patch("/{id}", h.UpdateCalculation) // Added patch
	r.Get("/{name}", h.GetCalculation)
	r.Get("/{id}/explain", h.ExplainCalculation)     // Added explain
	r.Post("/explain", h.ExplainExpressionStateless) // Added stateless explain

	return r
}

func (h *CalculationHandler) ListCalculations(w http.ResponseWriter, r *http.Request) {
	calcs, err := h.Service.ListCalculations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(calcs)
}

func (h *CalculationHandler) CreateCalculation(w http.ResponseWriter, r *http.Request) {
	var calc models.Calculation
	if err := json.NewDecoder(r.Body).Decode(&calc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate Expression
	if calc.Formula != "" {
		boID := ""
		if calc.DomainID != nil {
			boID = calc.DomainID.String()
		}
		if errs := h.validateExpression(calc.Formula, boID); len(errs) > 0 {
			writeValidationErrors(w, errs)
			return
		}
	}

	if err := h.Service.CreateCalculation(&calc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(calc)
}

func (h *CalculationHandler) UpdateCalculation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	var calc models.Calculation
	if err := json.NewDecoder(r.Body).Decode(&calc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	calcID, err := uuid.Parse(id)
	if err == nil {
		calc.ID = calcID
	}

	// Validate Expression
	if calc.Formula != "" {
		boID := ""
		if calc.DomainID != nil {
			boID = calc.DomainID.String()
		}
		if errs := h.validateExpression(calc.Formula, boID); len(errs) > 0 {
			writeValidationErrors(w, errs)
			return
		}
	}

	if err := h.Service.UpdateCalculation(&calc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(calc)
}

func (h *CalculationHandler) GetCalculation(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	calc, err := h.Service.GetCalculationByName(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(calc)
}

func (h *CalculationHandler) ExplainCalculation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var calc models.Calculation
	err := h.Service.GetDB().Get(&calc, "SELECT * FROM calculations WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Calculation not found", http.StatusNotFound)
		return
	}

	expr, err := boresolver.ParseExpression(calc.Formula)
	if err != nil {
		http.Error(w, "Cannot parse expression: "+err.Error(), http.StatusBadRequest)
		return
	}

	boID := ""
	if calc.DomainID != nil {
		boID = calc.DomainID.String()
	}
	env, err := analytics.NewCatalogTypeEnv(h.Service.GetDB(), boID)
	if err != nil {
		// Log error?
		env = nil // proceed with limited info or fail?
	}

	isAgg := boresolver.IsAggregateExpr(expr)
	t := boresolver.InferType(expr, env)
	// Resolve SQL (Stateful)
	var resolvedSQL string
	var joins []boresolver.JoinStep

	ctx, err := analytics.NewResolutionContext(h.Service.GetDB(), boID)
	if err == nil {
		ctx.SQLCache = h.SQLCache
		ctx.CalcID = calc.ID.String()
		ctx.VersionHash = calc.UpdatedAt.Format(time.RFC3339)

		// Resolve
		sql, j, err := boresolver.ResolveExpression(calc.Formula, "postgres", ctx)
		if err == nil {
			resolvedSQL = sql
			joins = j
		}
	}

	explanation := boresolver.ExplainExpression(expr, env)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"calculation_id":     calc.ID,
		"business_object_id": calc.DomainID,
		"expression":         calc.Formula,
		"inferred_type":      t,
		"is_aggregate":       isAgg,
		"explanation":        explanation,
		"sql":                resolvedSQL,
		"joins":              joins,
	})
}

func (h *CalculationHandler) ExplainExpressionStateless(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expression string `json:"expression"`
		BOID       string `json:"bo_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expr, err := boresolver.ParseExpression(req.Expression)
	if err != nil {
		http.Error(w, "Cannot parse expression: "+err.Error(), http.StatusBadRequest)
		return
	}

	env, err := analytics.NewCatalogTypeEnv(h.Service.GetDB(), req.BOID)
	// If env creation fails, we proceed with nil env (types unknown)
	if err != nil {
		// optional logging
	}

	// Resolve SQL (with Cache)
	var resolvedSQL string
	var joins []boresolver.JoinStep

	// 1. Build Context
	ctx, err := analytics.NewResolutionContext(h.Service.GetDB(), req.BOID)
	if err == nil {
		// 2. Inject Cache
		ctx.SQLCache = h.SQLCache
		// Use expression hash as CalcID for stateless requests to enable caching of identical ad-hoc queries
		// Simple hash for demo
		ctx.CalcID = "adhoc_" + req.Expression // In production use SHA256
		ctx.VersionHash = "latest"             // Ideally fetch BO version

		// 3. Resolve
		// Assume postgres dialect for preview
		sql, j, err := boresolver.ResolveExpression(req.Expression, "postgres", ctx)
		if err == nil {
			resolvedSQL = sql
			joins = j
		}
	}

	explanation := boresolver.ExplainExpression(expr, env)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"explanation":   explanation,
		"inferred_type": boresolver.InferType(expr, env),
		"is_aggregate":  boresolver.IsAggregateExpr(expr),
		"sql":           resolvedSQL,
		"joins":         joins,
	})
}

// Validation helper
func (h *CalculationHandler) validateExpression(expression string, boID string) []string {
	expr, err := boresolver.ParseExpression(expression)
	if err != nil {
		return []string{"Parse Error: " + err.Error()}
	}

	env, err := analytics.NewCatalogTypeEnv(h.Service.GetDB(), boID)
	if err != nil {
		// If env creation fails, maybe DB issue. Return error or skip semantic check?
		// We'll skip precise type check but parser check invalid syntax already.
		return nil
	}

	errs := boresolver.ValidateExpression(expr, env)
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Message)
	}
	return messages
}

func writeValidationErrors(w http.ResponseWriter, errors []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity) // 422
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": errors,
	})
}
