package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestPipelineTools_RegisteredAndRefusedWithoutSurface(t *testing.T) {
	s := NewServer(nil)
	names := map[string]bool{}
	for _, d := range s.ListTools() {
		names[d.Name] = true
	}
	for _, n := range []string{"describe_data_pipeline_steps", "list_data_pipelines", "get_data_pipeline", "check_data_pipeline",
		"draft_data_pipeline", "preview_data_pipeline", "save_data_pipeline", "start_data_pipeline_run",
		"list_data_pipeline_runs", "get_data_pipeline_run"} {
		if !names[n] {
			t.Errorf("tool %s not registered", n)
		}
	}
	_, err := s.CallTool(context.Background(), uuid.MustParse(testTenantID), "list_data_pipelines", json.RawMessage(`{}`))
	if !errors.Is(err, ErrPipelinesUnavailable) {
		t.Fatalf("err = %v, want ErrPipelinesUnavailable", err)
	}
}
