package datapipeline

import (
	"encoding/json"
	"testing"
)

func TestColumnsForInjectsSourceSystem(t *testing.T) {
	cfg, _ := json.Marshal(StagingSinkConfig{
		Table: "staging.account_data", SourceCd: "GOLDENSOURCE", Domain: "ACCOUNT",
	})
	s := &stagingSink{
		cfg: StagingSinkConfig{Table: "staging.account_data", SourceCd: "GOLDENSOURCE", Domain: "ACCOUNT"},
		valid: map[string]bool{
			"_load_run_id": true, "_source_row_num": true, "tenant_id": true,
			"account_cd": true, "source_system": true,
		},
		shape: map[string]colShape{},
	}
	_ = cfg
	rows := []Row{{Num: 1, Data: map[string]any{"account_cd": "A1"}}}
	fields, cols, err := s.columnsFor(rows)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i, f := range fields {
		if f == "__source_cd" && cols[i] == "source_system" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected __source_cd → source_system injection, fields=%v cols=%v", fields, cols)
	}
}
