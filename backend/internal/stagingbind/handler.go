package stagingbind

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// Handler is the staging bindings API (/api/staging-bindings). Every error
// is a catalog message; the tenant comes from the authenticated token only.
type Handler struct {
	Store   *Store
	Editor  *Editor
	Catalog *msgcat.Catalog
	// ActorFrom resolves the caller from the authenticated request.
	ActorFrom func(r *http.Request) (Actor, error)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/staging-bindings", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/resolve", h.resolve)
		r.Get("/changes", h.changes)
		r.Post("/changes", h.propose)
		r.Post("/changes/{id}/approve", h.decide(true))
		r.Post("/changes/{id}/reject", h.decide(false))
		r.Post("/changes/{id}/withdraw", h.withdraw)
	})
}

func (h *Handler) with(w http.ResponseWriter, r *http.Request, fn func(Actor) (any, int, error)) {
	a, err := h.ActorFrom(r)
	if err != nil {
		h.Catalog.WriteError(w, r, "", err)
		return
	}
	body, status, err := fn(a)
	if err != nil {
		h.Catalog.WriteError(w, r, a.TenantID, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return msgcat.MalformedJSON().Wrap(err)
	}
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Store.List(r.Context(), a.TenantID)
		return map[string]any{"bindings": l}, http.StatusOK, err
	})
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		bo, table := r.URL.Query().Get("bo"), r.URL.Query().Get("table")
		b, err := h.Store.Resolve(r.Context(), a.TenantID, bo, table)
		if err == nil && b == nil {
			err = msgNoBinding(table, bo)
		}
		return map[string]any{"binding": b}, http.StatusOK, err
	})
}

func (h *Handler) changes(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		l, err := h.Store.Changes(r.Context(), a.TenantID, r.URL.Query().Get("status"), limit)
		return map[string]any{"changes": l}, http.StatusOK, err
	})
}

func (h *Handler) propose(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		var p Proposal
		if err := decode(r, &p); err != nil {
			return nil, 0, err
		}
		c, err := h.Editor.Propose(r.Context(), a, p)
		return map[string]any{"change": c}, http.StatusCreated, err
	})
}

func (h *Handler) decide(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.with(w, r, func(a Actor) (any, int, error) {
			var in struct {
				Comment string `json:"comment"`
			}
			if r.ContentLength > 0 {
				if err := decode(r, &in); err != nil {
					return nil, 0, err
				}
			}
			c, err := h.Editor.Decide(r.Context(), a, chi.URLParam(r, "id"), approve, in.Comment)
			return map[string]any{"change": c}, http.StatusOK, err
		})
	}
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		c, err := h.Editor.Withdraw(r.Context(), a, chi.URLParam(r, "id"))
		return map[string]any{"change": c}, http.StatusOK, err
	})
}
