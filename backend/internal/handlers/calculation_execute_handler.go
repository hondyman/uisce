package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/models"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Execute is the HTTP entry point to the centralized calc engine — see
// analytics.SemanticCalculationService.ExecuteFormulaCalculation for the
// actual dispatch logic (pushdown SQL vs. boresolver.HostRuntimeExecutor),
// which this handler shares with pkg/workflows' ActivityCalculation
// Temporal activity, so a calc run through a workflow step produces the
// same result as one run through this endpoint.
func (h *CalculationHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var calc models.Calculation
	if err := h.Service.GetDB().Get(&calc, "SELECT * FROM calculations WHERE id = $1", id); err != nil {
		http.Error(w, "Calculation not found", http.StatusNotFound)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	results, tier, err := h.Service.ExecuteFormulaCalculation(r.Context(), claims.TenantID, &calc)
	if err != nil {
		http.Error(w, `{"error":"`+jsonEscape(err.Error())+`"}`, http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tier":    tier,
		"results": results,
	})
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}
