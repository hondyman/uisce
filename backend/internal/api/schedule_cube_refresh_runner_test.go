package api

import (
	"context"
	"errors"
	"testing"

	"github.com/hondyman/uisce/backend/internal/querybuilder"
	"github.com/hondyman/uisce/backend/internal/schedule"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
)

type stubStarter struct {
	calls []querybuilder.CubeMaterializeRequest
	err   error
	wfID  string
	plan  *querybuilder.CubeMaterializePlan
}

func (s *stubStarter) StartCubeMaterialize(_ context.Context, req querybuilder.CubeMaterializeRequest) (string, *querybuilder.CubeMaterializePlan, error) {
	s.calls = append(s.calls, req)
	if s.err != nil {
		return "", s.plan, s.err
	}
	return s.wfID, s.plan, nil
}

type stubCubeCatalog struct {
	cube *querybuilder.CubeDefinition
	list []querybuilder.CubeDefinition
}

func (s *stubCubeCatalog) GetCubeForTenant(_ context.Context, _, id string) (*querybuilder.CubeDefinition, error) {
	if s.cube == nil || s.cube.ID != id {
		return nil, errors.New("not found")
	}
	return s.cube, nil
}

func (s *stubCubeCatalog) ListCubesForTenant(_ context.Context, _, _, _ string, _ int, _ string) ([]querybuilder.CubeDefinition, string, error) {
	return s.list, "", nil
}

func TestIsAlreadyRunning(t *testing.T) {
	require.True(t, isAlreadyRunning(&serviceerror.WorkflowExecutionAlreadyStarted{}))
	require.True(t, isAlreadyRunning(errors.New("workflow execution already started")))
	require.False(t, isAlreadyRunning(errors.New("starrocks down")))
	require.False(t, isAlreadyRunning(nil))
}

func TestCubeRefreshRunner_RunStartsMaterialize(t *testing.T) {
	cube := &querybuilder.CubeDefinition{
		ID:     "c1",
		Name:   "Account Smoke",
		BOID:   "account",
		Grains: [][]string{{"account_id", "day"}, {"account_id"}},
	}
	starter := &stubStarter{
		wfID: "cube-materialize-t1-c1-v1-abc",
		plan: &querybuilder.CubeMaterializePlan{
			TenantID: "t1", CubeID: "c1", ContractVersion: 1, GrainHash: "abc", AttemptID: "att-1",
		},
	}
	r := &cubeRefreshRunner{cubes: &stubCubeCatalog{cube: cube, list: []querybuilder.CubeDefinition{*cube}}, starter: starter}

	out, err := r.Run(context.Background(), schedule.RunContext{
		TenantID: "t1", OwnerID: "u1", Ref: "c1", Params: map[string]any{},
	})
	require.NoError(t, err)
	require.Len(t, starter.calls, 2)
	require.Contains(t, out.Summary, "started=2")
	require.Equal(t, false, starter.calls[0].Force)
}

func TestCubeRefreshRunner_AlreadyRunningCounted(t *testing.T) {
	cube := &querybuilder.CubeDefinition{
		ID: "c1", Name: "X", Grains: [][]string{{"day"}},
	}
	starter := &stubStarter{
		err:  &serviceerror.WorkflowExecutionAlreadyStarted{},
		plan: &querybuilder.CubeMaterializePlan{GrainHash: "g1", ContractVersion: 1},
	}
	r := &cubeRefreshRunner{cubes: &stubCubeCatalog{cube: cube}, starter: starter}
	out, err := r.Run(context.Background(), schedule.RunContext{
		TenantID: "t1", OwnerID: "u1", Ref: "c1",
		Params: map[string]any{"force": true, "grain": []any{"day"}},
	})
	require.NoError(t, err)
	require.Contains(t, out.Summary, "already_running=1")
	require.True(t, starter.calls[0].Force)
	require.Equal(t, []string{"day"}, starter.calls[0].Grain)
}

func TestCubeRefreshRunner_KindInRegistry(t *testing.T) {
	r := &cubeRefreshRunner{}
	require.Equal(t, "cube_refresh", r.Kind())
	kinds := schedule.NewRegistry(r).Kinds()
	require.Len(t, kinds, 1)
	require.Equal(t, "Cube refresh", kinds[0].Label)
}

func TestGrainParam(t *testing.T) {
	g, ok := grainParam([]any{"account_id", "day"})
	require.True(t, ok)
	require.Equal(t, []string{"account_id", "day"}, g)
	_, ok = grainParam([]any{})
	require.False(t, ok)
}
