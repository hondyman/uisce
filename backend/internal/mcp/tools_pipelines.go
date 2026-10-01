package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// PipelineTools is the data-pipeline surface the API server provides (the
// same store, grounding catalog, runner and assistant the visual editor
// uses). The MCP package only describes and routes the tools.
type PipelineTools interface {
	List(ctx context.Context, tenantID uuid.UUID) (interface{}, error)
	Get(ctx context.Context, tenantID uuid.UUID, id string) (interface{}, error)
	Describe(ctx context.Context, tenantID uuid.UUID) (interface{}, error)
	Check(ctx context.Context, tenantID uuid.UUID, spec json.RawMessage) (interface{}, error)
	Save(ctx context.Context, tenantID uuid.UUID, id, name, description string, spec json.RawMessage) (interface{}, error)
	Preview(ctx context.Context, tenantID uuid.UUID, spec json.RawMessage, rows int) (interface{}, error)
	StartRun(ctx context.Context, tenantID uuid.UUID, id string) (interface{}, error)
	ListRuns(ctx context.Context, tenantID uuid.UUID, id string) (interface{}, error)
	GetRun(ctx context.Context, tenantID uuid.UUID, runID string) (interface{}, error)
	Draft(ctx context.Context, tenantID uuid.UUID, message string, spec json.RawMessage) (interface{}, error)
	SetSchedule(ctx context.Context, tenantID uuid.UUID, id, cron, timezone string, enabled bool) (interface{}, error)
}

// ErrPipelinesUnavailable is returned when the server was started without
// the data-pipeline surface (e.g. the standalone mcp-server binary).
var ErrPipelinesUnavailable = errors.New("data pipelines are not available on this MCP server")

// SetPipelines enables the data-pipeline tools.
func (s *Server) SetPipelines(p PipelineTools) *Server {
	s.pipelines = p
	return s
}

func (s *Server) registerPipelineTools() {
	obj := func(props map[string]interface{}, required ...string) map[string]interface{} {
		m := map[string]interface{}{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	str := map[string]string{"type": "string"}
	spec := map[string]interface{}{"type": "object", "description": "The pipeline document; see describe_data_pipeline_steps."}
	tenantNote := " Tenant is extracted from the authenticated JWT."

	s.RegisterTool("describe_data_pipeline_steps",
		"Explains how to build a data pipeline: the step types available in this environment (and why any are unavailable), each step's configuration, and the document format. Call this before drafting a pipeline."+tenantNote,
		obj(map[string]interface{}{}), s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, _ json.RawMessage) (interface{}, error) {
			return p.Describe(ctx, t)
		}))
	s.RegisterTool("list_data_pipelines",
		"Lists the tenant's data pipelines (loaders): name, steps, last edit."+tenantNote,
		obj(map[string]interface{}{}), s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, _ json.RawMessage) (interface{}, error) {
			return p.List(ctx, t)
		}))
	s.RegisterTool("get_data_pipeline",
		"Returns one data pipeline's full document."+tenantNote,
		obj(map[string]interface{}{"pipeline_id": str}, "pipeline_id"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID string `json:"pipeline_id"`
			}
			_ = json.Unmarshal(a, &args)
			return p.Get(ctx, t, args.ID)
		}))
	s.RegisterTool("check_data_pipeline",
		"Validates a pipeline document without saving it: structure plus grounding - every business object, field, validation rule, uploaded file and staging table it names must exist for the tenant. Returns problems per step."+tenantNote,
		obj(map[string]interface{}{"spec": spec}, "spec"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				Spec json.RawMessage `json:"spec"`
			}
			_ = json.Unmarshal(a, &args)
			return p.Check(ctx, t, args.Spec)
		}))
	s.RegisterTool("draft_data_pipeline",
		"Drafts or edits a pipeline from a plain-language request, grounded in the tenant's business objects, fields, rules, files and staging tables, and checked before it is returned. Nothing is saved: review the result, then save_data_pipeline."+tenantNote,
		obj(map[string]interface{}{"request": str, "spec": spec}, "request"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				Request string          `json:"request"`
				Spec    json.RawMessage `json:"spec"`
			}
			_ = json.Unmarshal(a, &args)
			if args.Request == "" {
				return nil, fmt.Errorf("request is required")
			}
			return p.Draft(ctx, t, args.Request, args.Spec)
		}))
	s.RegisterTool("preview_data_pipeline",
		"Runs a pipeline document on its first rows as a dry run: nothing is written (business object rows are still judged by their validation rules). Returns per-step counts, sample rows and every rejected row with its reason."+tenantNote,
		obj(map[string]interface{}{"spec": spec, "rows": map[string]interface{}{"type": "integer", "description": "1-500, default 100"}}, "spec"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				Spec json.RawMessage `json:"spec"`
				Rows int             `json:"rows"`
			}
			_ = json.Unmarshal(a, &args)
			return p.Preview(ctx, t, args.Spec, args.Rows)
		}))
	s.RegisterTool("save_data_pipeline",
		"Saves a pipeline document: creates one (no pipeline_id) or replaces an existing one's name and document. Does not run it."+tenantNote,
		obj(map[string]interface{}{"pipeline_id": str, "name": str, "description": str, "spec": spec}, "name", "spec"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID          string          `json:"pipeline_id"`
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Spec        json.RawMessage `json:"spec"`
			}
			_ = json.Unmarshal(a, &args)
			return p.Save(ctx, t, args.ID, args.Name, args.Description, args.Spec)
		}))
	s.RegisterTool("start_data_pipeline_run",
		"Starts a run of a saved pipeline on Temporal. It writes data: business object rows are judged by their validation rules, staging loads are tracked load runs. Returns the run id; poll get_data_pipeline_run."+tenantNote,
		obj(map[string]interface{}{"pipeline_id": str}, "pipeline_id"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID string `json:"pipeline_id"`
			}
			_ = json.Unmarshal(a, &args)
			return p.StartRun(ctx, t, args.ID)
		}))
	s.RegisterTool("set_data_pipeline_schedule",
		"Runs a saved pipeline on a schedule (Temporal): a 5-field cron expression in an IANA time zone, e.g. \"0 6 * * 1-5\" in \"Europe/Dublin\" for weekdays at 06:00. enabled=false stops it. A run still going when the next is due is not overlapped. Returns the next run times."+tenantNote,
		obj(map[string]interface{}{"pipeline_id": str, "cron": str, "timezone": str, "enabled": map[string]string{"type": "boolean"}}, "pipeline_id", "enabled"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID       string `json:"pipeline_id"`
				Cron     string `json:"cron"`
				TimeZone string `json:"timezone"`
				Enabled  bool   `json:"enabled"`
			}
			_ = json.Unmarshal(a, &args)
			return p.SetSchedule(ctx, t, args.ID, args.Cron, args.TimeZone, args.Enabled)
		}))
	s.RegisterTool("list_data_pipeline_runs",
		"Lists a pipeline's runs, newest first: status and row counts."+tenantNote,
		obj(map[string]interface{}{"pipeline_id": str}, "pipeline_id"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID string `json:"pipeline_id"`
			}
			_ = json.Unmarshal(a, &args)
			return p.ListRuns(ctx, t, args.ID)
		}))
	s.RegisterTool("get_data_pipeline_run",
		"Returns one run: status, per-step counts and timings, and a sample of rejected rows with reasons."+tenantNote,
		obj(map[string]interface{}{"run_id": str}, "run_id"),
		s.pipelineCall(func(ctx context.Context, p PipelineTools, t uuid.UUID, a json.RawMessage) (interface{}, error) {
			var args struct {
				ID string `json:"run_id"`
			}
			_ = json.Unmarshal(a, &args)
			return p.GetRun(ctx, t, args.ID)
		}))
}

func (s *Server) pipelineCall(fn func(context.Context, PipelineTools, uuid.UUID, json.RawMessage) (interface{}, error)) func(context.Context, uuid.UUID, json.RawMessage) (interface{}, error) {
	return func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error) {
		if s.pipelines == nil {
			return nil, ErrPipelinesUnavailable
		}
		return fn(ctx, s.pipelines, tenantID, args)
	}
}
