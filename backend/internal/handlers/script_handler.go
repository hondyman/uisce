package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// ScriptHandler encapsulates the handlers for the script management API.
type ScriptHandler struct {
	Service services.ScriptService
}

// NewScriptHandler creates a new script handler.
func NewScriptHandler(service services.ScriptService) *ScriptHandler {
	return &ScriptHandler{Service: service}
}

// RegisterRoutes adds the script management routes to the router.
func (h *ScriptHandler) RegisterRoutes(r *chi.Mux) {
	r.Route("/api/scripts", func(r chi.Router) {
		r.Get("/", h.listScripts)
		r.Post("/", h.createScript)
		r.Route("/{scriptID}", func(r chi.Router) {
			r.Get("/", h.getScript)
			r.Post("/publish", h.publishScript)
			r.Post("/impact", h.getImpact)
		})
	})
}

func (h *ScriptHandler) listScripts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		user = auth.FallbackUser()
	}
	query := r.URL.Query().Get("query")
	state := r.URL.Query().Get("state")
	scope := r.URL.Query().Get("scope")
	tag := r.URL.Query().Get("tag")
	steward := r.URL.Query().Get("steward")

	scripts, err := h.Service.ListScripts(user, query, state, scope, tag, steward)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(scripts)
}

func (h *ScriptHandler) createScript(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		user = auth.FallbackUser()
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	script, err := h.Service.CreateScript(user, req.Name, req.Description, req.Scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(script)
}

func (h *ScriptHandler) getScript(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		user = auth.FallbackUser()
	}
	scriptID := chi.URLParam(r, "scriptID")

	script, err := h.Service.GetScript(user, scriptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(script)
}

func (h *ScriptHandler) publishScript(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		user = auth.FallbackUser()
	}
	scriptID := chi.URLParam(r, "scriptID")

	err := h.Service.PublishScript(user, scriptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ScriptHandler) getImpact(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		user = auth.FallbackUser()
	}
	scriptID := chi.URLParam(r, "scriptID")
	version := r.URL.Query().Get("version")

	report, err := h.Service.GetImpactReport(user, scriptID, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(report)
}
