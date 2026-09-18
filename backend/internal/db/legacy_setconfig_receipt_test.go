package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Legacy activation receipt: the two warmed set_config sites must call
 // ApplyTenantGUCs (or WithTenantTransaction), not raw set_config alone.
func TestLegacySites_UseApplyTenantGUCs(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))

	checks := []struct {
		rel  string
		want string
	}{
		{"metadata/businessobject_service.go", "ApplyTenantGUCs"},
		{"api/api.go", "ApplyTenantGUCs"},
	}
	for _, c := range checks {
		src, err := os.ReadFile(filepath.Join(root, c.rel))
		if err != nil {
			t.Fatal(err)
		}
		body := string(src)
		if !strings.Contains(body, c.want) {
			t.Errorf("%s must call %s", c.rel, c.want)
		}
		// Still allow set_config only inside ApplyTenantGUCs / tenant_tx.go
	}
}
