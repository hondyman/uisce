package fix_test

import (
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/fix"
)

func TestNewServer_FixAcceptorPort_Default(t *testing.T) {
	os.Unsetenv("FIX_ACCEPTOR_PORT")
	_, err := fix.NewServer(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewServer_FixAcceptorPort_EnvOverride(t *testing.T) {
	os.Setenv("FIX_ACCEPTOR_PORT", "9980")
	defer os.Unsetenv("FIX_ACCEPTOR_PORT")

	_, err := fix.NewServer(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

