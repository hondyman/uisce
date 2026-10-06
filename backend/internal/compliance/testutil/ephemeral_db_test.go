package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEphemeralTestDB_CreationAndIsolation(t *testing.T) {
	db := GetEphemeralTestDB(t)
	require.NotNil(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ruleCount int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM compliance.compliance_rule").Scan(&ruleCount)
	require.NoError(t, err)
	require.Equal(t, 50, ruleCount)

	// Mutate something inside ephemeral DB
	_, err = db.ExecContext(ctx, "DELETE FROM compliance.compliance_ruleset_membership")
	require.NoError(t, err)

	var rulesetCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM compliance.compliance_ruleset_membership").Scan(&rulesetCount)
	require.NoError(t, err)
	require.Equal(t, 0, rulesetCount)
}
