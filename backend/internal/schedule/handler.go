package schedule

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// Handler is the scheduler API (/api/schedules). Every error is a catalog
// message; the tenant comes from the authenticated token only.
type Handler struct {
	Service *Service
	Catalog *msgcat.Catalog
	// ActorFrom resolves the caller (tenant, user, datasource, region) from
	// the authenticated request.
	ActorFrom func(r *http.Request) (Actor, error)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/schedules", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/kinds", h.kinds)
		r.Get("/targets", h.targets)
		r.Get("/calendars", h.calendars)
		r.Post("/preview", h.preview)
		r.Get("/runs", h.runs)
		r.Get("/runs/{runId}", h.run)
		r.Get("/runs/{runId}/output", h.output)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.remove)
		r.Post("/{id}/pause", h.setEnabled(false))
		r.Post("/{id}/resume", h.setEnabled(true))
		r.Post("/{id}/run", h.runNow)
		r.Get("/{id}/runs", h.scheduleRuns)
		r.Get("/{id}/audit", h.audit)
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
		l, err := h.Service.Store.List(r.Context(), a.TenantID, ListFilter{Kind: r.URL.Query().Get("kind"), Ref: r.URL.Query().Get("ref")})
		if l == nil {
			l = []*Schedule{}
		}
		return map[string]any{"schedules": l}, http.StatusOK, err
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		var in Input
		if err := decode(r, &in); err != nil {
			return nil, 0, err
		}
		sc, err := h.Service.Create(r.Context(), a, in)
		return sc, http.StatusCreated, err
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		sc, err := h.Service.Store.Get(r.Context(), a.TenantID, chi.URLParam(r, "id"))
		return sc, http.StatusOK, err
	})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		var in Input
		if err := decode(r, &in); err != nil {
			return nil, 0, err
		}
		sc, err := h.Service.Update(r.Context(), a, chi.URLParam(r, "id"), in)
		return sc, http.StatusOK, err
	})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		return map[string]any{"deleted": true}, http.StatusOK, h.Service.Delete(r.Context(), a, chi.URLParam(r, "id"))
	})
}

func (h *Handler) setEnabled(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.with(w, r, func(a Actor) (any, int, error) {
			sc, err := h.Service.SetEnabled(r.Context(), a, chi.URLParam(r, "id"), enabled)
			return sc, http.StatusOK, err
		})
	}
}

func (h *Handler) runNow(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		wid, err := h.Service.RunNow(r.Context(), a, chi.URLParam(r, "id"))
		return map[string]any{"started": true, "workflow_id": wid}, http.StatusAccepted, err
	})
}

func (h *Handler) kinds(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		return map[string]any{"kinds": h.Service.Runners.Kinds()}, http.StatusOK, nil
	})
}

func (h *Handler) targets(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		runner, err := h.Service.Runners.Get(r.URL.Query().Get("kind"))
		if err != nil {
			return nil, 0, err
		}
		t, err := runner.Targets(r.Context(), a.TenantID, a.UserID)
		if t == nil {
			t = []TargetInfo{}
		}
		return map[string]any{"targets": t}, http.StatusOK, err
	})
}

func (h *Handler) calendars(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		c, err := h.Service.Calendars.List(r.Context(), a.TenantID)
		if c == nil {
			c = []CalendarInfo{}
		}
		return map[string]any{"calendars": c}, http.StatusOK, err
	})
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		var in struct {
			Timing Timing `json:"timing"`
			Count  int    `json:"count"`
		}
		if err := decode(r, &in); err != nil {
			return nil, 0, err
		}
		u, err := h.Service.Preview(r.Context(), a.TenantID, in.Timing, in.Count)
		return map[string]any{"upcoming": u}, http.StatusOK, err
	})
}

func (h *Handler) runs(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		f, err := runFilterFrom(r)
		if err != nil {
			return nil, 0, err
		}
		l, err := h.Service.Store.Runs(r.Context(), a.TenantID, f)
		return map[string]any{"runs": l}, http.StatusOK, err
	})
}

// runFilterFrom reads the run history filters: status, kind, q (schedule
// name, target kind, error code or run id), from/to (YYYY-MM-DD, to
// inclusive, in UTC) and limit.
func runFilterFrom(r *http.Request) (RunFilter, error) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := RunFilter{Status: q.Get("status"), Kind: q.Get("kind"), Query: q.Get("q"), Limit: limit}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return f, msgBadDate(v)
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return f, msgBadDate(v)
		}
		f.To = t.AddDate(0, 0, 1)
	}
	return f, nil
}

func (h *Handler) scheduleRuns(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		f, err := runFilterFrom(r)
		if err != nil {
			return nil, 0, err
		}
		f.ScheduleID = chi.URLParam(r, "id")
		l, err := h.Service.Store.Runs(r.Context(), a.TenantID, f)
		return map[string]any{"runs": l}, http.StatusOK, err
	})
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		run, err := h.Service.Store.GetRun(r.Context(), a.TenantID, chi.URLParam(r, "runId"))
		if err != nil {
			return nil, 0, err
		}
		// The run's error in the caller's language, never its internal detail.
		var msg any
		if run.ErrorCode.Valid {
			if set, nbr, ok := msgcat.ParseCode(run.ErrorCode.String); ok {
				var params []any
				var ps []string
				_ = json.Unmarshal(run.ErrorParams, &ps)
				for _, p := range ps {
					params = append(params, p)
				}
				msg = h.Catalog.Render(r.Context(), a.TenantID, msgcat.Preferences(r.Header.Get("Accept-Language")),
					msgcat.New(set, nbr, params...), run.ID)
			}
		}
		return map[string]any{"run": run, "error": msg}, http.StatusOK, nil
	})
}

func (h *Handler) output(w http.ResponseWriter, r *http.Request) {
	a, err := h.ActorFrom(r)
	if err != nil {
		h.Catalog.WriteError(w, r, "", err)
		return
	}
	out, err := h.Service.Store.Output(r.Context(), a.TenantID, chi.URLParam(r, "runId"))
	if err != nil {
		h.Catalog.WriteError(w, r, a.TenantID, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+out.FileName+`"`)
	_, _ = w.Write(out.Content)
}

func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	h.with(w, r, func(a Actor) (any, int, error) {
		l, err := h.Service.Store.Audit(r.Context(), a.TenantID, chi.URLParam(r, "id"))
		if l == nil {
			l = []AuditEntry{}
		}
		return map[string]any{"audit": l}, http.StatusOK, err
	})
}
