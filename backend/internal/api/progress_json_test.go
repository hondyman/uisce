package api

import (
	"encoding/json"
	"testing"
)

// The frontend (useJobPolling.ts JobStatus) reads these exact lowercase keys.
// Untagged fields marshal as "Status"/"Done"/..., which left the progress
// modal stuck at 0/0 and "running" forever.
func TestJobJSONContract(t *testing.T) {
	job := Job{ID: "j1", TenantID: "t1", DatasourceID: "d1", Status: JobStatusCompleted, Total: 30, Done: 29, Failed: 1}
	raw, err := json.Marshal(job.snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "tenant_id", "datasource_id", "status", "total", "done", "failed", "results", "errors", "started_at", "finished_at"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing JSON key %q in %s", k, raw)
		}
	}
	if got["status"] != JobStatusCompleted || got["done"] != float64(29) || got["failed"] != float64(1) || got["total"] != float64(30) {
		t.Errorf("wrong values: %s", raw)
	}
}
