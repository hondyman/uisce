package msgcat

import (
	"regexp"
	"sort"
	"strings"
)

// Parameters are PeopleSoft-style %1..%9.
var placeholderRE = regexp.MustCompile(`%[1-9]`)

// Placeholders returns the distinct parameters a text uses, sorted.
func Placeholders(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range placeholderRE.FindAllString(text, -1) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// SamePlaceholders reports whether two texts use the same parameters.
func SamePlaceholders(a, b string) bool {
	return strings.Join(Placeholders(a), ",") == strings.Join(Placeholders(b), ",")
}

// Format substitutes %n with params[n-1] in one pass, so a parameter value
// that itself contains "%2" is never substituted again. A parameter the
// caller did not supply renders as empty.
func Format(text string, params []string) string {
	return placeholderRE.ReplaceAllStringFunc(text, func(p string) string {
		i := int(p[1] - '1')
		if i < len(params) {
			return params[i]
		}
		return ""
	})
}
