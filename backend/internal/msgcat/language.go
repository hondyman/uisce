package msgcat

import (
	"sort"
	"strconv"
	"strings"
)

// Language is one language the catalog can hold text in.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"` // in the language itself
	RTL  bool   `json:"rtl,omitempty"`
}

// Languages are the app's locales (frontend/src/i18n/locales.ts LOCALES),
// so a user's UI language is always a language the catalog can answer in.
var Languages = []Language{
	{Code: "en", Name: "English"},
	{Code: "es", Name: "Español"},
	{Code: "fr", Name: "Français"},
	{Code: "de", Name: "Deutsch"},
	{Code: "pt-BR", Name: "Português (Brasil)"},
	{Code: "ja", Name: "日本語"},
	{Code: "zh-CN", Name: "简体中文"},
	{Code: "ar", Name: "العربية", RTL: true},
}

// BaseLanguage is the message of record: every message has English text,
// and every other language falls back to it.
const BaseLanguage = "en"

// NormalizeLanguage maps a language tag to a catalog language: exact match
// ignoring case, then by primary subtag ("fr-CA" -> "fr", "pt" -> "pt-BR").
func NormalizeLanguage(tag string) (string, bool) {
	tag = strings.TrimSpace(strings.ReplaceAll(tag, "_", "-"))
	if tag == "" {
		return "", false
	}
	for _, l := range Languages {
		if strings.EqualFold(l.Code, tag) {
			return l.Code, true
		}
	}
	primary := strings.ToLower(strings.SplitN(tag, "-", 2)[0])
	for _, l := range Languages {
		if strings.ToLower(strings.SplitN(l.Code, "-", 2)[0]) == primary {
			return l.Code, true
		}
	}
	return "", false
}

// Preferences turns an Accept-Language header into catalog languages in
// order of preference, always ending with the base language.
func Preferences(acceptLanguage string) []string {
	type pref struct {
		code string
		q    float64
		pos  int
	}
	var prefs []pref
	for i, part := range strings.Split(acceptLanguage, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		q := 1.0
		for _, f := range fields[1:] {
			if v, ok := strings.CutPrefix(strings.TrimSpace(f), "q="); ok {
				if p, err := strconv.ParseFloat(v, 64); err == nil {
					q = p
				}
			}
		}
		if code, ok := NormalizeLanguage(fields[0]); ok && q > 0 {
			prefs = append(prefs, pref{code, q, i})
		}
	}
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].q > prefs[j].q })
	var out []string
	seen := map[string]bool{}
	for _, p := range prefs {
		if !seen[p.code] {
			seen[p.code] = true
			out = append(out, p.code)
		}
	}
	if !seen[BaseLanguage] {
		out = append(out, BaseLanguage)
	}
	return out
}
