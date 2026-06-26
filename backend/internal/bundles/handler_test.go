package bundles

import (
	"encoding/json"
	"testing"
)

func TestValidateGuardrailData_NegativeSOD(t *testing.T) {
	// missing pairs
	var bad json.RawMessage = json.RawMessage(`{"pairs": []}`)
	if err := validateGuardrailData("sod", bad); err == nil {
		t.Fatalf("expected error for empty pairs")
	}

	// pair with empty string
	bad = json.RawMessage(`{"pairs": [["", "b"]]}`)
	if err := validateGuardrailData("sod", bad); err == nil {
		t.Fatalf("expected error for empty pair item")
	}
}

func TestValidateGuardrailData_NegativeCertified(t *testing.T) {
	// missing claims
	var bad json.RawMessage = json.RawMessage(`{"claims": []}`)
	if err := validateGuardrailData("certified", bad); err == nil {
		t.Fatalf("expected error for empty claims")
	}

	// claim empty string
	bad = json.RawMessage(`{"claims": [""]}`)
	if err := validateGuardrailData("certified", bad); err == nil {
		t.Fatalf("expected error for empty claim value")
	}
}

func TestValidateGuardrailData_UnknownType(t *testing.T) {
	var msg json.RawMessage = json.RawMessage(`{"foo": "bar"}`)
	if err := validateGuardrailData("weird", msg); err == nil {
		t.Fatalf("expected error for unknown type")
	}
}
