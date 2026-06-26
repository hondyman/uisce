package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// getTenantContext extracts tenant_id from authenticated user and injects it into context
func getTenantContext(ctx context.Context) context.Context {
	if user, ok := auth.GetUserFromContext(ctx); ok {
		if user.TenantID != "" {
			return context.WithValue(ctx, "tenant_id", user.TenantID)
		}
	}
	// Default to uisce if no tenant found
	return context.WithValue(ctx, "tenant_id", "uisce")
}

// AbbreviationHandler exposes CRUD for abbreviations
type AbbreviationHandler struct {
	svc *services.AbbreviationService
}

// NewAbbreviationHandler creates a handler
func NewAbbreviationHandler(svc *services.AbbreviationService) *AbbreviationHandler {
	return &AbbreviationHandler{svc: svc}
}

// RegisterRoutes mounts abbreviation routes under /abbreviations
func (h *AbbreviationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/abbreviations", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/export", h.export) // Add export endpoint
		r.Post("/expand", h.expand)
		r.Post("/validate", h.validate)
		r.Post("/scan", h.scan)
		r.Post("/suggest", h.suggest)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})
}

func (h *AbbreviationHandler) export(w http.ResponseWriter, r *http.Request) {
	ctx := getTenantContext(r.Context())
	abbreviations, err := h.svc.GetAllAbbreviations(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment;filename=abbreviations.csv")

		// Write CSV header with tenant_id and is_core
		w.Write([]byte("id,abbreviation,full_word,notes,tenant_id,is_core\n"))

		for _, abbr := range abbreviations {
			// Simple CSV escaping
			row := fmt.Sprintf("%d,%s,%s,%s,%s,%t\n",
				abbr.ID,
				escapeCSV(abbr.Abbreviation),
				escapeCSV(abbr.FullWord),
				escapeCSV(abbr.Notes),
				escapeCSV(abbr.TenantID),
				abbr.IsCore,
			)
			w.Write([]byte(row))
		}
		return
	}

	// Default to JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment;filename=abbreviations.json")
	json.NewEncoder(w).Encode(abbreviations)
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
	}
	return s
}

func (h *AbbreviationHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := getTenantContext(r.Context())

	// Parse pagination params
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	search := r.URL.Query().Get("q")

	params := services.GetAbbreviationsParams{
		Limit:  limit,
		Offset: offset,
		Search: search,
	}

	result, err := h.svc.GetAbbreviations(ctx, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AbbreviationHandler) get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	abbreviations, err := h.svc.GetAllAbbreviations(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, abbr := range abbreviations {
		if abbr.ID == id {
			json.NewEncoder(w).Encode(abbr)
			return
		}
	}

	http.Error(w, "Abbreviation not found", http.StatusNotFound)
}

func (h *AbbreviationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Abbreviation string `json:"abbreviation"`
		FullWord     string `json:"full_word"`
		Notes        string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	err := h.svc.AddAbbreviation(ctx, req.Abbreviation, req.FullWord, req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message":      "Abbreviation created successfully",
		"abbreviation": req.Abbreviation,
		"full_word":    req.FullWord,
	})
}

func (h *AbbreviationHandler) update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Abbreviation string `json:"abbreviation"`
		FullWord     string `json:"full_word"`
		Notes        string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	err = h.svc.UpdateAbbreviation(ctx, id, req.Abbreviation, req.FullWord, req.Notes)
	if err != nil {
		// Check for permission errors
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":      "Abbreviation updated successfully",
		"abbreviation": req.Abbreviation,
		"full_word":    req.FullWord,
	})
}

func (h *AbbreviationHandler) delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	err = h.svc.DeleteAbbreviation(ctx, id)
	if err != nil {
		// Check for permission errors
		if strings.Contains(err.Error(), "permission denied") {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Abbreviation deleted successfully",
	})
}

func (h *AbbreviationHandler) expand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ColumnName string `json:"column_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	result, err := h.svc.ExpandAbbreviations(ctx, req.ColumnName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AbbreviationHandler) validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TermNames []string `json:"term_names"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	result, err := h.svc.ValidateSemanticTerms(ctx, req.TermNames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AbbreviationHandler) scan(w http.ResponseWriter, r *http.Request) {
	ctx := getTenantContext(r.Context())
	candidates, err := h.svc.ScanForAbbreviations(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"candidates": candidates,
		"count":      len(candidates),
	})
}

func (h *AbbreviationHandler) suggest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Candidates []string `json:"candidates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := getTenantContext(r.Context())
	suggestions, err := h.svc.SuggestExpansions(ctx, req.Candidates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
