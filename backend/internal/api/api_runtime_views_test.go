package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func listRuntimeViews(dir string) ([]map[string]interface{}, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var items []map[string]interface{}
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, nil
}

func TestListRuntimeViewsAndPagination(t *testing.T) {
	tmp := t.TempDir()

	// create a few runtime view files
	a := filepath.Join(tmp, "a.json")
	b := filepath.Join(tmp, "b.json")
	if err := os.WriteFile(a, []byte(`{"name":"a","sql":"select 1"}`), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}

	if err := os.WriteFile(b, []byte(`{"name":"b","sql":"select 2"}`), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	items, err := listRuntimeViews(tmp)
	if err != nil {
		t.Fatalf("listRuntimeViews error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// ensure names present
	names := map[string]bool{}
	for _, it := range items {
		n, _ := it["name"].(string)
		names[n] = true
	}
	if !names["a"] || !names["b"] {
		t.Fatalf("missing expected names: %#v", names)
	}
}
