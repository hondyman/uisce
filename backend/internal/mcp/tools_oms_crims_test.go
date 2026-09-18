package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestToolsOMS_DelegatesCRIMSLoadToTrading is the MCP-surface source receipt
// pairing TestOMSFIX_DelegatesCRIMSLoadToTrading: tools_oms must not own
// orm.order SQL after the unify.
func TestToolsOMS_DelegatesCRIMSLoadToTrading(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "tools_oms.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "trading.LoadOrder") {
		t.Fatal("start_fix_order_entry must call trading.LoadOrder")
	}
	if strings.Contains(body, `FROM orm."order"`) || strings.Contains(body, "omsLoadCRIMSOrder") {
		t.Fatal("tools_oms must not own CRIMS order SQL after unify")
	}
}
