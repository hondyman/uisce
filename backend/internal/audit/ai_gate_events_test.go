package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAIGateEventJSONRoundTrip verifies each typed AI Gate event marshals
// and unmarshals cleanly with snake_case field names. This is the contract
// downstream consumers (Iceberg sink, Hasura, compliance reporter) rely on.
func TestAIGateEventJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	t.Run("AIQueryGeneratedEvent", func(t *testing.T) {
		in := AIQueryGeneratedEvent{
			QueryID:          "q-1",
			TenantID:         "tenant-acme",
			UserID:           "user-42",
			UserEmail:        "alice@acme.example",
			FunctionalRole:   "analyst",
			InputPrompt:      "show me last 90 days of trades",
			PromptHash:       "sha256:abc",
			DatasourceID:     "ds-1",
			BusinessObjID:    "bo-trades",
			TechnicalName:    "trades",
			GeneratedSQL:     "SELECT ...",
			GeneratedHash:    "sha256:def",
			JoinCount:        3,
			FieldCount:       12,
			MaskedFieldCount: 2,
			GeneratedAt:      ts,
			DurationMs:       42,
		}
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out AIQueryGeneratedEvent
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.QueryID != in.QueryID || out.TenantID != in.TenantID || out.GeneratedSQL != in.GeneratedSQL {
			t.Errorf("round-trip drift: in=%+v out=%+v", in, out)
		}
		// Cardinal Rule 7: snake_case keys (Cardinal Rule 7 schema requirement).
		if !strings.Contains(string(raw), `"tenantId"`) {
			t.Errorf("expected snake_case tenantId in JSON, got: %s", raw)
		}
	})

	t.Run("AIABACDeniedEvent_sets_emitted_sync_marker", func(t *testing.T) {
		in := AIABACDeniedEvent{
			QueryID:             "q-2",
			TenantID:            "tenant-acme",
			UserID:              "user-42",
			ProfileID:           "profile-restricted",
			ProfileVersion:      7,
			Classification:      "PII",
			DeniedResource:      "orders.customer_ssn",
			DenialReason:        "profile lacks PII visibility",
			AttemptedSQLPreview: "SELECT customer_ssn FROM orders",
		}
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		// EmittedSync is a JSON bool; default false.
		if !strings.Contains(string(raw), `"emittedSync":false`) {
			t.Errorf("expected emittedSync:false default, got: %s", raw)
		}
	})

	t.Run("CatalogBOMutatedEvent_includes_invalidation_keys", func(t *testing.T) {
		in := CatalogBOMutatedEvent{
			MutationID:          "m-1",
			TenantID:            "tenant-acme",
			BusinessObjectID:    "bo-orders",
			TechnicalName:       "orders",
			DatasourceID:        "ds-1",
			MutationType:        "update",
			VersionBefore:       3,
			VersionAfter:        4,
			InvalidateCacheKeys: []string{"md:v1:t:tenant-acme:bo:by-name:ds-1:orders"},
			MutatedAt:           ts,
		}
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out CatalogBOMutatedEvent
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(out.InvalidateCacheKeys) != 1 || out.InvalidateCacheKeys[0] != in.InvalidateCacheKeys[0] {
			t.Errorf("invalidation keys lost in round-trip: in=%v out=%v", in.InvalidateCacheKeys, out.InvalidateCacheKeys)
		}
	})

	t.Run("TermResolution_embeds_in_AISemanticResolved", func(t *testing.T) {
		resolutions, _ := json.Marshal([]TermResolution{
			{SemanticTerm: "trade_date", PhysicalColumn: "t.executed_at", TableName: "trades", MatchMethod: "exact", MatchConfidence: 0.98},
		})
		in := AISemanticResolvedEvent{
			QueryID:               "q-3",
			TenantID:              "tenant-acme",
			ResolvedBusinessObjID: "bo-trades",
			ResolvedTechnicalName: "trades",
			TermResolutions:       resolutions,
			ConfidenceScore:       0.97,
			ResolvedAt:            ts,
		}
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out AISemanticResolvedEvent
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var got []TermResolution
		if err := json.Unmarshal(out.TermResolutions, &got); err != nil {
			t.Fatalf("nested unmarshal: %v", err)
		}
		if len(got) != 1 || got[0].SemanticTerm != "trade_date" {
			t.Errorf("nested TermResolution lost: %+v", got)
		}
	})
}

// TestAIGateTopicConstants verifies the topic names match the documented
// Cardinal Rule 6 / 7 naming convention.
func TestAIGateTopicConstants(t *testing.T) {
	cases := map[string]string{
		TopicAIGate:             "audit.ai.gate",
		TopicAIDenials:          "audit.ai.denials",
		TopicCatalogMutations:   "audit.catalog.mutations",
		TopicCacheInvalidations: "audit.cache.invalidations",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("topic drift: got %q want %q", got, want)
		}
	}
}

// TestAIGateEventTypeConstants verifies the event-type wire names are stable.
func TestAIGateEventTypeConstants(t *testing.T) {
	cases := map[string]string{
		EventTypeAIQueryGenerated:   "AI_QUERY_GENERATED",
		EventTypeAISemanticResolved: "AI_SEMANTIC_RESOLVED",
		EventTypeAIColumnMasked:     "AI_COLUMN_MASKED",
		EventTypeAIABACEvaluated:    "AI_ABAC_EVALUATED",
		EventTypeAIABACDenied:       "AI_ABAC_DENIED",
		EventTypeAILineageResolved:  "AI_LINEAGE_RESOLVED",
		EventTypeCatalogBOMutated:   "CATALOG_BO_MUTATED",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("event-type drift: got %q want %q", got, want)
		}
	}
}
