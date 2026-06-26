package canonical

import (
	"testing"
)

func TestMarshalDeterministic_Stability(t *testing.T) {
	v := map[string]interface{}{
		"z": "last",
		"a": "first",
		"m": map[string]interface{}{
			"nested_z": 2,
			"nested_a": 1,
		},
	}

	b1, err := MarshalDeterministic(v)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := MarshalDeterministic(v)
	if err != nil {
		t.Fatal(err)
	}

	if string(b1) != string(b2) {
		t.Fatalf("not deterministic:\n%s\nvs\n%s", b1, b2)
	}
}

func TestMarshalDeterministic_KeyOrder(t *testing.T) {
	v := map[string]interface{}{
		"c": 3,
		"b": 2,
		"a": 1,
	}

	b, err := MarshalDeterministic(v)
	if err != nil {
		t.Fatal(err)
	}

	// Should be sorted: {"a":1,"b":2,"c":3}
	expected := `{"a":1,"b":2,"c":3}`
	if string(b) != expected {
		t.Fatalf("expected %s, got %s", expected, b)
	}
}

func BenchmarkMarshalDeterministic(b *testing.B) {
	v := map[string]interface{}{
		"client_id":   "c_12345",
		"action_type": "tax_loss_harvest",
		"timestamp":   "2023-11-23T15:45:00Z",
		"details": map[string]interface{}{
			"symbol": "MSFT",
			"shares": 100,
			"loss":   -1500.50,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalDeterministic(v)
		if err != nil {
			b.Fatal(err)
		}
	}
}
