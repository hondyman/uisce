package msgcat

import (
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/security"
)

// Handler is the catalog's API (/api/message-catalog), used by the catalog
// editor and by API clients alike. Reads are open to any authenticated
// caller; edits and approvals follow Actor's permissions.
type Handler struct {
	store   *Store
	catalog *Catalog
	editor  *Editor
}

func NewHandler(store *Store, catalog *Catalog) *Handler {
	return &Handler{store: store, catalog: catalog, editor: NewEditor(store, catalog)}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/message-catalog", func(r chi.Router) {
		r.Get("/me", h.me)
		r.Get("/languages", h.languages)
		r.Get("/sets", h.sets)
		r.Get("/messages", h.messages)
		r.Get("/messages/{set}/{nbr}", h.message)
		r.Get("/render", h.render)
		r.Get("/changes", h.changes)
		r.Post("/changes", h.propose)
		r.Post("/changes/{id}/approve", h.decide(true))
		r.Post("/changes/{id}/reject", h.decide(false))
		r.Post("/changes/{id}/withdraw", h.withdraw)
	})
}

// actor identifies the caller. The tenant is the caller's own (or, for a
// platform administrator, the one selected), never taken from the request
// unchecked.
func actor(r *http.Request) (Actor, bool) {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok || auth.UserID == "" {
		return Actor{}, false
	}
	a := Actor{UserID: auth.UserID, Name: claimString(auth.RawClaims, "Email")}
	if t, ok := security.ResolveTenantID(auth, strings.TrimSpace(r.Header.Get("X-Tenant-ID"))); ok {
		a.TenantID = t
	}
	for _, role := range auth.Roles {
		switch role {
		case "global_admin":
			a.PlatformAdmin = true
		case "tenant_admin":
			a.TenantAdmin = true
		}
	}
	return a, true
}

// claimString reads a string field from the validated token claims (their
// concrete type lives in the services package).
func claimString(claims any, field string) string {
	v := reflect.ValueOf(claims)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	if f := v.FieldByName(field); f.IsValid() && f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, a Actor, err error) {
	h.catalog.WriteError(w, r, a.TenantID, err)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// authed runs fn for an authenticated caller.
func (h *Handler) authed(w http.ResponseWriter, r *http.Request, fn func(Actor)) {
	a, ok := actor(r)
	if !ok {
		h.fail(w, r, a, Unauthenticated())
		return
	}
	fn(a)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		out := map[string]any{
			"user_id":                a.UserID,
			"tenant_id":              a.TenantID,
			"can_edit_core":          a.CanAdminister(ScopeCore),
			"can_edit_tenant":        a.CanAdminister(ScopeTenant),
			"core_requires_approval": true,
		}
		if a.TenantID != "" {
			req, err := h.store.MakerCheckerRequired(r.Context(), a.TenantID)
			if err != nil {
				h.fail(w, r, a, err)
				return
			}
			out["tenant_requires_approval"] = req
		}
		writeJSON(w, http.StatusOK, out)
	})
}

func (h *Handler) languages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"languages": Languages, "base": BaseLanguage})
}

func (h *Handler) sets(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		sets, err := h.store.Sets(r.Context())
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sets": sets})
	})
}

// MessageView is one message with its core text and the tenant's own, per
// language.
type MessageView struct {
	SetNbr     int              `json:"set_nbr"`
	MessageNbr int              `json:"message_nbr"`
	Code       string           `json:"code"`
	Severity   string           `json:"severity"`
	Core       map[string]Entry `json:"core"`
	Tenant     map[string]Entry `json:"tenant"`
	Pending    int              `json:"pending"`
}

func (h *Handler) views(r *http.Request, a Actor) (map[key]*MessageView, error) {
	ctx := r.Context()
	core, err := h.store.CoreEntries(ctx)
	if err != nil {
		return nil, err
	}
	var tenant []Entry
	if a.TenantID != "" {
		if tenant, err = h.store.TenantEntries(ctx, a.TenantID); err != nil {
			return nil, err
		}
	}
	views := map[key]*MessageView{}
	get := func(e Entry) *MessageView {
		k := key{e.SetNbr, e.MessageNbr}
		if views[k] == nil {
			views[k] = &MessageView{SetNbr: e.SetNbr, MessageNbr: e.MessageNbr, Code: New(e.SetNbr, e.MessageNbr).Code(),
				Core: map[string]Entry{}, Tenant: map[string]Entry{}}
		}
		return views[k]
	}
	for _, e := range core {
		get(e).Core[e.Language] = e
	}
	for _, e := range tenant {
		get(e).Tenant[e.Language] = e
	}
	for _, v := range views {
		if en, ok := v.Tenant[BaseLanguage]; ok {
			v.Severity = en.Severity
		} else if en, ok := v.Core[BaseLanguage]; ok {
			v.Severity = en.Severity
		}
	}
	if a.CanAdminister(ScopeCore) || a.CanAdminister(ScopeTenant) {
		pending, err := h.store.Changes(ctx, ChangeFilter{TenantID: a.TenantID, IncludeCore: a.PlatformAdmin, Status: "pending", Limit: 500})
		if err != nil {
			return nil, err
		}
		for _, c := range pending {
			if v := views[key{c.SetNbr, c.MessageNbr}]; v != nil {
				v.Pending++
			}
		}
	}
	return views, nil
}

func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		views, err := h.views(r, a)
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		set, _ := strconv.Atoi(r.URL.Query().Get("set"))
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		out := []*MessageView{}
		for _, v := range views {
			if set != 0 && v.SetNbr != set {
				continue
			}
			if q != "" && !matches(v, q) {
				continue
			}
			out = append(out, v)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].SetNbr != out[j].SetNbr {
				return out[i].SetNbr < out[j].SetNbr
			}
			return out[i].MessageNbr < out[j].MessageNbr
		})
		total := len(out)
		if len(out) > 500 {
			out = out[:500]
		}
		writeJSON(w, http.StatusOK, map[string]any{"messages": out, "total": total})
	})
}

func matches(v *MessageView, q string) bool {
	if strings.Contains(v.Code, q) {
		return true
	}
	for _, m := range []map[string]Entry{v.Core, v.Tenant} {
		for _, e := range m {
			if strings.Contains(strings.ToLower(e.Text), q) || strings.Contains(strings.ToLower(e.Description), q) {
				return true
			}
		}
	}
	return false
}

func (h *Handler) message(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		set, err1 := strconv.Atoi(chi.URLParam(r, "set"))
		nbr, err2 := strconv.Atoi(chi.URLParam(r, "nbr"))
		if err1 != nil || err2 != nil {
			h.fail(w, r, a, InvalidInput("message code"))
			return
		}
		views, err := h.views(r, a)
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		v := views[key{set, nbr}]
		if v == nil {
			h.fail(w, r, a, msg(8, New(set, nbr).Code()).WithStatus(http.StatusNotFound))
			return
		}
		var history []Change
		if a.CanAdminister(ScopeCore) || a.CanAdminister(ScopeTenant) {
			if history, err = h.store.Changes(r.Context(), ChangeFilter{TenantID: a.TenantID, IncludeCore: a.PlatformAdmin, SetNbr: set, MessageNbr: nbr, Limit: 100}); err != nil {
				h.fail(w, r, a, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": v, "history": history})
	})
}

// render previews a message as a caller would see it: ?code=set-nbr,
// ?params=a|b (pipe-separated), ?lang= (else Accept-Language).
func (h *Handler) render(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		set, nbr, ok := ParseCode(r.URL.Query().Get("code"))
		if !ok {
			h.fail(w, r, a, InvalidInput("code"))
			return
		}
		langs := Preferences(r.Header.Get("Accept-Language"))
		if l := r.URL.Query().Get("lang"); l != "" {
			langs = Preferences(l)
		}
		if _, found := h.catalog.Lookup(r.Context(), a.TenantID, langs, set, nbr); !found {
			h.fail(w, r, a, msg(8, New(set, nbr).Code()).WithStatus(http.StatusNotFound))
			return
		}
		var params []any
		if p := r.URL.Query().Get("params"); p != "" {
			for _, s := range strings.Split(p, "|") {
				params = append(params, s)
			}
		}
		writeJSON(w, http.StatusOK, h.catalog.Render(r.Context(), a.TenantID, langs, New(set, nbr, params...), CorrelationID(r)))
	})
}

func (h *Handler) changes(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		if !a.CanAdminister(ScopeCore) && !a.CanAdminister(ScopeTenant) {
			h.fail(w, r, a, msg(12).WithStatus(http.StatusForbidden))
			return
		}
		q := r.URL.Query()
		set, _ := strconv.Atoi(q.Get("set"))
		nbr, _ := strconv.Atoi(q.Get("nbr"))
		status := q.Get("status")
		if status == "all" {
			status = ""
		}
		tenantID := ""
		if a.CanAdminister(ScopeTenant) {
			tenantID = a.TenantID
		}
		list, err := h.store.Changes(r.Context(), ChangeFilter{TenantID: tenantID, IncludeCore: a.PlatformAdmin, Status: status, SetNbr: set, MessageNbr: nbr})
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		if list == nil {
			list = []Change{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"changes": list})
	})
}

// propose accepts one ChangeRequest or {"changes": [...]}.
func (h *Handler) propose(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		var body struct {
			ChangeRequest
			Changes []ChangeRequest `json:"changes"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			h.fail(w, r, a, MalformedJSON().Wrap(err))
			return
		}
		reqs := body.Changes
		if len(reqs) == 0 {
			reqs = []ChangeRequest{body.ChangeRequest}
		}
		res, err := h.editor.Propose(r.Context(), a, reqs)
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		status := http.StatusAccepted
		if res.Applied {
			status = http.StatusOK
		}
		writeJSON(w, status, res)
	})
}

func decodeComment(r *http.Request) string {
	var body struct {
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(nil, r.Body, 64<<10)).Decode(&body)
	return strings.TrimSpace(body.Comment)
}

func (h *Handler) decide(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.authed(w, r, func(a Actor) {
			c, err := h.editor.Decide(r.Context(), a, chi.URLParam(r, "id"), approve, decodeComment(r))
			if err != nil {
				h.fail(w, r, a, err)
				return
			}
			writeJSON(w, http.StatusOK, c)
		})
	}
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	h.authed(w, r, func(a Actor) {
		c, err := h.editor.Withdraw(r.Context(), a, chi.URLParam(r, "id"))
		if err != nil {
			h.fail(w, r, a, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	})
}
