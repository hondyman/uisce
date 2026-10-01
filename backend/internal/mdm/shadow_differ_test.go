package mdm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareGoldenRecords(t *testing.T) {
	t.Run("clean parity between legacy and new engine", func(t *testing.T) {
		legacy := map[string]map[string]any{
			"ACC-101": {
				"account_name": "Apex Global Fund",
				"status":       "ACTIVE",
				"balance":      500000.0,
			},
		}
		newRecs := map[string]map[string]any{
			"ACC-101": {
				"account_name": "Apex Global Fund",
				"status":       "ACTIVE",
				"balance":      500000.0,
			},
		}

		summary := CompareGoldenRecords(legacy, newRecs)
		assert.True(t, summary.IsParityClean)
		assert.Equal(t, 1, summary.TotalEntities)
		assert.Equal(t, 1, summary.MatchedEntities)
		assert.Equal(t, 0, summary.TotalDiffs)
		assert.Empty(t, summary.AttributeDiffs)
	})

	t.Run("detects attribute divergence", func(t *testing.T) {
		legacy := map[string]map[string]any{
			"ACC-101": {
				"account_name": "Apex Global Fund",
				"status":       "ACTIVE",
			},
		}
		newRecs := map[string]map[string]any{
			"ACC-101": {
				"account_name": "Apex Global Legacy",
				"status":       "ACTIVE",
			},
		}

		summary := CompareGoldenRecords(legacy, newRecs)
		assert.False(t, summary.IsParityClean)
		assert.Equal(t, 1, summary.TotalEntities)
		assert.Equal(t, 0, summary.MatchedEntities)
		assert.Equal(t, 1, summary.TotalDiffs)
		require.Len(t, summary.AttributeDiffs, 1)
		assert.Equal(t, "ACC-101", summary.AttributeDiffs[0].EntityKey)
		assert.Equal(t, "account_name", summary.AttributeDiffs[0].AttributeName)
		assert.Equal(t, "Apex Global Fund", summary.AttributeDiffs[0].LegacyValue)
		assert.Equal(t, "Apex Global Legacy", summary.AttributeDiffs[0].NewValue)
	})
}
