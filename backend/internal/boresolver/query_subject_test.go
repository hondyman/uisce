package boresolver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractVersionPin_Unmarshal(t *testing.T) {
	t.Run("latest string", func(t *testing.T) {
		var p ContractVersionPin
		require.NoError(t, json.Unmarshal([]byte(`"latest"`), &p))
		assert.True(t, p.Latest)
		assert.Equal(t, 0, p.Version)
	})
	t.Run("positive number", func(t *testing.T) {
		var p ContractVersionPin
		require.NoError(t, json.Unmarshal([]byte(`3`), &p))
		assert.False(t, p.Latest)
		assert.Equal(t, 3, p.Version)
	})
	t.Run("zero means latest", func(t *testing.T) {
		var p ContractVersionPin
		require.NoError(t, json.Unmarshal([]byte(`0`), &p))
		assert.True(t, p.Latest)
	})
}

func TestQuerySubject_RoundTrip(t *testing.T) {
	raw := []byte(`{
		"kind":"cube",
		"cubeId":"c1",
		"contractVersion":"latest"
	}`)
	var s QuerySubject
	require.NoError(t, json.Unmarshal(raw, &s))
	assert.Equal(t, QuerySubjectCube, s.NormalizedKind())
	assert.Equal(t, "c1", s.CubeID)
	assert.True(t, s.ContractVersion.Latest)

	out, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"contractVersion":"latest"`)
}

func TestQueryContext_SubjectOptional(t *testing.T) {
	raw := []byte(`{"boId":"bo1","bindingId":"b1","tenantId":"t1"}`)
	var ctx QueryContext
	require.NoError(t, json.Unmarshal(raw, &ctx))
	assert.Nil(t, ctx.Subject)
	assert.Equal(t, "bo1", ctx.BOID)
}
