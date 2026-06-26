package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/nl_intelligence"
)

// NLIntelligenceHandler handles natural language intelligence requests
type NLIntelligenceHandler struct {
	service *nl_intelligence.NLService
}

// NewNLIntelligenceHandler creates a new NL intelligence handler
func NewNLIntelligenceHandler(service *nl_intelligence.NLService) *NLIntelligenceHandler {
	return &NLIntelligenceHandler{service: service}
}

// Routes returns the router for NL Intelligence
func (h *NLIntelligenceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/interpret", h.Interpret)
	r.Post("/execute", h.Execute)
	r.Post("/summarize", h.Summarize)
	return r
}

// Interpret translates natural language to an intent and query plan
func (h *NLIntelligenceHandler) Interpret(w http.ResponseWriter, r *http.Request) {
	var req nl_intelligence.NLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.Interpret(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// Execute runs a query plan and returns structured data
func (h *NLIntelligenceHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var plan nl_intelligence.QueryPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := h.service.Execute(r.Context(), &plan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// Summarize explains a query result in natural language
func (h *NLIntelligenceHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Question string          `json:"question"`
		Result   json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	summary, err := h.service.Summarize(r.Context(), req.Question, req.Result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}
