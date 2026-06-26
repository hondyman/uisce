package api

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
)

// simple in-memory session service for unit tests
type memSessionSvc struct {
	store map[string]string
}

func (m *memSessionSvc) StoreSession(ctx context.Context, userID, accessToken, refreshToken string, r *http.Request) error {
	if m.store == nil {
		m.store = map[string]string{}
	}
	m.store[accessToken] = userID
	return nil
}
func (m *memSessionSvc) InvalidateSession(ctx context.Context, token string) error {
	delete(m.store, token)
	return nil
}
func (m *memSessionSvc) VerifySessionToken(ctx context.Context, token string) (string, error) {
	if uid, ok := m.store[token]; ok {
		return uid, nil
	}
	return "", sql.ErrNoRows
}

func TestMemSessionServiceBasicFlow(t *testing.T) {
	m := &memSessionSvc{store: map[string]string{}}
	req := &http.Request{RemoteAddr: "127.0.0.1:1234"}
	if err := m.StoreSession(req.Context(), "user-1", "tok-1", "rt-1", req); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	uid, err := m.VerifySessionToken(req.Context(), "tok-1")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if uid != "user-1" {
		t.Fatalf("expected user-1, got %s", uid)
	}
	if err := m.InvalidateSession(req.Context(), "tok-1"); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}
	_, err = m.VerifySessionToken(req.Context(), "tok-1")
	if err == nil {
		t.Fatalf("expected error after invalidation")
	}
}
