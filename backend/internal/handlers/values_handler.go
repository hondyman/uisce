package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
)

type ValuesHandler struct {
	Service services.ValuesService
}

func NewValuesHandler(service services.ValuesService) *ValuesHandler {
	return &ValuesHandler{Service: service}
}

func (h *ValuesHandler) CreateValueTheme(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	theme, err := h.Service.CreateValueTheme(r.Context(), input.Name, input.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(theme)
}

func (h *ValuesHandler) GetValueThemes(w http.ResponseWriter, r *http.Request) {
	themes, err := h.Service.GetValueThemes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(themes)
}

func (h *ValuesHandler) CreateValueSignal(w http.ResponseWriter, r *http.Request) {
	var input services.ValueSignalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	signal, err := h.Service.CreateValueSignal(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signal)
}

func (h *ValuesHandler) GetValueSignals(w http.ResponseWriter, r *http.Request) {
	issuerID := r.URL.Query().Get("issuer_id")
	if issuerID == "" {
		http.Error(w, "issuer_id is required", http.StatusBadRequest)
		return
	}

	signals, err := h.Service.GetValueSignals(r.Context(), issuerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

func (h *ValuesHandler) CreateClientValuesProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ClientID           string     `json:"client_id"`
		StrategyTemplateID *uuid.UUID `json:"strategy_template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.Service.CreateClientValuesProfile(r.Context(), input.ClientID, input.StrategyTemplateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *ValuesHandler) GetClientValuesProfile(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		http.Error(w, "clientID is required", http.StatusBadRequest)
		return
	}

	profile, err := h.Service.GetClientValuesProfile(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *ValuesHandler) UpdateClientValuesProfile(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientID")
	if clientID == "" {
		http.Error(w, "clientID is required", http.StatusBadRequest)
		return
	}

	var input struct {
		Preferences json.RawMessage `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.Service.UpdateClientValuesProfile(r.Context(), clientID, input.Preferences)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *ValuesHandler) CreateConstraint(w http.ResponseWriter, r *http.Request) {
	var input services.ConstraintInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	constraint, err := h.Service.CreateConstraint(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(constraint)
}

func (h *ValuesHandler) GetConstraints(w http.ResponseWriter, r *http.Request) {
	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		http.Error(w, "profile_id is required", http.StatusBadRequest)
		return
	}
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile_id", http.StatusBadRequest)
		return
	}

	constraints, err := h.Service.GetConstraints(r.Context(), profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(constraints)
}

func (h *ValuesHandler) RegisterRoutes(r chi.Router) {
	r.Route("/values", func(r chi.Router) {
		r.Post("/themes", h.CreateValueTheme)
		r.Get("/themes", h.GetValueThemes)
		r.Post("/signals", h.CreateValueSignal)
		r.Get("/signals", h.GetValueSignals)
		r.Post("/profiles", h.CreateClientValuesProfile)
		r.Get("/profiles/{clientID}", h.GetClientValuesProfile)
		r.Put("/profiles/{clientID}", h.UpdateClientValuesProfile)
		r.Post("/constraints", h.CreateConstraint)
		r.Get("/constraints", h.GetConstraints)
	})
}
