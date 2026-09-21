package handlers

import (
	"encoding/json"
)

// stringOf extracts a string from a Debezium event map. Debezium emits numeric
// PKs and UUIDs as JSON strings; some fields come as float64 (when the JSON
// schema says integer and the upstream value fits in a float).
func stringOf(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// Lossy but acceptable for non-PK columns; for PKs we always get a string.
		return ""
	default:
		return ""
	}
}

func intOf(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

func boolOf(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// numericOf extracts a numeric column value as float64. Returns 0 if absent
// or non-numeric. Debezium emits numeric(p,s) as a base64-encoded Decimal
// with the stream_loader's decoder — the aggregate consumer doesn't decode
// that (yet), so we get whatever pgjdbc gave us. For screening purposes a 0
// from an unparseable numeric is conservative (the commitment check fails).
func numericOf(v any) float64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		// Try JSON unquote + parse if it looks like a number string.
		var f float64
		if err := json.Unmarshal([]byte(t), &f); err == nil {
			return f
		}
	}
	return 0
}

// mustJSON marshals v to a JSON string. If v is nil, returns "null". The
// INSERT statement casts the result to jsonb so an empty array or object is
// preserved correctly.
func mustJSON(v any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

// jsonEqual compares two Debezium event values for equality including their
// JSON shape. Used to implement IS DISTINCT FROM gating for trigger logic.
//
// Two nils are equal. Two values whose JSON serializations match are equal.
// nil IS DISTINCT FROM any non-nil value.
func jsonEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

// ColEq compares two raw Debezium column values for equality. Delegates to
// jsonEqual for a definitive answer; the JSON comparison is correct for all
// JSON-compatible types that Debezium emits (string, number, bool, null, array, object).
func ColEq(a, b any) bool {
	return jsonEqual(a, b)
}
