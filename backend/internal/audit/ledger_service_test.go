package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordLedgerEntry_HashChain(t *testing.T) {
	svc := NewLedgerService(nil)

	entry1, err := svc.RecordLedgerEntry(context.Background(), LedgerEntry{
		TenantID:        "core",
		EntityType:      "BusinessObject",
		EntityID:        "bo_customer",
		ActionType:      "UPDATE",
		PayloadSnapshot: map[string]interface{}{"field": "balance", "new_val": 5000000},
	})
	assert.NoError(t, err)
	assert.Equal(t, "GENESIS_BLOCK", entry1.PreviousHash)
	assert.NotEmpty(t, entry1.CurrentHash)

	entry2, err := svc.RecordLedgerEntry(context.Background(), LedgerEntry{
		TenantID:        "core",
		EntityType:      "BusinessObject",
		EntityID:        "bo_customer",
		ActionType:      "SCHEMA_CHANGE",
		PayloadSnapshot: map[string]interface{}{"added_attribute": "esg_rating"},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, entry2.CurrentHash)
}
