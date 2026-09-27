package mastering

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// Actor is the caller: tenant and user from the authenticated token only.
type Actor struct {
	TenantID string
	UserID   string
	Name     string
	// CanRun: may start mastering runs (administrators and data stewards).
	CanRun bool
}

// Handler is the mastering API (/api/mastering). Every error is a catalog
// message.
type Handler struct {
	Engine    *Engine
	Catalog   *msgcat.Catalog
	ActorFrom func(r *http.Request) (Actor, error)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/mastering", func(r chi.Router) {
		r.Get("/profiles", h.profiles)
		r.Route("/{entity}", func(r chi.Router) {
			r.Get("/runs", h.runs)
			r.Post("/runs", h.start)
			r.Get("/runs/{id}", h.getRun)
			r.Get("/golden", h.golden)
			r.Get("/golden/{id}", h.goldenByID)
			r.Get("/exceptions", h.exceptions)
			r.Get("/candidates", h.candidates)
			r.Get("/loads", h.loads)
			r.Post("/exceptions/{id}/resolve", h.resolve)
		})
	})
}

func (h *Handler) with(w http.ResponseWriter, r *http.Request, fn func(Actor) (any, int, error)) {
	if h.Engine == nil || h.Engine.Data == nil {
		h.Catalog.WriteError(w, r, "", msgNoDataPlane())
		return
	}
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

func limit(r *http.Request) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return n
}

func (h *Handler) profiles(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Engine.Profiles(r.Context(), a.TenantID)
		return map[string]any{"profiles": l}, http.StatusOK, err
	})
}

func (h *Handler) runs(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Engine.Runs(r.Context(), a.TenantID, chi.URLParam(r, "entity"), limit(r))
		return map[string]any{"runs": l}, http.StatusOK, err
	})
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		run, err := h.Engine.GetRun(r.Context(), a.TenantID, chi.URLParam(r, "id"))
		return map[string]any{"run": run}, http.StatusOK, err
	})
}

// start masters one staging load now and returns the finished run (or the
// earlier run with the same idempotency key).
func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		if !a.CanRun {
			return nil, 0, msgNotAllowed()
		}
		var req RunRequest
		dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return nil, 0, msgcat.MalformedJSON().Wrap(err)
		}
		req.Entity = chi.URLParam(r, "entity")
		if req.DryRun {
			pv, err := h.Engine.Preview(r.Context(), a.TenantID, req)
			return map[string]any{"preview": pv}, http.StatusOK, err
		}
		req.Trigger = "manual"
		req.StartedBy = a.Name
		req.StartedByID = a.UserID
		run, fresh, err := h.Engine.Start(r.Context(), a.TenantID, req)
		if err != nil {
			return nil, 0, err
		}
		if !fresh {
			return map[string]any{"run": run}, http.StatusOK, nil
		}
		run, err = h.Engine.Execute(r.Context(), a.TenantID, run, req)
		if run != nil && err != nil {
			// The run is recorded as failed, with the reason; report it.
			return map[string]any{"run": run}, http.StatusOK, nil
		}
		return map[string]any{"run": run}, http.StatusCreated, err
	})
}

func (h *Handler) golden(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		q := r.URL.Query()
		l, err := h.Engine.Golden(r.Context(), a.TenantID, chi.URLParam(r, "entity"), GoldenFilter{Q: q.Get("q"), Status: q.Get("status"), Limit: limit(r)})
		return map[string]any{"golden": l}, http.StatusOK, err
	})
}

func (h *Handler) goldenByID(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		d, err := h.Engine.GoldenByID(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"))
		return map[string]any{"golden": d}, http.StatusOK, err
	})
}

func (h *Handler) exceptions(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Engine.Exceptions(r.Context(), a.TenantID, chi.URLParam(r, "entity"), r.URL.Query().Get("status"), limit(r))
		return map[string]any{"exceptions": l}, http.StatusOK, err
	})
}

func (h *Handler) candidates(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Engine.Candidates(r.Context(), a.TenantID, chi.URLParam(r, "entity"), r.URL.Query().Get("status"), limit(r))
		return map[string]any{"candidates": l}, http.StatusOK, err
	})
}

func (h *Handler) loads(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Engine.Loads(r.Context(), a.TenantID, chi.URLParam(r, "entity"), limit(r))
		return map[string]any{"loads": l}, http.StatusOK, err
	})
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		if !a.CanRun {
			return nil, 0, msgNotAllowed()
		}
		var in struct {
			Status string `json:"status"`
			Note   string `json:"note"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			return nil, 0, msgcat.MalformedJSON().Wrap(err)
		}
		err := h.Engine.ResolveException(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), in.Status, in.Note, a.Name)
		return map[string]any{"ok": err == nil}, http.StatusOK, err
	})
}
