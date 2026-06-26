package workflows

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/bp"
)

// FromPkgBPSteps converts a slice of pkg/bp.BPStep to workflow-local BPStep
func FromPkgBPSteps(src []bp.BPStep) []BPStep {
	out := make([]BPStep, 0, len(src))
	for _, s := range src {
		out = append(out, FromPkgBPStep(s))
	}
	return out
}

// FromPkgBPStep converts a single pkg/bp.BPStep into the workflow-local BPStep
func FromPkgBPStep(s bp.BPStep) BPStep {
	var cfg map[string]interface{}
	if len(s.Config) > 0 {
		_ = json.Unmarshal(s.Config, &cfg) // best-effort; ignore errors for adapter
	}
	var assignee string
	if s.AssigneeRole != nil {
		assignee = *s.AssigneeRole
	}

	stepID := s.ID.String()
	if s.ID == uuid.Nil {
		stepID = ""
	}

	return BPStep{
		StepID:        stepID,
		StepName:      s.StepName,
		StepType:      s.StepType,
		StepOrder:     int(s.StepOrder),
		DurationHours: int(s.DurationHours),
		AssigneeRole:  assignee,
		Config:        cfg,
	}
}

// ToPkgBPStep converts a workflow-local BPStep into the canonical pkg/bp.BPStep
// Note: ID parsing is best-effort; if StepID is empty or invalid, ID will be uuid.Nil.
func ToPkgBPStep(w BPStep) bp.BPStep {
	var raw json.RawMessage
	if w.Config != nil {
		if b, err := json.Marshal(w.Config); err == nil {
			raw = json.RawMessage(b)
		}
	}

	var assigneePtr *string
	if w.AssigneeRole != "" {
		s := w.AssigneeRole
		assigneePtr = &s
	}

	var id uuid.UUID
	if w.StepID != "" {
		if parsed, err := uuid.Parse(w.StepID); err == nil {
			id = parsed
		}
	}

	return bp.BPStep{
		ID:            id,
		StepOrder:     int16(w.StepOrder),
		StepType:      w.StepType,
		StepName:      w.StepName,
		AssigneeRole:  assigneePtr,
		DurationHours: int16(w.DurationHours),
		Config:        raw,
	}
}
