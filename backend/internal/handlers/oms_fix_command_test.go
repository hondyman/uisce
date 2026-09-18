package handlers

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestOMSFIX_DelegatesCRIMSLoadToTrading is the HTTP-surface receipt for the
// CRIMS unify: the live FIX command handler must call trading.LoadOrder (same
// fence as MCP), not own parallel SQL.
func TestOMSFIX_DelegatesCRIMSLoadToTrading(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "oms_fix_command.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "trading.LoadOrder") {
		t.Fatal("OMSFIXCommandHandler must delegate CRIMS load to trading.LoadOrder")
	}
	if strings.Contains(body, `FROM orm."order"`) {
		t.Fatal("HTTP FIX path must not own orm.order SQL after unify")
	}
}
