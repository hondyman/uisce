package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// DomainHandler exposes CRUD for data domains
type DomainHandler struct {
	svc *services.DomainService
}

// NewDomainHandler creates a handler
func NewDomainHandler(svc *services.DomainService) *DomainHandler {
	return &DomainHandler{svc: svc}
}

// RegisterRoutes mounts domain routes under /data-domains
func (h *DomainHandler) RegisterRoutes(r chi.Router) {
	r.Route("/data-domains", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/search", h.search)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})
}

func (h *DomainHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.svc.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *DomainHandler) get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if d == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(d)
}

func (h *DomainHandler) create(w http.ResponseWriter, r *http.Request) {
	var in models.DataDomain
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	// simple validation
	if in.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	out, err := h.svc.Create(r.Context(), &in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(out)
}

func (h *DomainHandler) update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var in models.DataDomain
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	in.ID = id
	out, err := h.svc.Update(r.Context(), &in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(out)
}

func (h *DomainHandler) delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DomainHandler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	lstr := r.URL.Query().Get("limit")
	limit := 10
	if lstr != "" {
		if v, err := strconv.Atoi(lstr); err == nil {
			limit = v
		}
	}
	out, err := h.svc.Search(r.Context(), q, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(out)
}
