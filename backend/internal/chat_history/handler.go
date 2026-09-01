package chat_history

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/chat-history/sessions", func(r chi.Router) {
		r.Get("/", h.ListSessions)
		r.Get("/export.csv", h.ExportCSV)
		r.Get("/{id}", h.GetSession)
		r.Get("/{id}/messages", h.GetMessages)
		r.Post("/{id}/feedback", h.SubmitFeedback)
		r.Post("/{id}/end", h.EndSession)
	})
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f := h.parseListFilters(r, tenantID, isGlobalAdmin)

	sessions, total, err := h.svc.ListSessions(r.Context(), f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"total":    total,
	})
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	detail, err := h.svc.GetSessionDetail(r.Context(), tenantID, sessionID, isGlobalAdmin)
	if err == ErrSessionNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	tenantID, _, _, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	messages, err := h.svc.GetMessagesOnly(r.Context(), tenantID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}

func (h *Handler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	var req struct {
		Score   int16   `json:"score"`
		Comment *string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.svc.SubmitFeedback(r.Context(), tenantID, sessionID, userID, isGlobalAdmin, req.Score, req.Comment); err != nil {
		if err == ErrSessionNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) EndSession(w http.ResponseWriter, r *http.Request) {
	tenantID, _, _, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	if err := h.svc.EndSession(r.Context(), tenantID, sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f := h.parseListFilters(r, tenantID, isGlobalAdmin)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"chat-history-%s.csv\"", time.Now().Format("2006-01-02")))

	if err := h.svc.StreamCSV(r.Context(), w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// parseListFilters applies query parameters uniformly for both ListSessions and ExportCSV.
func (h *Handler) parseListFilters(r *http.Request, tenantID uuid.UUID, isGlobalAdmin bool) ListFilters {
	q := r.URL.Query()
	var f ListFilters

	f.AllTenants = isGlobalAdmin && q.Get("all_tenants") == "true"
	if targetTenant := q.Get("tenant_id"); targetTenant != "" && isGlobalAdmin {
		if tid, err := uuid.Parse(targetTenant); err == nil {
			f.TenantID = &tid
		}
	} else if !f.AllTenants {
		f.TenantID = &tenantID
	}

	if v := q.Get("agent_id"); v != "" {
		f.AgentID = &v
	}
	if v := q.Get("view_type"); v != "" {
		f.ViewType = &v
	}
	if v := q.Get("feedback"); v != "" {
		f.Feedback = &v
	}
	if v := q.Get("search"); v != "" {
		f.Search = &v
	}
	if v := q.Get("embedded"); v != "" {
		b := v == "true"
		f.Embedded = &b
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.FromDate = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.ToDate = &t
		}
	}
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		f.Limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		f.Offset = v
	}
	return f
}

// extractAuthContext reads JWT claims ONLY. The X-Tenant-ID / X-User-Roles
// headers are client-controlled and are never used as a trust boundary.
func extractAuthContext(r *http.Request) (tenantID uuid.UUID, userID string, isGlobalAdmin bool, ok bool) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return uuid.Nil, "", false, false
	}

	tid, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return uuid.Nil, "", false, false
	}

	isGlobal := false
	for _, role := range claims.Roles {
		if role == "global_admin" || role == "global_ops" {
			isGlobal = true
			break
		}
	}

	return tid, claims.UserID, isGlobal, true
}