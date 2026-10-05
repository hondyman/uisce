package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
)

func TestCubeMaterializeWorkflowID_Stable(t *testing.T) {
	id := CubeMaterializeWorkflowID(
		"99e99e99-99e9-49e9-89e9-99e99e99e999",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		1,
		GrainHash([]string{"status", "account_id"}),
	)
	assert.Equal(t,
		"cube-materialize-99e99e99-99e9-49e9-89e9-99e99e99e999-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa-v1-"+GrainHash([]string{"account_id", "status"}),
		id,
	)
}

func TestCubeMaterializeStartOptions_RejectDuplicate(t *testing.T) {
	opts := CubeMaterializeStartOptions("t1", "c1", 2, "deadbeef")
	assert.Equal(t, CubeTaskQueue, opts.TaskQueue)
	assert.Equal(t, "cube-materialize-t1-c1-v2-deadbeef", opts.ID)
	assert.Equal(t, enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE, opts.WorkflowIDReusePolicy)
	assert.True(t, opts.WorkflowExecutionErrorWhenAlreadyStarted)
}

func TestQualifyCubeMVName(t *testing.T) {
	ddl := "CREATE MATERIALIZED VIEW cube_t_account_x\nAS SELECT 1;"
	got := qualifyCubeMVName(ddl, "cube_t_account_x", "`tenant_t`.`cube_t_account_x`")
	require.Contains(t, got, "CREATE MATERIALIZED VIEW `tenant_t`.`cube_t_account_x`")
	require.NotContains(t, got, "CREATE MATERIALIZED VIEW cube_t_account_x\n")
}
