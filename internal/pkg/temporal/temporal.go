package temporal

import (
	"go.temporal.io/sdk/client"

	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
)

// NewClientWithRetry is a thin wrapper that delegates to the canonical
// implementation in libs/temporal-client. This preserves existing imports
// to "internal/pkg/temporal" while centralizing behavior in the libs module.
func NewClientWithRetry() (client.Client, error) {
	return temporalclient.NewClientWithRetry()
}
