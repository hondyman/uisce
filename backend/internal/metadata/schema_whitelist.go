package metadata

import "strings"

// parseSchemaWhitelist turns the connection's schema setting into the list of schemas to scan. The setting is
// a comma-separated list ("orm,vend,ref,mdm"); it is also what goes into the connection's search_path, which
// accepts the same form. Names are trimmed, blanks and repeats dropped, and order kept. A setting with no
// comma yields the single schema it names, as before.
func parseSchemaWhitelist(raw string) []string {
	var out []string
	seen := map[string]bool{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
