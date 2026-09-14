package fix_test

import (
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/fix"
)

// TestNewServer_NilAdapter_DefaultPort confirms the constructor succeeds
// with no adapter, no DB, no admin API — a smoke test that the wiring
// hasn't broken. This is the legacy mode (in-memory store, no admin
// listener). Per HANDOFF_FIX_OVER_PIPELINE.md §8, this is only
// acceptable when `allow_seq_reset=true` is configured.
func TestNewServer_NilAdapter_DefaultPort(t *testing.T) {
	os.Unsetenv("FIX_ACCEPTOR_PORT")
	_, err := fix.NewServer(nil, "", nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewServer_NilAdapter_EnvOverride(t *testing.T) {
	os.Setenv("FIX_ACCEPTOR_PORT", "9980")
	defer os.Unsetenv("FIX_ACCEPTOR_PORT")

	_, err := fix.NewServer(nil, "", nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestNewServer_AdminAddrWithoutToken_Refuses confirms the safety belt:
// the admin API cannot be enabled without a token. This prevents
// shipping a misconfiguration that would expose the admin API without
// authentication.
func TestNewServer_AdminAddrWithoutToken_Refuses(t *testing.T) {
	_, err := fix.NewServer(nil, "", nil, "127.0.0.1:8981", "")
	if err == nil {
		t.Fatalf("expected error when adminAddr is set but adminToken is empty")
	}
}
