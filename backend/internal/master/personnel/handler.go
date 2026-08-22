package personnel

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type PersonnelServiceInterface interface {
	List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PersonnelRecord, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*PersonnelRecord, error)
	Create(ctx context.Context, tenantID uuid.UUID, rec *PersonnelRecord) error
	SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error
}

type Handler struct {
	svc PersonnelServiceInterface
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{svc: NewService(db)}
}

func NewHandlerWithService(svc PersonnelServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/master/personnel", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.SoftDelete)
	})
}

func (h *Handler) tenantID(r *http.Request) uuid.UUID {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		return uuid.Nil
	}
	id, _ := uuid.Parse(claims.TenantID)
	return id
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	subtype := r.URL.Query().Get("subtype_code")
	records, err := h.svc.List(r.Context(), tenantID, subtype)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	rec, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if rec == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var rec PersonnelRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	if err := h.svc.Create(r.Context(), tenantID, &rec); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

func (h *Handler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.svc.SoftDelete(r.Context(), tenantID, id); err != nil {
		if err == ErrNotFound {
			http.Error(w, `{"error":"not found or already deleted"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
