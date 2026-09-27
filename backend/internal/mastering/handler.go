package mastering

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// runTimeout bounds a manual run started from the API.
const runTimeout = 2 * time.Hour

// Actor is the caller: tenant and user from the authenticated token only.
type Actor struct {
	TenantID string
	UserID   string
	Name     string
	// CanRun: may start mastering runs and act as a steward (propose and
	// approve overrides, merge duplicates): administrators and data stewards.
	CanRun bool
	// Admin: may change the entity's override policy.
	Admin bool
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
			r.Post("/candidates/{id}/decide", h.decide)
			r.Post("/merges/{id}/approve", h.voteMerge(true))
			r.Post("/merges/{id}/reject", h.voteMerge(false))
			r.Post("/merges/{id}/withdraw", h.withdrawMerge)
			r.Get("/policy", h.getPolicy)
			r.Put("/policy", h.setPolicy)
			r.Get("/overrides", h.overrides)
			r.Post("/golden/{id}/overrides", h.proposeOverride)
			r.Post("/overrides/{id}/approve", h.voteOverride(true))
			r.Post("/overrides/{id}/reject", h.voteOverride(false))
			r.Post("/overrides/{id}/withdraw", h.withdrawOverride)
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
		// The run continues whatever happens to this request (a closed tab,
		// a proxy timeout): it is recorded, and GET /runs/{id} follows it.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), runTimeout)
		go func() {
			defer cancel()
			_, _ = h.Engine.Execute(ctx, a.TenantID, run, req)
		}()
		return map[string]any{"run": run}, http.StatusAccepted, nil
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
		version, _ := strconv.Atoi(r.URL.Query().Get("version"))
		d, err := h.Engine.GoldenByID(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), version)
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
		l, err := h.Engine.Candidates(r.Context(), a.TenantID, chi.URLParam(r, "entity"), r.URL.Query().Get("status"), a.UserID, limit(r))
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

// decide: a steward merges a possible duplicate pair or says they differ.
func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		if !a.CanRun {
			return nil, 0, msgNotAllowed()
		}
		var d CandidateDecision
		dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&d); err != nil {
			return nil, 0, msgcat.MalformedJSON().Wrap(err)
		}
		res, err := h.Engine.DecideCandidate(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), d, a.UserID, a.Name)
		return map[string]any{"decision": res}, http.StatusOK, err
	})
}

func decodeBody(r *http.Request, v any) error {
	if r.ContentLength == 0 {
		return nil
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return msgcat.MalformedJSON().Wrap(err)
	}
	return nil
}

func (h *Handler) getPolicy(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		p, err := h.Engine.GetPolicy(r.Context(), a.TenantID, chi.URLParam(r, "entity"))
		return map[string]any{"policy": p, "can_edit": a.Admin}, http.StatusOK, err
	})
}

// setPolicy: administrators choose whether the entity's overrides need
// approval (and how many) or apply directly. Every change is audited.
func (h *Handler) setPolicy(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		if !a.Admin {
			return nil, 0, msgNotAdmin()
		}
		var in Policy
		if err := decodeBody(r, &in); err != nil {
			return nil, 0, err
		}
		p, err := h.Engine.SetPolicy(r.Context(), a.TenantID, chi.URLParam(r, "entity"), in, a.Name)
		return map[string]any{"policy": p, "can_edit": true}, http.StatusOK, err
	})
}

func (h *Handler) overrides(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		q := r.URL.Query()
		l, err := h.Engine.Overrides(r.Context(), a.TenantID, chi.URLParam(r, "entity"), q.Get("status"), q.Get("golden"), a.UserID, limit(r))
		return map[string]any{"overrides": l}, http.StatusOK, err
	})
}

func (h *Handler) proposeOverride(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		if !a.CanRun {
			return nil, 0, msgNotAllowed()
		}
		var in OverrideRequest
		if err := decodeBody(r, &in); err != nil {
			return nil, 0, err
		}
		o, err := h.Engine.ProposeOverride(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), in, a.UserID, a.Name)
		return map[string]any{"override": o}, http.StatusCreated, err
	})
}

func (h *Handler) voteOverride(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.with(w, r, func(a Actor) (any, int, error) {
			if !a.CanRun {
				return nil, 0, msgNotAllowed()
			}
			var in struct {
				Comment string `json:"comment"`
			}
			if err := decodeBody(r, &in); err != nil {
				return nil, 0, err
			}
			o, err := h.Engine.DecideOverride(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), approve, in.Comment, a.UserID, a.Name)
			return map[string]any{"override": o}, http.StatusOK, err
		})
	}
}

func (h *Handler) withdrawOverride(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		o, err := h.Engine.WithdrawOverride(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), a.UserID)
		return map[string]any{"override": o}, http.StatusOK, err
	})
}

func (h *Handler) voteMerge(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.with(w, r, func(a Actor) (any, int, error) {
			if !a.CanRun {
				return nil, 0, msgNotAllowed()
			}
			var in struct {
				Comment string `json:"comment"`
			}
			if err := decodeBody(r, &in); err != nil {
				return nil, 0, err
			}
			res, err := h.Engine.DecideMerge(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), approve, in.Comment, a.UserID, a.Name)
			return map[string]any{"decision": res}, http.StatusOK, err
		})
	}
}

func (h *Handler) withdrawMerge(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		err := h.Engine.WithdrawMerge(r.Context(), a.TenantID, chi.URLParam(r, "entity"), chi.URLParam(r, "id"), a.UserID)
		return map[string]any{"ok": err == nil}, http.StatusOK, err
	})
}
