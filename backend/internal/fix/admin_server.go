package fix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

// AdminServer is the internal HTTP API exposed by the acceptor process
// on 127.0.0.1:8981 per HANDOFF_FIX_OVER_PIPELINE.md §6 (Amendment 1).
//
// Endpoints (all require X-Fix-Admin-Token header):
//
//	GET  /sessions                          — list active sessions + tenant/broker tags
//	GET  /sessions/{id}                     — single session detail
//	GET  /sessions/{id}/health              — health probe (used by SessionLivenessCheckActivity)
//	POST /sessions/{id}/logon               — initiate logon for the session
//	POST /sessions/{id}/logout              — initiate logout
//	POST /sessions/{id}/reset               — reset sequence numbers + message log
//	                                          (Amendment 3 escape hatch)
//
// Why this exists: Temporal workflows must not own TCP sockets, but they
// DO need to control session lifecycle (logon/logout/reconnect) and
// observe session state. The admin API is the bridge — workflow code
// calls into it via activities; the acceptor process owns the sockets.
type AdminServer struct {
	addr    string
	token   string
	adapter *Adapter
	server  *http.Server

	// mu protects sessionOverrides — per-session overrides set via
	// POST /sessions/{id}/logon or /logout. The acceptor may be
	// holding the actual session; this is a thin policy layer over
	// quickfix's session callbacks.
	mu               sync.RWMutex
	sessionOverrides map[string]SessionState
}

// SessionState is a snapshot of the admin API's view of a session.
type SessionState struct {
	SessionID string    `json:"session_id"`
	TenantID  uuid.UUID `json:"tenant_id,omitempty"`
	BrokerID  uuid.UUID `json:"broker_id,omitempty"`
	LoggedIn  bool      `json:"logged_in"`
	LastSeen  time.Time `json:"last_seen"`
}

// NewAdminServer builds (but does not start) an admin server.
func NewAdminServer(addr, token string, adapter *Adapter) (*AdminServer, error) {
	if addr == "" {
		return nil, errors.New("admin server addr is empty")
	}
	if token == "" {
		return nil, errors.New("admin server token is empty")
	}

	s := &AdminServer{
		addr:             addr,
		token:            token,
		adapter:          adapter,
		sessionOverrides: make(map[string]SessionState),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sessions", s.handleSessions)
	mux.HandleFunc("/sessions/", s.handleSessionByID)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s, nil
}

// Start launches the admin HTTP server in a goroutine.
func (s *AdminServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(50 * time.Millisecond):
		// Listener is up.
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop shuts down the admin HTTP server.
func (s *AdminServer) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
}

// Addr returns the bound address (useful for tests with port 0).
func (s *AdminServer) Addr() string {
	if s.server == nil {
		return s.addr
	}
	return s.server.Addr
}

// authenticate checks the X-Fix-Admin-Token header. Constant-time
// comparison to avoid timing leaks.
func (s *AdminServer) authenticate(r *http.Request) bool {
	got := r.Header.Get("X-Fix-Admin-Token")
	if got == "" || s.token == "" {
		return false
	}
	// Both are short opaque strings; standard library compare is fine.
	return subtleEqual(got, s.token)
}

// subtleEqual is a constant-time string comparison. Inlined to avoid
// importing crypto/subtle for one call.
func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// handleSessions handles GET /sessions.
func (s *AdminServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	sessions := make([]SessionState, 0, len(s.adapter.sessionMap))
	for sid := range s.adapter.sessionMap {
		sessions = append(sessions, SessionState{
			SessionID: sid.String(),
			LoggedIn:  true, // sessionMap is populated on logon, so presence == logged in
		})
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessions)
}

// handleSessionByID handles GET /sessions/{id}, GET /sessions/{id}/health,
// POST /sessions/{id}/logon, POST /sessions/{id}/logout,
// POST /sessions/{id}/reset.
func (s *AdminServer) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := r.URL.Path
	// Expected format: /sessions/{id} or /sessions/{id}/{action}
	// We parse the suffix after /sessions/ manually for clarity.
	const prefix = "/sessions/"
	if len(path) <= len(prefix) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rest := path[len(prefix):]
	var idStr, action string
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			idStr = rest[:i]
			action = rest[i+1:]
			break
		}
	}
	if idStr == "" {
		idStr = rest
	}

	sessionID := quickfix.SessionID{}
	if err := parseSessionIDString(idStr, &sessionID); err != nil {
		http.Error(w, "invalid session id: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch action {
	case "":
		s.serveSessionDetail(w, r, sessionID)
	case "health":
		s.serveHealth(w, r, sessionID)
	case "logon":
		s.serveLogon(w, r, sessionID)
	case "logout":
		s.serveLogout(w, r, sessionID)
	case "reset":
		s.serveReset(w, r, sessionID)
	case "send":
		s.serveSend(w, r, sessionID)
	default:
		http.Error(w, "unknown action: "+action, http.StatusBadRequest)
	}
}

type adminSendRequest struct {
	MsgType string            `json:"msgType"`
	Fields  map[string]string `json:"fields"`
}

// serveSend is the activity-facing outbound path. Temporal
// SendFixOrderActivity POSTs a NewOrderSingle (or cancel/replace)
// here; the acceptor is the only process allowed to call
// quickfix.SendToTarget.
func (s *AdminServer) serveSend(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req adminSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.MsgType == "" {
		http.Error(w, "msgType is required", http.StatusBadRequest)
		return
	}
	msg := quickfix.NewMessage()
	begin := sessionID.BeginString
	if begin == "" {
		begin = "FIX.4.4"
	}
	msg.Header.SetString(tag.BeginString, begin)
	msg.Header.SetInt(tag.BodyLength, 0)
	msg.Header.SetString(tag.MsgType, req.MsgType)
	for k, v := range req.Fields {
		if k == "" {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(k, "%d", &n); err != nil || n <= 0 {
			continue
		}
		msg.Body.SetString(quickfix.Tag(n), v)
	}
	if err := s.adapter.SendMessage(msg, sessionID); err != nil {
		http.Error(w, "send failed: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent", "session_id": sessionID.String(), "msg_type": req.MsgType})
}

func (s *AdminServer) serveSessionDetail(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state := SessionState{
		SessionID: sessionID.String(),
		LastSeen:  time.Now().UTC(),
	}
	if s.adapter.resolver != nil {
		if t, b, ok := s.adapter.resolver.Resolve(sessionID); ok {
			state.TenantID = t
			state.BrokerID = b
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *AdminServer) serveHealth(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.adapter.sessionMu.RLock()
	_, loggedIn := s.adapter.sessionMap[sessionID]
	s.adapter.sessionMu.RUnlock()

	resp := map[string]interface{}{
		"session_id": sessionID.String(),
		"healthy":    loggedIn,
		"checked_at": time.Now().UTC(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *AdminServer) serveLogon(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// On an acceptor, logon is broker-driven (counterparty connects and
	// sends Logon). There is no programmatic "force logon" in
	// quickfix v0.9.0's public API. Return 501 to make this explicit
	// rather than silently no-op.
	http.Error(w, "logon is broker-driven on acceptor; no programmatic force-logon available in quickfix v0.9.0", http.StatusNotImplemented)
}

func (s *AdminServer) serveLogout(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Build a Logout message (MsgType=5) and send via SendToTarget.
	// This works on both acceptor and initiator sides; the counterparty
	// receives the logout and disconnects.
	logout := quickfix.NewMessage()
	logout.Header.SetString(tagBeginString, "FIX.4.4")
	logout.Header.SetInt(tagBodyLength, 0)
	logout.Header.SetString(tagMsgType, "5")
	logout.Body.SetString(tag.Text, "admin-initiated logout")
	if err := quickfix.SendToTarget(logout, sessionID); err != nil {
		http.Error(w, fmt.Sprintf("logout send failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logout_sent"})
}

func (s *AdminServer) serveReset(w http.ResponseWriter, r *http.Request, sessionID quickfix.SessionID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Reset requires recreating the MessageStore. quickfix's API for
	// this is to reset the session, which clears in-memory state. For
	// Postgres-backed stores, the Reset method on the store is what
	// actually clears the rows.
	//
	// We do not yet have direct access to the per-session MessageStore
	// from the adapter (quickfix owns it). This is a TODO for the
	// acceptance test (build step #13): the reset must propagate to
	// Postgres. For now, mark the session as needing a full reconnect.
	s.mu.Lock()
	s.sessionOverrides[sessionID.String()] = SessionState{
		SessionID: sessionID.String(),
		LoggedIn:  false,
		LastSeen:  time.Now().UTC(),
	}
	s.mu.Unlock()

	// Reset the session's sequence numbers + message log. quickfix's
	// ResetSession calls MessageStore.Reset() under the hood, which our
	// Postgres store implements to clear the fix_session_state and
	// fix_message_store rows. The next logon from the broker rebuilds
	// from sequence 1 — use only when `allow_seq_reset=true` per
	// HANDOFF_FIX_OVER_PIPELINE.md §8.
	if err := quickfix.ResetSession(sessionID); err != nil {
		http.Error(w, fmt.Sprintf("reset failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reset_complete"})
}

// parseSessionIDString decodes a FIX session id string like
// "FIX.4.4:SENDER->TARGET" into a quickfix.SessionID. quickfix's
// SessionID.String() is stable in this format.
func parseSessionIDString(s string, out *quickfix.SessionID) error {
	// FIX.4.4:COMP1->COMP2
	const sep1 = ":"
	const sep2 = "->"
	i1 := -1
	i2 := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ':' && i1 == -1 {
			i1 = i
			continue
		}
		if i+1 < len(s) && s[i] == '-' && s[i+1] == '>' {
			i2 = i
			break
		}
	}
	if i1 == -1 || i2 == -1 || i2 <= i1 {
		return errors.New("expected format FIX.x.y:SENDER->TARGET")
	}
	*out = quickfix.SessionID{
		BeginString:  s[:i1],
		SenderCompID: s[i1+1 : i2],
		TargetCompID: s[i2+2:],
	}
	return nil
}
