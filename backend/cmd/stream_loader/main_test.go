package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeDebeziumDecimal(t *testing.T) {
	cases := []struct {
		name  string
		b64   string
		scale string
		want  string
	}{
		{"zero scale 0", "AAAAAA==", "0", "0"},
		{"123.45 scale 2", "MDk=", "2", "123.45"},
		// -1 scale 0: 0xFF byte is base64 encoded as "//8="
		{"-1 scale 0", "//8=", "0", "-1"},
	}
	for _, c := range cases {
		got, err := decodeDebeziumDecimal(c.b64, c.scale)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDurationMicros(t *testing.T) {
	cases := []struct {
		name                string
		months, days, millis int64
		want                int64
	}{
		{"zero", 0, 0, 0, 0},
		{"30 days in months", 1, 0, 0, 30 * 24 * 3600 * 1000 * 1000},
		{"1 day + 500 ms", 0, 1, 500, (24*3600*1000 + 500) * 1000},
		{"negative days", 0, -1, 0, -24 * 3600 * 1000 * 1000},
	}
	for _, c := range cases {
		got := durationMicros(c.months, c.days, c.millis)
		if got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFormatDaysSinceEpochToDate(t *testing.T) {
	cases := []struct {
		days int64
		want string
	}{
		{0, "1970-01-01"},
		{1, "1970-01-02"},
		{18262, "2020-01-01"},
		{-1, "1969-12-31"},
	}
	for _, c := range cases {
		got := formatDaysSinceEpochToDate(c.days)
		if got != c.want {
			t.Errorf("days=%d: got %q, want %q", c.days, got, c.want)
		}
	}
}

func TestFormatUnixMillisToISO(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "1970-01-01T00:00:00Z"},
		{1577836800000, "2020-01-01T00:00:00Z"},
	}
	for _, c := range cases {
		got := formatUnixMillisToISO(c.ms)
		if got != c.want {
			t.Errorf("ms=%d: got %q, want %q", c.ms, got, c.want)
		}
	}
}

func TestDecodeRecord_DeleteOp(t *testing.T) {
	// A delete event with `before` but no `after` should be classified as
	// a delete with payload equal to the before row.
	raw := []byte(`{
        "schema": null,
        "payload": {
            "before": {"id": "abc-123", "name": "old"},
            "after":  null,
            "op": "d",
            "ts_ms": 100
        }
    }`)
	res, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord: %v", err)
	}
	if res.skip {
		t.Fatal("expected non-skip result for delete event")
	}
	if res.op != "d" {
		t.Fatalf("expected op=d, got %q", res.op)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(res.payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["id"] != "abc-123" {
		t.Errorf("expected id=abc-123 in payload, got %v", got["id"])
	}
}

func TestDecodeRecord_TombstoneIsSkip(t *testing.T) {
	// Tombstone (entire payload null) should be skipped.
	raw := []byte(`{"payload": {"before": null, "after": null, "op": "d", "ts_ms": 100}}`)
	res, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord: %v", err)
	}
	if !res.skip {
		t.Errorf("expected skip for tombstone, got %+v", res)
	}
}

func TestDecodeRecord_UpsertOp(t *testing.T) {
	raw := []byte(`{
        "payload": {
            "before": null,
            "after":  {"id": "xyz", "value": 42},
            "op": "u",
            "ts_ms": 100
        }
    }`)
	res, err := decodeRecord(raw)
	if err != nil {
		t.Fatalf("decodeRecord: %v", err)
	}
	if res.skip {
		t.Fatal("expected non-skip for upsert")
	}
	if res.op != "u" {
		t.Errorf("expected op=u, got %q", res.op)
	}
}

// TestDecodeDurationB64_RoundTrip catches symmetric bugs in the wire-format
// decoder by encoding a known (months, days, millis) through packDurationB64
// (which is the inverse of decodeDurationB64) and asserting the round-trip.
func TestDecodeDurationB64_RoundTrip(t *testing.T) {
	cases := []struct {
		name                string
		months, days, millis int32
	}{
		{"zero", 0, 0, 0},
		{"30 days via month", 1, 0, 0},
		{"days only", 0, 5, 0},
		{"millis only", 0, 0, 123456},
		{"mixed", 2, 14, 987654},
		{"negative months", -1, 0, 0},
		{"all negative", -1, -1, -1},
	}
	for _, c := range cases {
		packed := packDurationB64(c.months, c.days, c.millis)
		got, ok := decodeDurationB64(packed)
		if !ok {
			t.Errorf("%s: decode failed for packed=%q", c.name, packed)
			continue
		}
		want := durationMicros(int64(c.months), int64(c.days), int64(c.millis))
		if got != want {
			t.Errorf("%s: got %d, want %d (packed=%q)", c.name, got, want, packed)
		}
	}
}

// TestDecodeDurationB64_LengthGuard asserts the fail-soft behavior when
// the base64-decoded bytes are NOT exactly 12 bytes long. The decoder
// returns ok=false and the value should land as the undecoded base64 string
// in StarRocks (NULL on type mismatch) rather than panic or produce garbage.
func TestDecodeDurationB64_LengthGuard(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
	}{
		{"nil", nil},
		{"not a string", 42},
		{"not base64", "not-base64-at-all!@"},
		{"too short", "AAA="},                    // 3 bytes
		{"too long", strings.Repeat("A", 20)},    // 15 bytes
	}
	for _, c := range cases {
		_, ok := decodeDurationB64(c.in)
		if ok {
			t.Errorf("%s: expected ok=false", c.name)
		}
	}
}

// TestDecodeDecimals_Duration_Field is the integration test for the
// Duration branch: build a Debezium envelope with a schema declaring
// a Duration field on the after row, confirm decodeDecimals translates
// the value to integer microseconds end-to-end.
//
// Schema fragment shape: matches what Debezium's JSON converter emits
// with time.precision.mode=connect on Postgres `interval`.
func TestDecodeDecimals_Duration_Field(t *testing.T) {
	packed := packDurationB64(0, 1, 500) // 1 day + 500 ms
	after := map[string]interface{}{
		"id":       "id-1",
		"duration": packed,
	}
	schemaRaw := json.RawMessage(`{
        "fields": [
            {
                "type": "struct",
                "field": "after",
                "fields": [
                    {"type": "int64", "field": "id", "name": "io.debezium.time.MicroTime"},
                    {"type": "int32", "field": "duration", "name": "org.apache.kafka.connect.data.Duration"}
                ]
            }
        ]
    }`)
	decodeDecimals(schemaRaw, after)

	want := int64((24*3600*1000 + 500) * 1000)
	if v, ok := after["duration"].(int64); !ok || v != want {
		t.Errorf("duration: got %v (%T), want %d", after["duration"], after["duration"], want)
	}
}
