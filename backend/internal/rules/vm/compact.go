package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Compact returns a canonical JSON representation of a RuleNode suitable for
// checksumming and diffing:
//
//   - object keys are sorted lexicographically
//   - whitespace is stripped
//   - float literals are normalized to their shortest round-trip form
//     (e.g. "105.50" -> "105.5")
//   - integer literals are preserved verbatim (no float precision loss for
//     large integers)
//
// Two semantically identical ASTs produce byte-identical Compact output,
// regardless of key ordering or float formatting in the source.
//
// Compact relies on RuleNode's custom MarshalJSON (flat wire representation).
// Any change to that marshaling changes checksums globally — the round-trip
// tests in compact_test.go exist to make that visible.
func Compact(node RuleNode) (json.RawMessage, error) {
	raw, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("vm.Compact: marshal rule node: %w", err)
	}
	return CanonicalJSON(raw)
}

// CanonicalJSON normalizes arbitrary JSON bytes into canonical form using the
// same rules as Compact. It accepts any JSON value (objects, arrays, scalars).
func CanonicalJSON(raw []byte) (json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // preserve numeric literals; we normalize explicitly below
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("vm.CanonicalJSON: decode: %w", err)
	}
	var buf bytes.Buffer
	if err := encodeCanonical(&buf, v); err != nil {
		return nil, fmt.Errorf("vm.CanonicalJSON: encode: %w", err)
	}
	return json.RawMessage(buf.Bytes()), nil
}

func encodeCanonical(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case json.Number:
		buf.WriteString(normalizeNumber(string(x)))
	case string:
		b, err := json.Marshal(x) // deterministic escaping
		if err != nil {
			return err
		}
		buf.Write(b)
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeCanonical(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(kb)
			buf.WriteByte(':')
			if err := encodeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("vm.CanonicalJSON: unsupported type %T", v)
	}
	return nil
}

// normalizeNumber returns the shortest round-trip representation of a JSON
// number literal. Integers (no '.', 'e', or 'E') are preserved verbatim so
// that large integer values (e.g. IDs) never lose precision through float64
// conversion. Floats are re-emitted via encoding/json's float encoder, which
// produces the shortest form that round-trips (Ryu / shortest-digits).
func normalizeNumber(lit string) string {
	if !strings.ContainsAny(lit, ".eE") {
		return lit
	}
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil {
		return lit // defensively preserve; invalid numbers cannot occur in valid JSON
	}
	b, err := json.Marshal(f)
	if err != nil {
		return lit
	}
	return string(b)
}
