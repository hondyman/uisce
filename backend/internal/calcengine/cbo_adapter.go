package calcengine

import (
	"context"
)

type TelemetryRouter interface {
	GetOptimalFlavor(ctx context.Context, tenantID string, boKey string, defaultFlavor string) (string, error)
}

type CBOAdapter struct {
	router TelemetryRouter
}

func NewCBOAdapter(router TelemetryRouter) *CBOAdapter {
	return &CBOAdapter{router: router}
}

func (a *CBOAdapter) GetOptimalFlavor(ctx context.Context, tenantID string, boKey string, defaultFlavor string) (string, error) {
	if a.router == nil {
		return defaultFlavor, nil
	}
	return a.router.GetOptimalFlavor(ctx, tenantID, boKey, defaultFlavor)
}

type NoOpOptimizer struct{}

func (n *NoOpOptimizer) GetOptimalFlavor(ctx context.Context, tenantID string, boKey string, defaultFlavor string) (string, error) {
	return defaultFlavor, nil
}

func NewNoOpOptimizer() *NoOpOptimizer {
	return &NoOpOptimizer{}
}


