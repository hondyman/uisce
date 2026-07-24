package starlib

import "strings"

// ProjectRecord returns a copy of data containing only the values referenced by fieldPaths.
//
// Field paths are dot-separated (e.g. "account.account_type" or "page.aum").
// If fieldPaths is empty, it returns data as-is to avoid allocations.
//
// Missing paths are ignored.
func ProjectRecord(data map[string]interface{}, fieldPaths []string) map[string]interface{} {
	if len(fieldPaths) == 0 {
		return data
	}
	out := make(map[string]interface{}, len(fieldPaths))
	for _, raw := range fieldPaths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) == 1 {
			if v, ok := data[parts[0]]; ok {
				out[parts[0]] = v
			}
			continue
		}

		v, ok := getByPath(data, parts)
		if !ok {
			continue
		}
		setByPath(out, parts, v)
	}
	return out
}

func getByPath(root map[string]interface{}, parts []string) (interface{}, bool) {
	var cur interface{} = root
	for i := 0; i < len(parts); i++ {
		key := parts[i]
		switch m := cur.(type) {
		case map[string]interface{}:
			next, ok := m[key]
			if !ok {
				return nil, false
			}
			cur = next
		default:
			return nil, false
		}
	}
	return cur, true
}

func setByPath(dst map[string]interface{}, parts []string, value interface{}) {
	m := dst
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		next, ok := m[key]
		if !ok {
			nm := map[string]interface{}{}
			m[key] = nm
			m = nm
			continue
		}
		switch mm := next.(type) {
		case map[string]interface{}:
			m = mm
		default:
			nm := map[string]interface{}{}
			m[key] = nm
			m = nm
		}
	}
	m[parts[len(parts)-1]] = value
}
