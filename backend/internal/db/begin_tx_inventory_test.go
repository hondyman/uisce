package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Known files that BeginTx/BeginTxx AND reference tenant-bearing tables
// (page_definitions / business_objects) without ApplyTenantGUCs /
// WithTenantTransaction. Full DATABASE_URL cutover to uisce_mcp_app is
// blocked until these (and the broader ~60-file BeginTx set) migrate or are
// explicitly classified as global/schema-only.
var unfencedTenantTableTxFiles = []string{
	"boresolver/save_service.go",
	"handlers/bo_wizard_handler.go",
	"api/onboarding_handler.go",
	"handlers/calc_handler.go",
}

// TestBeginTxInventory_DocumentsCutoverBlastRadius is the standing inventory
// receipt for the DSN cutover. It does not fail the build on count drift —
// it fails if a known unfenced+tenant-table file loses its Begin without
// gaining the choke point (or is deleted without updating this list).
func TestBeginTxInventory_DocumentsCutoverBlastRadius(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))

	beginFiles := 0
	unfenced := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(b)
		if !strings.Contains(body, "BeginTx(") && !strings.Contains(body, "BeginTxx(") {
			return nil
		}
		beginFiles++
		if !strings.Contains(body, "ApplyTenantGUCs") &&
			!strings.Contains(body, "WithTenantTransaction") &&
			!strings.Contains(body, "WithTenantGoldTransaction") {
			unfenced++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("BeginTx/BeginTxx files under internal/: %d", beginFiles)
	t.Logf("of which without choke-point symbol: %d", unfenced)
	t.Logf("cutover blocked until unfenced paths set GUCs or are classified global-only")

	for _, rel := range unfencedTenantTableTxFiles {
		path := filepath.Join(root, rel)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("inventory stale: missing %s — update unfencedTenantTableTxFiles", rel)
			continue
		}
		body := string(b)
		hasBegin := strings.Contains(body, "BeginTx(") || strings.Contains(body, "BeginTxx(")
		hasChoke := strings.Contains(body, "ApplyTenantGUCs") ||
			strings.Contains(body, "WithTenantTransaction") ||
			strings.Contains(body, "WithTenantGoldTransaction")
		if hasBegin && hasChoke {
			t.Errorf("%s now has choke point — remove from unfencedTenantTableTxFiles", rel)
		}
		if !hasBegin {
			t.Errorf("%s no longer Begins — remove from unfencedTenantTableTxFiles", rel)
		}
	}
}
