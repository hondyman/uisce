package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Classification for BeginTx/BeginTxx files without a choke-point symbol.
const (
	classFenceNeeded = "tenant-table-fenced-needed"
	classNoPolicy    = "no-policy-applies-or-global"
	classChoked      = "uses-choke-point"
)

// Quartet completed this wave (were fence-needed; now use ApplyTenantGUCs).
var quartetChoked = []string{
	"boresolver/save_service.go",
	"handlers/bo_wizard_handler.go",
	"api/onboarding_handler.go",
	"handlers/calc_handler.go",
}

// Remaining known fence-needed paths (tenant-bearing tables + Begin, no choke).
// Grow this list as classification proceeds; shrink as files migrate.
var fenceNeededRemaining = []string{
	// empty after quartet — next wave fills from walk heuristics
}

var tenantTableHints = []string{
	"page_definitions", "business_objects", "business_object_fields",
	"catalog_edge", "calc_fields", "universal_exception_queue",
	"schema_drift_proposals", "tenant_id",
}

func TestBeginTxInventory_DocumentsCutoverBlastRadius(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))

	beginFiles := 0
	unfenced := 0
	fenceNeeded := 0
	noPolicy := 0

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
		hasChoke := strings.Contains(body, "ApplyTenantGUCs") ||
			strings.Contains(body, "WithTenantTransaction") ||
			strings.Contains(body, "WithTenantGoldTransaction")
		if hasChoke {
			return nil
		}
		unfenced++
		if touchesTenantTable(body) {
			fenceNeeded++
		} else {
			noPolicy++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("BeginTx files=%d unfenced=%d class[%s]=%d class[%s]=%d",
		beginFiles, unfenced, classFenceNeeded, fenceNeeded, classNoPolicy, noPolicy)
	t.Logf("flip strategy: staged MCP-first (see cutover.go); fleet flip waits on fence-needed→0")

	for _, rel := range quartetChoked {
		path := filepath.Join(root, rel)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("quartet missing %s", rel)
			continue
		}
		body := string(b)
		if !strings.Contains(body, "ApplyTenantGUCs") {
			t.Errorf("quartet %s must call ApplyTenantGUCs", rel)
		}
	}
	for _, rel := range fenceNeededRemaining {
		path := filepath.Join(root, rel)
		if _, err := os.ReadFile(path); err != nil {
			t.Errorf("fenceNeededRemaining stale: %s", rel)
		}
	}
}

func touchesTenantTable(body string) bool {
	for _, h := range tenantTableHints {
		if strings.Contains(body, h) {
			return true
		}
	}
	return false
}

func TestCutoverStrategy_IsStagedMCPFirst(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "cutover.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "Staged MCP-first") {
		t.Fatal("cutover.go must document staged MCP-first decision")
	}
}
