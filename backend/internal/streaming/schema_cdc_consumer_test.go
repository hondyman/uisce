package streaming_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/streaming"
)

func TestProcessSchemaEvent_Debounce(t *testing.T) {
	rewarmer := streaming.NewTenantRewarmerAdapter(nil)
	repo := streaming.NewRuleRepoAdapter(nil)

	consumer := streaming.NewSchemaCDCConsumer(rewarmer, repo, 1)

	evt := streaming.SchemaChangeEvent{
		TenantID:  "99e99e99-99e9-49e9-89e9-99e99e99e999",
		BOID:      "BO-PORTFOLIO",
		EventType: "ADD_COLUMN",
		Table:     "ibor_position",
	}

	payload, _ := json.Marshal(evt)

	for i := 0; i < 3; i++ {
		err := consumer.ProcessSchemaEvent(context.Background(), payload)
		if err != nil {
			t.Fatalf("unexpected error processing event: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1200 * time.Millisecond)
}

func TestProcessSchemaEvent_IgnoresOtherEventTypes(t *testing.T) {
	rewarmer := streaming.NewTenantRewarmerAdapter(nil)
	repo := streaming.NewRuleRepoAdapter(nil)

	consumer := streaming.NewSchemaCDCConsumer(rewarmer, repo, 1)

	evt := streaming.SchemaChangeEvent{
		TenantID:  "99e99e99-99e9-49e9-89e9-99e99e99e999",
		BOID:      "BO-PORTFOLIO",
		EventType: "DROP_TABLE",
		Table:     "ibor_position",
	}

	payload, _ := json.Marshal(evt)

	err := consumer.ProcessSchemaEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error processing event: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestProcessSchemaEvent_InvalidTenantID(t *testing.T) {
	rewarmer := streaming.NewTenantRewarmerAdapter(nil)
	repo := streaming.NewRuleRepoAdapter(nil)

	consumer := streaming.NewSchemaCDCConsumer(rewarmer, repo, 1)

	evt := streaming.SchemaChangeEvent{
		TenantID:  "not-a-valid-uuid",
		BOID:      "BO-PORTFOLIO",
		EventType: "ADD_COLUMN",
		Table:     "ibor_position",
	}

	payload, _ := json.Marshal(evt)

	err := consumer.ProcessSchemaEvent(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for invalid tenant ID")
	}
}
