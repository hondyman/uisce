package querybuilder

import (
	"fmt"
	"strings"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// CubeTaskQueue is the Temporal task queue for CubeMaterializeWorkflow.
// Schedules (CUBE-1.5) and Designer Deploy/Refresh target this queue.
const CubeTaskQueue = "uisce-cubes"

// CubeMaterializeWorkflowName is the registered Temporal workflow type name.
const CubeMaterializeWorkflowName = "CubeMaterializeWorkflow"

// CubeMaterializeWorkflowID builds the single-flight workflow ID for one grain
// of one cube contract version. Temporal REJECT_DUPLICATE refuses a second run
// while this ID is open (CUBE-1.2 review fix #4).
func CubeMaterializeWorkflowID(tenantID, cubeID string, contractVersion int, grainHash string) string {
	return fmt.Sprintf("cube-materialize-%s-%s-v%d-%s",
		strings.TrimSpace(tenantID),
		strings.TrimSpace(cubeID),
		contractVersion,
		strings.TrimSpace(grainHash),
	)
}

// CubeMaterializeStartOptions returns StartWorkflowOptions with REJECT_DUPLICATE
// so concurrent Deploy/Refresh for the same grain is refused rather than queued.
func CubeMaterializeStartOptions(tenantID, cubeID string, contractVersion int, grainHash string) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:                                       CubeMaterializeWorkflowID(tenantID, cubeID, contractVersion, grainHash),
		TaskQueue:                                CubeTaskQueue,
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}
