package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/hondyman/uisce/backend/internal/security"
)

// pipelineMCP gives MCP clients the same pipeline surface as the visual
// editor: the same store, grounding, preview, runner and assistant.
type pipelineMCP struct{ h *DataPipelineHandler }

var _ mcp.PipelineTools = pipelineMCP{}

// request builds a request carrying the caller's authentication (from the
// MCP call's context), pinned to the tenant MCP resolved, so grounding goes
// through exactly the checks the editor's requests do.
func (p pipelineMCP) request(ctx context.Context, tenant uuid.UUID) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	r.Header.Set("X-Tenant-ID", tenant.String())
	return r
}

func decodeSpec(raw json.RawMessage) (datapipeline.Spec, error) {
	var s datapipeline.Spec
	if len(raw) == 0 {
		return s, fmt.Errorf("spec is required")
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("spec is not a valid pipeline document: %w", err)
	}
	if s.Version == 0 {
		s.Version = datapipeline.SpecVersion
	}
	return s, nil
}

func problems(err error) (interface{}, error) {
	var probs *errHasProblems
	if errors.As(err, &probs) {
		return map[string]any{"ok": false, "message": probs.Error(), "issues": probs.issues}, nil
	}
	return nil, err
}

func (p pipelineMCP) List(ctx context.Context, t uuid.UUID) (interface{}, error) {
	defs, err := p.h.store.List(ctx, t.String())
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		out = append(out, map[string]any{"id": d.ID, "name": d.Name, "description": d.Description,
			"steps": len(d.Spec.Nodes), "last_modified_at": d.LastModifiedAt})
	}
	return map[string]any{"pipelines": out}, nil
}

func (p pipelineMCP) Get(ctx context.Context, t uuid.UUID, id string) (interface{}, error) {
	return p.h.store.Get(ctx, t.String(), id)
}

func (p pipelineMCP) Describe(context.Context, uuid.UUID) (interface{}, error) {
	return map[string]any{"steps": datapipeline.Palette(p.h.deps), "guide": datapipeline.AuthoringGuide()}, nil
}

func (p pipelineMCP) Check(ctx context.Context, t uuid.UUID, raw json.RawMessage) (interface{}, error) {
	spec, err := decodeSpec(raw)
	if err != nil {
		return nil, err
	}
	issues := p.h.check(p.request(ctx, t), t.String(), &spec)
	return map[string]any{"ok": len(issues) == 0, "issues": issues}, nil
}

func (p pipelineMCP) Save(ctx context.Context, t uuid.UUID, id, name, description string, raw json.RawMessage) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	spec, err := decodeSpec(raw)
	if err != nil {
		return nil, err
	}
	user := ""
	if a, ok := security.AuthInfoFromContext(ctx); ok {
		user = a.UserID
	}
	d := &datapipeline.Definition{ID: id, Name: name, Description: description, Spec: spec}
	if err := p.h.store.Save(ctx, t.String(), user, d); err != nil {
		return nil, err
	}
	issues := p.h.check(p.request(ctx, t), t.String(), &spec)
	return map[string]any{"pipeline_id": d.ID, "saved": true, "ready_to_run": len(issues) == 0, "issues": issues}, nil
}

func (p pipelineMCP) Preview(ctx context.Context, t uuid.UUID, raw json.RawMessage, rows int) (interface{}, error) {
	spec, err := decodeSpec(raw)
	if err != nil {
		return nil, err
	}
	resp, err := p.h.previewCore(p.request(ctx, t), t.String(), spec, rows, 10)
	if err != nil {
		return problems(err)
	}
	return resp, nil
}

func (p pipelineMCP) StartRun(ctx context.Context, t uuid.UUID, id string) (interface{}, error) {
	runID, err := p.h.startRunCore(p.request(ctx, t), t.String(), id)
	if err != nil {
		return problems(err)
	}
	return map[string]any{"ok": true, "run_id": runID, "status": "queued"}, nil
}

func (p pipelineMCP) ListRuns(ctx context.Context, t uuid.UUID, id string) (interface{}, error) {
	runs, err := p.h.store.ListRuns(ctx, t.String(), id, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"runs": runs}, nil
}

func (p pipelineMCP) GetRun(ctx context.Context, t uuid.UUID, runID string) (interface{}, error) {
	return p.h.store.GetRun(ctx, t.String(), runID)
}

func (p pipelineMCP) Draft(ctx context.Context, t uuid.UUID, message string, raw json.RawMessage) (interface{}, error) {
	if p.h.assistant == nil || p.h.catalog == nil {
		return nil, fmt.Errorf("the AI assistant is not configured for this environment")
	}
	var spec datapipeline.Spec
	if len(raw) > 0 {
		var err error
		if spec, err = decodeSpec(raw); err != nil {
			return nil, err
		}
	}
	return p.h.assistant.Respond(ctx, p.h.catalog(p.request(ctx, t), t.String()), datapipeline.AssistRequest{Message: message, Spec: spec})
}

func (p pipelineMCP) SetSchedule(ctx context.Context, t uuid.UUID, id, cron, tz string, enabled bool) (interface{}, error) {
	v, err := p.h.setScheduleCore(ctx, t.String(), id, datapipeline.Schedule{Cron: cron, TimeZone: tz, Enabled: enabled})
	var bad *errBadSchedule
	if errors.As(err, &bad) {
		return map[string]any{"ok": false, "message": bad.Error()}, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
