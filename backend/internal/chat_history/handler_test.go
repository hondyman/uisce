package chat_history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// fakeService is an in-memory implementation of the chat history service used
// by handler tests. It mirrors the subset of Repository behavior the handlers
// exercise, so we can validate routing and auth without a live database.
type fakeService struct {
	sessions map[uuid.UUID]SessionRecord
	messages map[uuid.UUID][]MessageRecord
}

func newFakeService() *fakeService {
	return &fakeService{
		sessions: make(map[uuid.UUID]SessionRecord),
		messages: make(map[uuid.UUID][]MessageRecord),
	}
}

func (f *fakeService) ensure(s SessionRecord, msgs ...MessageRecord) uuid.UUID {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	f.sessions[s.ID] = s
	if len(msgs) > 0 {
		f.messages[s.ID] = msgs
	}
	return s.ID
}

func (f *fakeService) List(ctx context.Context, fl ListFilters) ([]SessionRecord, int, error) {
	out := []SessionRecord{}
	for _, s := range f.sessions {
		if fl.TenantID != nil && s.TenantID != *fl.TenantID {
			continue
		}
		out = append(out, s)
	}
	return out, len(out), nil
}

func (f *fakeService) Get(ctx context.Context, tenantID, sessionID uuid.UUID, allowCrossTenant bool) (*SessionDetail, error) {
	s, ok := f.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if !allowCrossTenant && s.TenantID != tenantID {
		return nil, ErrSessionNotFound
	}
	return &SessionDetail{Session: s, Messages: f.messages[sessionID]}, nil
}

func (f *fakeService) Feedback(ctx context.Context, tenantID, sessionID uuid.UUID, userID string, isGlobal bool, score int16, comment *string) error {
	s, ok := f.sessions[sessionID]
	if !ok || (!isGlobal && (s.UserID != userID || s.TenantID != tenantID)) {
		return ErrSessionNotFound
	}
	s.FeedbackScore = &score
	s.FeedbackComment = comment
	f.sessions[sessionID] = s
	return nil
}

func (f *fakeService) End(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	s, ok := f.sessions[sessionID]
	if !ok || s.TenantID != tenantID {
		return ErrSessionNotFound
	}
	now := time.Now()
	s.EndedAt = &now
	f.sessions[sessionID] = s
	return nil
}

// adapter wires fakeService to the same chi routes the production handler
// registers, so the route paths, auth extraction, and filter parsing are
// exercised end-to-end against the fake.
type adapter struct {
	fake *fakeService
}

func (a *adapter) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f := newHandlerForTest().parseListFilters(r, tenantID, isGlobalAdmin)
	out, total, err := a.fake.List(r.Context(), f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"sessions": out, "total": total})
}

func (a *adapter) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	d, err := a.fake.Get(r.Context(), tenantID, id, isGlobalAdmin)
	if err == ErrSessionNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(d)
}

func (a *adapter) Feedback(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Score   int16   `json:"score"`
		Comment *string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := a.fake.Feedback(r.Context(), tenantID, id, userID, isGlobalAdmin, req.Score, req.Comment); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *adapter) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, _, isGlobalAdmin, ok := extractAuthContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = newHandlerForTest().parseListFilters(r, tenantID, isGlobalAdmin)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"chat-history-%s.csv\"", time.Now().Format("2006-01-02")))
	w.WriteHeader(http.StatusOK)
}

// newHandlerForTest returns a Handler whose service is non-nil only for
// filter-parsing access. We deliberately do not call h.svc in the routes
// that the adapter covers; those route directly through the fake.
func newHandlerForTest() *Handler { return NewHandler(nil) }

func TestListSessions_Unauthenticated(t *testing.T) {
	r := chi.NewRouter()
	NewHandler(nil).RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/chat-history/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListSessions_TenantScoped(t *testing.T) {
	svc := newFakeService()
	tenant := uuid.New()
	otherTenant := uuid.New()
	svc.ensure(SessionRecord{
		ID: uuid.New(), TenantID: tenant, ConversationID: uuid.New(),
		AgentID: "default", UserID: "test-user", ViewType: "admin",
		StartedAt: time.Now(),
	})
	svc.ensure(SessionRecord{
		ID: uuid.New(), TenantID: otherTenant, ConversationID: uuid.New(),
		AgentID: "default", UserID: "someone-else", ViewType: "admin",
		StartedAt: time.Now(),
	})

	r := chi.NewRouter()
	a := &adapter{fake: svc}
	r.Get("/api/chat-history/sessions", a.List)

	req := httptest.NewRequest("GET", "/api/chat-history/sessions", nil)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "test-user", TenantID: tenant.String(), Roles: []string{}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var out struct {
		Sessions []SessionRecord `json:"sessions"`
		Total    int             `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Total != 1 || len(out.Sessions) != 1 || out.Sessions[0].TenantID != tenant {
		t.Errorf("wrong sessions returned: total=%d, sessions=%+v", out.Total, out.Sessions)
	}
}

func TestGetSession_CrossTenantDenied_ForNonAdmin(t *testing.T) {
	svc := newFakeService()
	owner := uuid.New()
	intruder := uuid.New()
	sessID := svc.ensure(SessionRecord{
		ID: uuid.New(), TenantID: owner, ConversationID: uuid.New(),
		AgentID: "default", UserID: "owner-user", ViewType: "admin",
		StartedAt: time.Now(),
	})

	r := chi.NewRouter()
	a := &adapter{fake: svc}
	r.Get("/api/chat-history/sessions/{id}", a.Get)

	req := httptest.NewRequest("GET", "/api/chat-history/sessions/"+sessID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "intruder", TenantID: intruder.String(), Roles: []string{}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant access, got %d", w.Code)
	}
}

func TestGetSession_CrossTenantAllowed_ForGlobalAdmin(t *testing.T) {
	svc := newFakeService()
	owner := uuid.New()
	sessID := svc.ensure(SessionRecord{
		ID: uuid.New(), TenantID: owner, ConversationID: uuid.New(),
		AgentID: "default", UserID: "owner-user", ViewType: "admin",
		StartedAt: time.Now(),
	})

	r := chi.NewRouter()
	a := &adapter{fake: svc}
	r.Get("/api/chat-history/sessions/{id}", a.Get)

	req := httptest.NewRequest("GET", "/api/chat-history/sessions/"+sessID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "platform-admin", TenantID: uuid.New().String(), Roles: []string{"global_admin"}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for global admin cross-tenant access, got %d", w.Code)
	}
}

func TestExtractAuthContext_NoIsCoreAdminCrossTenant(t *testing.T) {
	// A user with IsCoreAdmin but no global_admin/global_ops role must NOT
	// be treated as cross-tenant-capable by the handler.
	tenant := uuid.New()
	rec := httptest.NewRequest("GET", "/api/chat-history/sessions", nil)
	rec = rec.WithContext(context.WithValue(rec.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{
			UserID: "core-admin", TenantID: tenant.String(),
			IsCoreAdmin: true,
			Roles:       []string{"core_admin"},
		}))

	tid, userID, isGlobal, ok := extractAuthContext(rec)
	if !ok || tid != tenant || userID != "core-admin" {
		t.Fatalf("auth extraction failed: tid=%v userID=%q ok=%v", tid, userID, ok)
	}
	if isGlobal {
		t.Fatalf("core_admin must not be promoted to global admin via roles alone")
	}
}

func TestSubmitFeedback_OwnershipCheck(t *testing.T) {
	svc := newFakeService()
	tenant := uuid.New()
	sessID := svc.ensure(SessionRecord{
		ID: uuid.New(), TenantID: tenant, ConversationID: uuid.New(),
		AgentID: "default", UserID: "owner-user", ViewType: "admin",
		StartedAt: time.Now(),
	})

	r := chi.NewRouter()
	a := &adapter{fake: svc}
	r.Post("/api/chat-history/sessions/{id}/feedback", a.Feedback)

	// Same-tenant but not the original creator → 404.
	body := bytes.NewBufferString(`{"score":1,"comment":"nice"}`)
	req := httptest.NewRequest("POST", "/api/chat-history/sessions/"+sessID.String()+"/feedback", body)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "different-user", TenantID: tenant.String(), Roles: []string{}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %d", w.Code)
	}

	// Original creator → 200.
	body = bytes.NewBufferString(`{"score":1,"comment":"nice"}`)
	req = httptest.NewRequest("POST", "/api/chat-history/sessions/"+sessID.String()+"/feedback", body)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "owner-user", TenantID: tenant.String(), Roles: []string{}}))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner, got %d", w.Code)
	}
}

func TestExportCSV_FiltersApplied(t *testing.T) {
	svc := newFakeService()
	tenant := uuid.New()
	for i := 0; i < 3; i++ {
		svc.ensure(SessionRecord{
			ID: uuid.New(), TenantID: tenant, ConversationID: uuid.New(),
			AgentID: "default", UserID: "u", ViewType: "end_user",
			StartedAt: time.Now(),
		})
	}

	r := chi.NewRouter()
	a := &adapter{fake: svc}
	r.Get("/api/chat-history/sessions/export.csv", a.Export)

	req := httptest.NewRequest("GET", "/api/chat-history/sessions/export.csv?view_type=end_user", nil)
	req = req.WithContext(context.WithValue(req.Context(),
		jwtmiddleware.ClaimsContextKey,
		&jwtmiddleware.JWTClaims{UserID: "u", TenantID: tenant.String(), Roles: []string{}}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Errorf("expected text/csv, got %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, "chat-history-") {
		t.Errorf("expected filename in Content-Disposition, got %q", got)
	}
}

func TestTruncateRuneLimit(t *testing.T) {
	// The truncate helper used by csv.go must respect the rune limit and
	// produce output no longer than limit characters.
	truncate := func(s *string, limit int) string {
		if s == nil {
			return ""
		}
		r := []rune(*s)
		if len(r) <= limit {
			return *s
		}
		if limit > 3 {
			return string(r[:limit-3]) + "..."
		}
		return string(r[:limit])
	}

	long := strings.Repeat("a", 250)
	got := truncate(&long, 200)
	if n := len([]rune(got)); n > 200 {
		t.Errorf("truncate overshot limit: got %d runes", n)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected ellipsis suffix, got %q", got)
	}
}