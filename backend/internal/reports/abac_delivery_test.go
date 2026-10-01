package reports

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestABACDelivery_PerRecipientDistinctMaskingAndIdempotency(t *testing.T) {
	orchestrator := NewDeliveryOrchestrator()

	// 1. Two recipients with different BO attribute scopes
	recipientExecutive := RecipientProfile{
		UserID:          "user-exec-1",
		Email:           "exec@fund.com",
		TenantID:        "tenant-111",
		Roles:           []string{"executive_admin"},
		PermittedFields: []string{"revenue", "aum", "salary", "pnl"},
		MaskedFields:    []string{},
	}

	recipientAdvisor := RecipientProfile{
		UserID:          "user-advisor-2",
		Email:           "advisor@fund.com",
		TenantID:        "tenant-111",
		Roles:           []string{"client_advisor"},
		PermittedFields: []string{"revenue", "aum"},
		MaskedFields:    []string{"salary", "pnl"}, // Masked sensitive fields
	}

	recipientAdvisor2 := RecipientProfile{
		UserID:          "user-advisor-3",
		Email:           "advisor2@fund.com",
		TenantID:        "tenant-111",
		Roles:           []string{"client_advisor"},
		PermittedFields: []string{"revenue", "aum"},
		MaskedFields:    []string{"salary", "pnl"}, // Identical scope as advisor 1
	}

	hashExec := ComputeABACContextHash(recipientExecutive)
	hashAdv1 := ComputeABACContextHash(recipientAdvisor)
	hashAdv2 := ComputeABACContextHash(recipientAdvisor2)

	assert.NotEqual(t, hashExec, hashAdv1, "recipients with different masking scopes must have distinct ABAC context hashes")
	assert.Equal(t, hashAdv1, hashAdv2, "recipients with identical scopes must share the same ABAC context hash")

	// 2. Build delivery plan for 3 recipients
	fireTime := time.Date(2026, 11, 30, 8, 0, 0, 0, time.UTC)
	scheduleID := "sched-daily-overview-9"
	fixedParams := map[string]interface{}{"asOfDate": "2026-11-30", "region": "US"}

	plan := BuildDeliveryPlan(scheduleID, fireTime, fixedParams, []RecipientProfile{
		recipientExecutive,
		recipientAdvisor,
		recipientAdvisor2,
	})

	require.Len(t, plan.RecipientGroups, 2, "must create exactly 2 isolated render groups across the 3 recipients")
	assert.Len(t, plan.RecipientGroups[hashExec], 1)
	assert.Len(t, plan.RecipientGroups[hashAdv1], 2)

	// 3. Execute delivery and count render invocations
	renderInvocations := make(map[string]int)
	mockRenderFn := func(ctx context.Context, abacHash string, recipients []RecipientProfile, params map[string]interface{}) ([]byte, error) {
		renderInvocations[abacHash]++
		assert.Equal(t, "US", params["region"])
		return []byte("rendered-report-bytes-for-" + abacHash), nil
	}

	result1, err := orchestrator.ExecuteDelivery(context.Background(), plan, mockRenderFn)
	require.NoError(t, err)

	assert.Equal(t, 2, result1.RendersCount)
	assert.Equal(t, 3, result1.RecipientsSent)
	assert.Equal(t, 1, renderInvocations[hashExec], "executive must receive an isolated unmasked render")
	assert.Equal(t, 1, renderInvocations[hashAdv1], "advisors must receive their specific masked render")

	// 4. Test Idempotency: retry the same scheduled fire
	result2, err := orchestrator.ExecuteDelivery(context.Background(), plan, mockRenderFn)
	require.NoError(t, err)

	assert.Equal(t, result1.IdempotencyKey, result2.IdempotencyKey)
	assert.Equal(t, 1, renderInvocations[hashExec], "retry with same idempotency key must not re-render")
	assert.Equal(t, 1, renderInvocations[hashAdv1], "retry with same idempotency key must not re-render")
}
