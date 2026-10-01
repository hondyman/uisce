package msgcat

import (
	"errors"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestPreferences(t *testing.T) {
	cases := map[string][]string{
		"":                        {"en"},
		"fr-CA,fr;q=0.9,en;q=0.8": {"fr", "en"},
		"de;q=0.5, es":            {"es", "de", "en"},
		"pt":                      {"pt-BR", "en"},
		"zh-Hans-CN, xx, ja;q=0":  {"zh-CN", "en"},
		"ar-EG;q=0.7,en-GB;q=0.9": {"en", "ar"},
		"*":                       {"en"},
	}
	for in, want := range cases {
		if got := Preferences(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Preferences(%q) = %v, want %v", in, got, want)
		}
	}
}

// The catalog's languages are exactly the app's locales, so a user's UI
// language is always one the catalog can answer in.
func TestLanguagesMatchFrontendLocales(t *testing.T) {
	src, err := os.ReadFile("../../../frontend/src/i18n/locales.ts")
	if err != nil {
		t.Skipf("frontend not checked out: %v", err)
	}
	m := regexp.MustCompile(`export const LOCALES = \[([^\]]+)\] as const`).FindSubmatch(src)
	if m == nil {
		t.Fatal("LOCALES not found in locales.ts")
	}
	var frontend []string
	for _, s := range regexp.MustCompile(`'([^']+)'`).FindAllSubmatch(m[1], -1) {
		frontend = append(frontend, string(s[1]))
	}
	var ours []string
	for _, l := range Languages {
		ours = append(ours, l.Code)
	}
	if !reflect.DeepEqual(ours, frontend) {
		t.Fatalf("msgcat.Languages = %v, frontend LOCALES = %v", ours, frontend)
	}
}

func TestFormat(t *testing.T) {
	if got := Format("Run %1 failed: %2.", []string{"R-7", "bad %1 value"}); got != "Run R-7 failed: bad %1 value." {
		t.Errorf("got %q (a parameter's own %%1 must not be substituted)", got)
	}
	if got := Format("%2 then %1 and %3", []string{"a", "b"}); got != "b then a and " {
		t.Errorf("got %q", got)
	}
}

func TestPlaceholders(t *testing.T) {
	if got := Placeholders("%2 of %1, again %2"); !reflect.DeepEqual(got, []string{"%1", "%2"}) {
		t.Errorf("got %v", got)
	}
	if !SamePlaceholders("Run %1 failed: %2", "Échec %2 de %1") || SamePlaceholders("%1", "%1 %2") {
		t.Error("SamePlaceholders")
	}
}

func TestErrorCodeAndWrap(t *testing.T) {
	cause := errors.New(`pq: relation "x" does not exist`)
	e := New(3000, 12, "R-7").WithStatus(409).Wrap(cause)
	if e.Code() != "3000-12" || e.Status != 409 || !errors.Is(e, cause) {
		t.Fatalf("%+v", e)
	}
	var me *Error
	if !errors.As(error(e), &me) || !strings.Contains(e.Error(), "3000-12") {
		t.Fatal(e.Error())
	}
	if s, n, ok := ParseCode(" 9100-17 "); !ok || s != 9100 || n != 17 {
		t.Fatal("ParseCode")
	}
	for _, bad := range []string{"", "9100", "a-1", "0-1", "1-0", "1-x"} {
		if _, _, ok := ParseCode(bad); ok {
			t.Errorf("ParseCode(%q) accepted", bad)
		}
	}
}

func TestClaimString(t *testing.T) {
	type claims struct {
		Email string
		Roles []string
	}
	var nilClaims *claims
	cases := []struct {
		in   any
		want string
	}{
		{&claims{Email: "a@example.com"}, "a@example.com"},
		{claims{Email: "b@example.com"}, "b@example.com"},
		{nilClaims, ""},
		{nil, ""},
		{"not a struct", ""},
		{&struct{ Roles []string }{}, ""},
	}
	for _, c := range cases {
		if got := claimString(c.in, "Email"); got != c.want {
			t.Errorf("claimString(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}
