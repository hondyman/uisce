package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// CalcTermHandler exposes CRUD + preview/evaluate for calc terms (real
// vm.Expression-backed calculated semantic terms) - the calc side's
// mirror of ValidationRuleHandler, and the save target the Monaco
// expression surface in AdvancedRuleBuilderPage's calc-term mode posts
// to. A ParseError from the service (bad expression syntax) is reported
// as 400 with its position, not a generic 500 - the whole reason
// vm.ParseExpression returns a position-bearing error type.
type CalcTermHandler struct {
	svc *analytics.CalcTermService
}

func NewCalcTermHandler(svc *analytics.CalcTermService) *CalcTermHandler {
	return &CalcTermHandler{svc: svc}
}

func (h *CalcTermHandler) RegisterRoutes(r chi.Router) {
	r.Route("/calc-terms", func(r chi.Router) {
		r.Post("/", h.handleUpsert)
		r.Get("/", h.handleListByBO)
		r.Get("/{id}", h.handleGetByID)
		r.Post("/preview-sql", h.handlePreviewSQL)
		r.Post("/evaluate", h.handleEvaluate)
	})
}

func (h *CalcTermHandler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req models.UpsertCalcTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID.String()

	desc, err := h.svc.UpsertCalcTerm(r.Context(), req)
	if err != nil {
		writeCalcTermError(w, "calc-term: upsert failed", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

func (h *CalcTermHandler) handleListByBO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	list, err := h.svc.ListByBO(r.Context(), tenantID.String(), boName)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("calc-term: list failed: %v", err)
		http.Error(w, "failed to list calc terms", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"calcTerms": list})
}

func (h *CalcTermHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	desc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "calc term not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

type previewSQLRequest struct {
	BOName     string `json:"bo_name"`
	Expression string `json:"expression"`
}

func (h *CalcTermHandler) handlePreviewSQL(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req previewSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := h.svc.PreviewSQL(r.Context(), tenantID.String(), req.BOName, req.Expression)
	if err != nil {
		writeCalcTermError(w, "calc-term: preview failed", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sql": sql})
}

type evaluateRequest struct {
	Expression string                 `json:"expression"`
	Context    map[string]interface{} `json:"context"`
}

func (h *CalcTermHandler) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	var req evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Evaluate(req.Expression, req.Context)
	if err != nil {
		writeCalcTermError(w, "calc-term: evaluate failed", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"result": result})
}

// writeCalcTermError reports a *vm.ParseError as 400 with its position
// (an authoring mistake, not a server fault), and everything else as
// 500 - logging the server-side ones the way every other handler in this
// file does.
func writeCalcTermError(w http.ResponseWriter, logPrefix string, err error) {
	if perr, ok := err.(*vm.ParseError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": perr.Message, "pos": perr.Pos})
		return
	}
	logging.GetLogger().Sugar().Errorf("%s: %v", logPrefix, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
