package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrInvalidJSON is returned when input data cannot be canonicalized.
var ErrInvalidJSON = errors.New("canonical: invalid JSON")

// Transform canonicalizes an arbitrary JSON input into RFC 8785 compliant canonical JSON.
func Transform(input []byte) ([]byte, error) {
	var val interface{}
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&val); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	// Verify no trailing garbage
	if dec.More() {
		return nil, fmt.Errorf("%w: unexpected trailing data", ErrInvalidJSON)
	}
	var buf bytes.Buffer
	if err := serializeValue(&buf, val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Marshal encodes an arbitrary Go data structure into RFC 8785 canonical JSON.
func Marshal(v interface{}) ([]byte, error) {
	intermediate, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return Transform(intermediate)
}

func serializeValue(w *bytes.Buffer, v interface{}) error {
	switch val := v.(type) {
	case nil:
		w.WriteString("null")
	case bool:
		if val {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
	case string:
		writeCanonicalString(w, val)
	case json.Number:
		if err := writeCanonicalNumber(w, val.String()); err != nil {
			return err
		}
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return errors.New("canonical: NaN and Infinity are not valid JSON numbers")
		}
		if err := writeCanonicalFloat(w, val); err != nil {
			return err
		}
	case int:
		w.WriteString(strconv.Itoa(val))
	case int64:
		w.WriteString(strconv.FormatInt(val, 10))
	case uint64:
		w.WriteString(strconv.FormatUint(val, 10))
	case []interface{}:
		w.WriteByte('[')
		for i, elem := range val {
			if i > 0 {
				w.WriteByte(',')
			}
			if err := serializeValue(w, elem); err != nil {
				return err
			}
		}
		w.WriteByte(']')
	case map[string]interface{}:
		w.WriteByte('{')
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sortKeys(keys)
		for i, k := range keys {
			if i > 0 {
				w.WriteByte(',')
			}
			writeCanonicalString(w, k)
			w.WriteByte(':')
			if err := serializeValue(w, val[k]); err != nil {
				return err
			}
		}
		w.WriteByte('}')
	default:
		// Fallback: roundtrip through json marshal with json.Number decoder
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return serializeValue(w, raw)
	}
	return nil
}

func sortKeys(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		return utf16Less(keys[i], keys[j])
	})
}

// utf16Less compares two UTF-8 strings by their UTF-16 code units (per RFC 8785 §3.2.3).
func utf16Less(a, b string) bool {
	u1 := utf16.Encode([]rune(a))
	u2 := utf16.Encode([]rune(b))
	minLen := len(u1)
	if len(u2) < minLen {
		minLen = len(u2)
	}
	for i := 0; i < minLen; i++ {
		if u1[i] != u2[i] {
			return u1[i] < u2[i]
		}
	}
	return len(u1) < len(u2)
}

func writeCanonicalString(w *bytes.Buffer, s string) {
	w.WriteByte('"')
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		switch r {
		case '"':
			w.WriteString(`\"`)
		case '\\':
			w.WriteString(`\\`)
		case '\b':
			w.WriteString(`\b`)
		case '\f':
			w.WriteString(`\f`)
		case '\n':
			w.WriteString(`\n`)
		case '\r':
			w.WriteString(`\r`)
		case '\t':
			w.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(w, `\u%04x`, r)
			} else {
				w.WriteRune(r)
			}
		}
	}
	w.WriteByte('"')
}

func writeCanonicalNumber(w *bytes.Buffer, numStr string) error {
	// If it's a negative zero string: "-0", "-0.0", "-0.00" -> serialize to 0
	if numStr == "-0" || numStr == "-0.0" || numStr == "-0.00" {
		w.WriteByte('0')
		return nil
	}

	// If it's an exact integer (no '.', no 'e', no 'E')
	if !strings.ContainsAny(numStr, ".eE") {
		// Verify valid integer syntax
		if _, err := strconv.ParseInt(numStr, 10, 64); err == nil {
			w.WriteString(numStr)
			return nil
		}
		if _, err := strconv.ParseUint(numStr, 10, 64); err == nil {
			w.WriteString(numStr)
			return nil
		}
	}

	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return err
	}
	return writeCanonicalFloat(w, f)
}

func writeCanonicalFloat(w *bytes.Buffer, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("canonical: invalid float NaN/Inf")
	}
	if f == 0 {
		// IEEE 754 -0 should serialize to 0 per RFC 8785 §3.2.2.3
		w.WriteByte('0')
		return nil
	}

	// Format matching ECMAScript Number.prototype.toString() per RFC 8785 §3.2.2.3
	s := formatESNumber(f)
	w.WriteString(s)
	return nil
}

func formatESNumber(f float64) string {
	abs := math.Abs(f)
	// In ECMAScript 6:
	// - If k <= 0 and k >= -6: standard fixed decimal notation
	// - If k <= -7: exponential notation (e.g. 1e-7)
	// - If k > 21: exponential notation (e.g. 1e+21)
	// - If 1 <= k <= 21: standard integer/fixed notation
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		return canonicalizeExp(s)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func canonicalizeExp(s string) string {
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] == 'e' {
			buf.WriteByte('e')
			if i+1 < len(s) && s[i+1] == '+' {
				buf.WriteByte('+')
				i++
			} else if i+1 < len(s) && s[i+1] == '-' {
				buf.WriteByte('-')
				i++
			}
			// Skip leading zeros in exponent (e.g. e+07 -> e+7)
			for i+1 < len(s) && s[i+1] == '0' && i+2 < len(s) {
				i++
			}
			continue
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}
