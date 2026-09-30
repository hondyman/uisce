package mdm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoldenPublishingActivity_PublishGoldenRecordsActivity(t *testing.T) {
	mockPub := &InMemoryEventPublisher{}
	activity := NewGoldenPublishingActivity(mockPub)
	ctx := context.Background()

	tenantID := "11111111-1111-1111-1111-111111111111"
	batchID := uuid.New()

	records := []GoldenPublishRecord{
		{
			EntityKey: "ACC-001",
			Payload: map[string]interface{}{
				"account_number": "ACC-001",
				"account_name":   "Apex Prime Custody",
				"status":         "ACTIVE",
			},
			Provenance: map[string]interface{}{
				"account_name": map[string]interface{}{
					"winning_source": "CUSTODIAN_A",
					"confidence":     0.95,
				},
			},
		},
		{
			EntityKey: "ACC-002",
			Payload: map[string]interface{}{
				"account_number": "ACC-002",
				"account_name":   "Beacon Wealth SMA",
				"status":         "PENDING",
			},
		},
	}

	t.Run("EngineModeNew emits events to topic", func(t *testing.T) {
		req := PublishGoldenRecordsRequest{
			TenantID:   tenantID,
			EntityType: "ACCOUNT",
			BatchID:    batchID,
			Topic:      "oms.account.gold",
			EngineMode: EngineModeNew,
			Records:    records,
		}

		res, err := activity.PublishGoldenRecordsActivity(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.False(t, res.Gated)
		assert.Equal(t, 2, res.TotalPublished)
		assert.Equal(t, "oms.account.gold", res.Topic)
		require.Len(t, res.EventIDs, 2)
		assert.Contains(t, res.EventIDs[0], "gold.11111111-1111-1111-1111-111111111111.ACCOUNT.ACC-001.")
		assert.Contains(t, res.EventIDs[1], "gold.11111111-1111-1111-1111-111111111111.ACCOUNT.ACC-002.")

		// Verify events emitted to publisher
		require.Len(t, mockPub.Events, 2)
		assert.Equal(t, batchID.String(), mockPub.Events[0].BatchID)
		assert.Equal(t, "ACC-001", mockPub.Events[0].EntityKey)
		assert.Equal(t, "Apex Prime Custody", mockPub.Events[0].Payload["account_name"])
	})

	t.Run("EngineModeShadow gates live emission and returns review artifact", func(t *testing.T) {
		shadowPub := &InMemoryEventPublisher{}
		shadowActivity := NewGoldenPublishingActivity(shadowPub)

		req := PublishGoldenRecordsRequest{
			TenantID:   tenantID,
			EntityType: "ACCOUNT",
			BatchID:    batchID,
			Topic:      "oms.account.gold",
			EngineMode: EngineModeShadow,
			Records:    records,
		}

		res, err := shadowActivity.PublishGoldenRecordsActivity(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.True(t, res.Gated)
		assert.Equal(t, 0, res.TotalPublished)
		assert.Contains(t, res.GatedReason, "shadow")
		require.Len(t, res.EventIDs, 2, "Event IDs should still be calculated for review artifact")
		assert.Empty(t, shadowPub.Events, "Zero events emitted to broker in shadow mode")
	})

	// Assert deterministic deduplication: Same payload produces identical event ID
	hash1, _ := ComputePayloadHash(records[0].Payload)
	hash2, _ := ComputePayloadHash(records[0].Payload)
	assert.Equal(t, hash1, hash2, "Identical payloads must yield identical hash components")
}
