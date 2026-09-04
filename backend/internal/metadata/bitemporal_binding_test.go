package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildBiTemporalWhereClause(t *testing.T) {
	config := BiTemporalConfig{
		ValidStartCol:       "effective_from",
		ValidEndCol:         "effective_to",
		TransactionStartCol: "sys_start_time",
		TransactionEndCol:   "sys_end_time",
	}

	whereClause := BuildBiTemporalWhereClause(config, "2025-12-31 00:00:00", "2026-01-01 00:00:00")
	assert.Contains(t, whereClause, "effective_from")
	assert.Contains(t, whereClause, "sys_start_time")
	assert.Contains(t, whereClause, "AND")
}

