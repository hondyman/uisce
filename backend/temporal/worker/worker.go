package worker

import (
	"github.com/hondyman/semlayer/backend/internal/workflows"
	temporalclient "go.temporal.io/sdk/client"
	tworker "go.temporal.io/sdk/worker"
)

// Start starts a Temporal worker that registers the test workflow and activity.
// It blocks until the worker stops or an interrupt is received.
func Start(c temporalclient.Client) error {
	w := tworker.New(c, "e2e_test_queue", tworker.Options{})
	w.RegisterWorkflow(workflows.TestWorkflow)
	w.RegisterActivity(workflows.PublishEventActivity)
	return w.Run(tworker.InterruptCh())
}
