package querybuilder

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failColdWriter struct{}

func (failColdWriter) ApplyCold(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) (*CubeMaterializeColdResult, error) {
	return nil, errors.New("forced cold failure")
}

func TestCompleteDualCommit_RequiresBothTiers(t *testing.T) {
	m := NewCubeMaterializer(nil, nil)
	plan := &CubeMaterializePlan{NodeID: "11111111-1111-1111-1111-111111111111", AttemptID: "a1"}
	hot := &CubeMaterializeHotResult{AppliedDDL: true, RowCount: 1}

	err := m.CompleteDualCommit(context.Background(), plan, hot, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dual-commit requires successful hot and cold")

	err = m.CompleteDualCommit(context.Background(), plan, hot, &CubeMaterializeColdResult{Applied: false})
	require.Error(t, err)
}

func TestSetColdWriter_InjectedFailure(t *testing.T) {
	m := NewCubeMaterializer(nil, nil)
	m.SetColdWriter(failColdWriter{})
	_, err := m.ApplyCold(context.Background(), &CubeMaterializePlan{
		MaterializationName: "cube_x",
		TargetDatabase:      "tenant_t",
		IcebergCatalog:      "iceberg_catalog",
		IcebergDatabase:     "cubes",
	}, &CubeMaterializeHotResult{AppliedDDL: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forced cold failure")
}

func TestDefaultIcebergCatalog(t *testing.T) {
	t.Setenv("CUBE_ICEBERG_CATALOG", "")
	assert.Equal(t, "iceberg_catalog", defaultIcebergCatalog())
	t.Setenv("CUBE_ICEBERG_CATALOG", "my_iceberg")
	assert.Equal(t, "my_iceberg", defaultIcebergCatalog())
}
