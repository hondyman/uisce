package querybuilder

import (
	"encoding/json"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSavedQueryIdentity_Cube(t *testing.T) {
	req := &savedQueryCreateRequest{
		Name: "Positions by account",
		Subject: &boresolver.QuerySubject{
			Kind:            boresolver.QuerySubjectCube,
			CubeID:          "b3b29340-a32b-4512-ac28-5e9a7d02fef6",
			ContractVersion: boresolver.ContractVersionPin{Version: 1},
		},
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "t-acct", Alias: "account_id"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "m-qty", Alias: "qty", Aggregation: "SUM"}},
		},
	}
	kind, sourceID, binding, err := normalizeSavedQueryIdentity(req)
	require.NoError(t, err)
	assert.Equal(t, savedQuerySourceCube, kind)
	assert.Equal(t, "b3b29340-a32b-4512-ac28-5e9a7d02fef6", sourceID)
	assert.Nil(t, binding)
	require.NotNil(t, req.State.Subject)
	assert.Equal(t, boresolver.QuerySubjectCube, req.State.Subject.NormalizedKind())
	assert.Equal(t, 1, req.State.Subject.ContractVersion.Version)
}

func TestNormalizeSavedQueryIdentity_CubeRequiresCubeID(t *testing.T) {
	req := &savedQueryCreateRequest{
		Name:    "bad",
		Subject: &boresolver.QuerySubject{Kind: boresolver.QuerySubjectCube},
	}
	_, _, _, err := normalizeSavedQueryIdentity(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cubeId")
}

func TestNormalizeSavedQueryIdentity_BOStillRequiresBOID(t *testing.T) {
	req := &savedQueryCreateRequest{Name: "x"}
	_, _, _, err := normalizeSavedQueryIdentity(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boId")
}

func TestSavedQueryDef_CubeSubjectRoutePinnedFields(t *testing.T) {
	sq := SavedQuery{
		SourceKind: savedQuerySourceCube,
		BOID:       "cube-1",
		BindingID:  "should-clear",
		State: SavedQueryState{
			Subject: &boresolver.QuerySubject{
				Kind:            boresolver.QuerySubjectCube,
				CubeID:          "cube-1",
				ContractVersion: boresolver.ContractVersionPin{Version: 3},
			},
			Dimensions: []SavedQueryDimension{{TermNodeID: "dim-a", Alias: "a"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "met-b", Alias: "b", Aggregation: "SUM"}},
		},
	}
	qd := savedQueryDef(sq, "tenant-1", nil, 50)
	require.NotNil(t, qd.Context.Subject)
	assert.Equal(t, boresolver.QuerySubjectCube, qd.Context.Subject.NormalizedKind())
	assert.Equal(t, "cube-1", qd.Context.Subject.CubeID)
	assert.Equal(t, 3, qd.Context.Subject.ContractVersion.Version)
	assert.Empty(t, qd.Context.BOID)
	assert.Empty(t, qd.Context.BindingID)
	assert.Equal(t, 50, qd.Query.Limit)
	require.Len(t, qd.Query.Dimensions, 1)
	assert.Equal(t, "dim-a", qd.Query.Dimensions[0].TermNodeID)
}

func TestSavedQueryDef_LegacyBOSynthesizesSubject(t *testing.T) {
	sq := SavedQuery{
		SourceKind: savedQuerySourceBusinessObject,
		BOID:       "bo-1",
		BindingID:  "bind-1",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "t1", Alias: "x"}},
		},
	}
	qd := savedQueryDef(sq, "tenant-1", nil, 10)
	require.NotNil(t, qd.Context.Subject)
	assert.Equal(t, boresolver.QuerySubjectBusinessObject, qd.Context.Subject.NormalizedKind())
	assert.Equal(t, "bo-1", qd.Context.BOID)
	assert.Equal(t, "bo-1", qd.Context.Subject.BOID)
}

func TestSavedQueryRow_ToSavedQuery_RoundTripSubject(t *testing.T) {
	state := SavedQueryState{
		Subject: &boresolver.QuerySubject{
			Kind:            boresolver.QuerySubjectCube,
			CubeID:          "cube-99",
			ContractVersion: boresolver.ContractVersionPin{Latest: true},
		},
		Dimensions: []SavedQueryDimension{{TermNodeID: "d", Alias: "d"}},
	}
	raw, err := json.Marshal(state)
	require.NoError(t, err)
	row := savedQueryRow{
		ID:         "sq-1",
		TenantID:   "t1",
		UserID:     "u1",
		Name:       "cube q",
		SourceKind: savedQuerySourceCube,
		SourceID:   "cube-99",
		QueryState: raw,
		ChartType:  "table",
		Visibility: "private",
		Status:     "active",
	}
	sq := row.toSavedQuery()
	assert.Equal(t, savedQuerySourceCube, sq.SourceKind)
	assert.Equal(t, "cube-99", sq.BOID)
	require.NotNil(t, sq.Subject)
	assert.Equal(t, "cube-99", sq.Subject.CubeID)
	assert.True(t, sq.Subject.ContractVersion.Latest)
}
