package canonical

import (
	"bytes"
	"encoding/json"
	"sort"
)

// MarshalDeterministic returns canonical JSON with sorted object keys.
// This ensures consistent hashing for audit chains.
func MarshalDeterministic(v interface{}) ([]byte, error) {
	// First marshal to get proper JSON encoding
	intermediate, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Unmarshal into generic structure
	var generic interface{}
	if err := json.Unmarshal(intermediate, &generic); err != nil {
		return nil, err
	}

	// Canonicalize (sort keys recursively)
	canonicalize(generic)

	// Marshal with sorted keys
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	if err := enc.Encode(generic); err != nil {
		return nil, err
	}

	// Remove trailing newline
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func canonicalize(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		// Get sorted keys
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Recursively canonicalize values
		for _, k := range keys {
			canonicalize(t[k])
		}

	case []interface{}:
		// Canonicalize array elements
		for i := range t {
			canonicalize(t[i])
		}

	default:
		// Primitives: nothing to do
	}
}
