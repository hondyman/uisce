package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/querybuilder"
	"github.com/hondyman/uisce/backend/internal/schedule"
)

// cubeMaterializeStarter starts CubeMaterializeWorkflow (Server implements this).
type cubeMaterializeStarter interface {
	StartCubeMaterialize(ctx context.Context, req querybuilder.CubeMaterializeRequest) (string, *querybuilder.CubeMaterializePlan, error)
}

// cubeCatalog looks up cubes for the cube_refresh schedule runner.
type cubeCatalog interface {
	GetCubeForTenant(ctx context.Context, tenantID, id string) (*querybuilder.CubeDefinition, error)
	ListCubesForTenant(ctx context.Context, tenantID, scope, boID string, limit int, cursor string) ([]querybuilder.CubeDefinition, string, error)
}

// cubeRefreshRunner schedules cube grain materialize/refresh (kind cube_refresh).
// Ref is cube_definition.id. Optional params: grain ([]string), force (bool).
type cubeRefreshRunner struct {
	cubes   cubeCatalog
	starter cubeMaterializeStarter
}

func (r *cubeRefreshRunner) Kind() string  { return "cube_refresh" }
func (r *cubeRefreshRunner) Label() string { return "Cube refresh" }

func (r *cubeRefreshRunner) Check(ctx context.Context, tenantID, _ string, ref string, _ map[string]any) error {
	if r.cubes == nil {
		return fmt.Errorf("cube refresh runner: cubes handler not configured")
	}
	if strings.TrimSpace(ref) == "" {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	_, err := r.cubes.GetCubeForTenant(ctx, tenantID, ref)
	if err != nil {
		return schedule.MsgTargetNotFound(r.Label(), ref)
	}
	return nil
}

func (r *cubeRefreshRunner) Targets(ctx context.Context, tenantID, _ string) ([]schedule.TargetInfo, error) {
	if r.cubes == nil {
		return nil, fmt.Errorf("cube refresh runner: cubes handler not configured")
	}
	items, _, err := r.cubes.ListCubesForTenant(ctx, tenantID, "all", "", 200, "")
	if err != nil {
		return nil, err
	}
	out := make([]schedule.TargetInfo, 0, len(items))
	for _, c := range items {
		out = append(out, schedule.TargetInfo{
			Ref:         c.ID,
			Name:        c.Name,
			Description: fmt.Sprintf("v%d %s", c.ContractVersion, c.BOID),
		})
	}
	return out, nil
}

func (r *cubeRefreshRunner) Run(ctx context.Context, rc schedule.RunContext) (*schedule.Outcome, error) {
	if r.starter == nil {
		return nil, fmt.Errorf("cube refresh runner: materialize starter not configured")
	}
	if err := r.Check(ctx, rc.TenantID, rc.OwnerID, rc.Ref, rc.Params); err != nil {
		return nil, err
	}
	cube, err := r.cubes.GetCubeForTenant(ctx, rc.TenantID, rc.Ref)
	if err != nil {
		return nil, err
	}

	force := false
	if v, ok := rc.Params["force"].(bool); ok {
		force = v
	}
	grains := cube.Grains
	if g, ok := grainParam(rc.Params["grain"]); ok {
		grains = [][]string{g}
	}
	if len(grains) == 0 {
		return nil, fmt.Errorf("cube %s has no grains", rc.Ref)
	}

	started := 0
	already := 0
	refs := map[string]any{}
	var workflowIDs []string
	for _, grain := range grains {
		if rc.Heartbeat != nil {
			rc.Heartbeat(map[string]any{"grain": grain})
		}
		wfID, plan, startErr := r.starter.StartCubeMaterialize(ctx, querybuilder.CubeMaterializeRequest{
			TenantID: rc.TenantID,
			CubeID:   rc.Ref,
			Grain:    grain,
			Force:    force,
		})
		if startErr != nil {
			if isAlreadyRunning(startErr) {
				already++
				continue
			}
			return nil, startErr
		}
		started++
		if wfID != "" {
			workflowIDs = append(workflowIDs, wfID)
		}
		if plan != nil && plan.Noop {
			refs["noop_"+plan.GrainHash] = plan.NoopReason
		}
	}
	refs["workflow_ids"] = workflowIDs
	refs["already_running"] = already
	return &schedule.Outcome{
		Summary: fmt.Sprintf("started=%d already_running=%d grains=%d", started, already, len(grains)),
		Refs:    refs,
	}, nil
}

func grainParam(v any) ([]string, bool) {
	switch x := v.(type) {
	case []string:
		if len(x) == 0 {
			return nil, false
		}
		return x, true
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s, ok := e.(string)
			if !ok || strings.TrimSpace(s) == "" {
				continue
			}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil, false
		}
		return out, true
	default:
		return nil, false
	}
}
