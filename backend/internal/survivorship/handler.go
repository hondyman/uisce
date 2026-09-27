package survivorship

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// Handler exposes CRUD for semantic survivorship rules on alpha.
type Handler struct {
	svc *Service
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{svc: NewService(db)}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/mdm/survivorship-rules", func(r chi.Router) {
		r.Get("/sources", h.ListSources)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Patch("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

func (h *Handler) tenantID(r *http.Request) (uuid.UUID, error) {
	tenant, err := security.ResolveTenantForRequest(r)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(tenant)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant id")
	}
	return id, nil
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.svc.ListSourceSystems(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		writeErr(w, http.StatusBadRequest, "entity_type required")
		return
	}
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	rules, err := h.svc.ListRules(r.Context(), tenantID, entityType, includeInactive)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	var in CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	rule, err := h.svc.CreateRule(r.Context(), tenantID, in)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrMissingTerm) || errors.Is(err, ErrUnknownStrategy) || errors.Is(err, ErrInvalidInput) {
			status = http.StatusBadRequest
		} else {
			status = http.StatusInternalServerError
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	rule, err := h.svc.UpdateRule(r.Context(), tenantID, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrUnknownStrategy) || errors.Is(err, ErrInvalidInput) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeactivateRule(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
