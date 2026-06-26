package workflows

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/bp"
)

func TestBPStepAdaptersRoundTrip(t *testing.T) {
	id := uuid.New()
	assignee := "approver"
	cfg := map[string]interface{}{"key": "value"}
	rawCfg, _ := json.Marshal(cfg)

	src := bp.BPStep{
		ID:            id,
		StepOrder:     2,
		StepType:      "approve",
		StepName:      "Manager Approval",
		AssigneeRole:  &assignee,
		DurationHours: 24,
		Config:        rawCfg,
	}

	w := FromPkgBPStep(src)
	if w.StepID != id.String() {
		t.Fatalf("expected StepID %s got %s", id.String(), w.StepID)
	}
	if w.AssigneeRole != assignee {
		t.Fatalf("expected assignee %s got %s", assignee, w.AssigneeRole)
	}

	back := ToPkgBPStep(w)
	if back.StepName != src.StepName || back.StepType != src.StepType || back.StepOrder != src.StepOrder {
		t.Fatalf("round-trip mismatch: %v -> %v", src, back)
	}
	if back.AssigneeRole == nil || *back.AssigneeRole != assignee {
		t.Fatalf("assignee round-trip mismatch: %v", back.AssigneeRole)
	}
}
