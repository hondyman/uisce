package audit

import (
	"context"
	"encoding/json"
	"testing"
)

// TestSeverityRules is the cardinal-rule-7 compliance contract. If a
// future change ever inverts any of these, that's a regulatory violation
// and the SIEM alerting will silently degrade.
func TestSeverityRules(t *testing.T) {
	cases := []struct {
		internal string
		want     AuditSeverity
	}{
		{EventTypeAIABACDenied, SeverityCritical},
		{EventTypeCatalogBOMutated, SeverityWarn},
		{EventTypeAIQueryGenerated, SeverityInfo},
		{EventTypeAIColumnMasked, SeverityInfo},
		{EventTypeAIABACEvaluated, SeverityInfo},
		{EventTypeAISemanticResolved, SeverityInfo},
		{EventTypeAILineageResolved, SeverityInfo},
	}
	for _, c := range cases {
		got := SeverityFor(c.internal)
		if got != c.want {
			t.Errorf("SeverityFor(%q): got %q, want %q", c.internal, got, c.want)
		}
	}
}

func TestECSNamesAreLowercaseDotSeparated(t *testing.T) {
	// SIEM tooling assumes this convention.
	cases := map[string]string{
		EventTypeAIQueryGenerated:   "ai.query.generated",
		EventTypeAIABACDenied:       "ai.abac.denied",
		EventTypeCatalogBOMutated:   "catalog.bo.mutated",
		EventTypeAIColumnMasked:     "ai.column.masked",
		EventTypeAIABACEvaluated:    "ai.abac.evaluated",
		EventTypeAISemanticResolved: "ai.semantic.resolved",
		EventTypeAILineageResolved:  "ai.lineage.resolved",
	}
	for internal, want := range cases {
		got := ECSNameFor(internal)
		if got != want {
			t.Errorf("ECSNameFor(%q): got %q, want %q", internal, got, want)
		}
	}
}

func TestExtractActor_returns_empty_when_ctx_unpopulated(t *testing.T) {
	a := ExtractActor(context.Background())
	if a.TenantID != "" || a.UserEmail != "" || a.FunctionalRole != "" || a.ClientID != "" {
		t.Errorf("expected zero actor, got %+v", a)
	}
	a = ExtractActor(nil) //nolint:staticcheck // intentional nil test
	if a.TenantID != "" {
		t.Errorf("nil ctx should yield zero actor, got %+v", a)
	}
}

func TestWithActor_and_ExtractActor_roundtrip(t *testing.T) {
	ctx := WithActor(context.Background(),
		"tenant-uuid",
		"alice@acme.example",
		"analyst",
		"keycloak-client-azp",
	)
	got := ExtractActor(ctx)
	if got.TenantID != "tenant-uuid" {
		t.Errorf("tenant: got %q", got.TenantID)
	}
	if got.UserEmail != "alice@acme.example" {
		t.Errorf("email: got %q", got.UserEmail)
	}
	if got.FunctionalRole != "analyst" {
		t.Errorf("role: got %q", got.FunctionalRole)
	}
	if got.ClientID != "keycloak-client-azp" {
		t.Errorf("azp: got %q", got.ClientID)
	}
}

func TestWithActor_ignores_empty_fields(t *testing.T) {
	// AuthEnrichmentMiddleware typically sets fields as they're parsed;
	// we must not store zero values that would shadow real claims later.
	ctx := WithActor(context.Background(), "", "", "analyst", "")
	a := ExtractActor(ctx)
	if a.TenantID != "" {
		t.Errorf("empty tenant should not be stored")
	}
	if a.FunctionalRole != "analyst" {
		t.Errorf("non-empty role lost: %q", a.FunctionalRole)
	}
}

func TestWrapInEnvelope_produces_ECS_shape(t *testing.T) {
	ctx := WithActor(context.Background(), "tenant-1", "u@x", "analyst", "azp-1")
	payload := AIABACDeniedEvent{
		QueryID:        "q-1",
		TenantID:       "tenant-1",
		UserID:         "user-1",
		DeniedResource: "x.y",
		DenialReason:   "test",
	}
	env := WrapInEnvelope(ctx, EventTypeAIABACDenied, payload)
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	meta, ok := decoded["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing metadata: %s", raw)
	}
	if meta["event_type"] != "ai.abac.denied" {
		t.Errorf("event_type: got %v want ai.abac.denied", meta["event_type"])
	}
	if meta["severity"] != "CRITICAL" {
		t.Errorf("severity: got %v want CRITICAL", meta["severity"])
	}
	if meta["timestamp"] == nil {
		t.Errorf("missing timestamp: %s", raw)
	}

	actor, ok := decoded["actor"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing actor: %s", raw)
	}
	if actor["tenant_id"] != "tenant-1" {
		t.Errorf("actor.tenant_id: got %v", actor["tenant_id"])
	}
	if actor["user_email"] != "u@x" {
		t.Errorf("actor.user_email: got %v", actor["user_email"])
	}
	if actor["functional_role"] != "analyst" {
		t.Errorf("actor.functional_role: got %v", actor["functional_role"])
	}
	if actor["client_id"] != "azp-1" {
		t.Errorf("actor.client_id: got %v", actor["client_id"])
	}

	body, ok := decoded["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing payload: %s", raw)
	}
	if body["queryId"] != "q-1" {
		t.Errorf("payload.queryId: got %v", body["queryId"])
	}
	if body["deniedResource"] != "x.y" {
		t.Errorf("payload.deniedResource: got %v", body["deniedResource"])
	}
}

func TestMarshalEnvelopeJSON_emits_raw_message(t *testing.T) {
	ctx := WithActor(context.Background(), "t", "u", "r", "c")
	raw, err := MarshalEnvelopeJSON(ctx, EventTypeAIQueryGenerated,
		AIQueryGeneratedEvent{QueryID: "q", TenantID: "t"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("empty envelope")
	}
	// Should contain an embedded envelope, not a flat event.
	if !contains(raw, `"event_type"`) {
		t.Errorf("envelope missing event_type field: %s", raw)
	}
}

func contains(haystack []byte, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}
