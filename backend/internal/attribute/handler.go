package attribute

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	svc     *Service
	valueDB ValueDB
}

func NewHandler(db *sqlx.DB, valueDB ValueDB) *Handler {
	if valueDB == nil {
		valueDB = AlphaValueDB{DB: db}
	}
	return &Handler{svc: NewService(db), valueDB: valueDB}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/attributes", func(r chi.Router) {
		r.Get("/entities", h.ListEntities)
		r.Get("/preview", h.Preview)
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

func (h *Handler) ListEntities(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	entities, err := h.svc.ListEligibleEntities(r.Context(), tenantID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entities == nil {
		entities = []EligibleEntity{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entities": entities})
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
	defs, err := h.svc.List(r.Context(), tenantID, entityType, includeInactive)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if defs == nil {
		defs = []AttributeDef{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"attributes": defs})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	def, err := h.svc.Create(r.Context(), tenantID, in)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, def)
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
	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	def, err := h.svc.Update(r.Context(), tenantID, id, in)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, def)
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
	if err := h.svc.Delete(r.Context(), tenantID, id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeErr(w, status, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}
	q := r.URL.Query()
	req := PreviewRequest{
		TenantID:   tenantID,
		EntityType: q.Get("entity_type"),
		TableRef:   q.Get("table_ref"),
		OrderBy:    q.Get("order_by"),
		Limit:      50,
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "limit must be integer")
			return
		}
		req.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "offset must be integer")
			return
		}
		req.Offset = n
	}

	resp, err := h.svc.Preview(r.Context(), req, h.valueDB)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
