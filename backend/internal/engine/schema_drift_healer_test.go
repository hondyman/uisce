package engine_test

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/engine"
	"github.com/hondyman/uisce/backend/internal/governance"
	"github.com/stretchr/testify/assert"
)

func TestSchemaDriftHealer_Similarity(t *testing.T) {
	sim := engine.ComputeSimilarity("client_id", "customer_identifier")
	assert.Greater(t, sim, 0.4)

	exactSim := engine.ComputeSimilarity("client_id", "client_id")
	assert.Equal(t, 1.0, exactSim)
}

func TestSchemaDriftHealer_InterceptAndHeal(t *testing.T) {
	govSvc := governance.NewMakerCheckerService()
	healer := engine.NewSchemaDriftHealer(govSvc)

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")
	ctx = context.WithValue(ctx, "user_id", "auto-healer-bot")

	candidates := []string{"customer_identifier", "created_at", "total_amount"}
	changeReq, err := healer.InterceptAndHeal(ctx, "bo-customers", "public.customers", "client_id", candidates)

	assert.NoError(t, err)
	assert.NotNil(t, changeReq)
	assert.Equal(t, governance.StatusPendingApproval, changeReq.Status)
	assert.Contains(t, changeReq.Justification, "customer_identifier")
}
