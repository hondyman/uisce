package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactBody_RedactsPasswordField(t *testing.T) {
	in := []byte(`{"email":"x@y","password":"super-secret"}`)
	out := string(redactBody(in))
	if strings.Contains(out, "super-secret") {
		t.Fatalf("password not redacted: %s", out)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output not valid JSON: %s", out)
	}
	if parsed["password"] != "<redacted>" {
		t.Fatalf("expected password=<redacted>, got %v", parsed["password"])
	}
	if parsed["email"] != "x@y" {
		t.Fatalf("email field lost: %v", parsed["email"])
	}
}

func TestRedactBody_LeavesNonCredentialBodiesUnchanged(t *testing.T) {
	in := []byte(`{"name":"foo","description":"bar"}`)
	out := redactBody(in)
	if string(out) != string(in) {
		t.Fatalf("body changed for non-credential body: %s -> %s", in, out)
	}
}

func TestRedactBody_PassesNonJSONThrough(t *testing.T) {
	in := []byte("not json")
	out := redactBody(in)
	if string(out) != string(in) {
		t.Fatalf("non-JSON body changed: %s -> %s", in, out)
	}
}

func TestRedactBody_RedactsMultipleFields(t *testing.T) {
	in := []byte(`{"password":"a","token":"b","secret":"c","api_key":"d","name":"keep"}`)
	out := string(redactBody(in))
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output not valid JSON: %s", out)
	}
	for _, f := range []string{"password", "token", "secret", "api_key"} {
		if parsed[f] != "<redacted>" {
			t.Fatalf("field %s not redacted: %v", f, parsed[f])
		}
	}
	if parsed["name"] != "keep" {
		t.Fatalf("non-credential field lost: %v", parsed["name"])
	}
}

func TestRedactBody_NestedCredentialPassesThrough(t *testing.T) {
	// Documents the top-level-only limitation. If a nested-credential
	// endpoint appears, extend redactBody to recurse. A test that
	// asserts "nested password passes through" fails loudly the moment
	// recursive redaction is added — that's the "you just changed
	// documented behavior" signal.
	in := []byte(`{"user":{"password":"nested-secret"},"email":"x@y"}`)
	out := string(redactBody(in))
	if !strings.Contains(out, "nested-secret") {
		t.Fatalf("nested password unexpectedly redacted (top-level-only limitation broken): %s", out)
	}
}

func TestRedactBody_OverRedactsLiteralTokenField(t *testing.T) {
	// A pagination cursor named "token" is also redacted. Acceptable
	// for log safety; over-redaction is preferred to under-redaction.
	// This test makes the over-redaction visible-by-test rather than
	// discovered-in-production.
	in := []byte(`{"token":"page-cursor-123","name":"keep"}`)
	out := string(redactBody(in))
	if strings.Contains(out, "page-cursor-123") {
		t.Fatalf("token cursor not redacted: %s", out)
	}
}

func TestRedactAuthHeader_BearerTokenRedacted(t *testing.T) {
	in := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig"
	out := redactAuthHeader(in)
	if out != "Bearer <redacted>" {
		t.Fatalf("expected Bearer <redacted>, got %q", out)
	}
	if strings.Contains(out, "eyJ") {
		t.Fatalf("token not redacted: %s", out)
	}
}

func TestRedactAuthHeader_NonBearerRedacted(t *testing.T) {
	// Basic credentials are base64(user:pass). Logging the raw value
	// would leak both halves of the credential pair. The helper's
	// contract is "Authorization is safe to log" — applies to all
	// schemes, not just Bearer.
	in := "Basic dXNlcjpwYXNz"
	out := redactAuthHeader(in)
	if out != "Basic <redacted>" {
		t.Fatalf("expected Basic <redacted>, got %q", out)
	}
	if strings.Contains(out, "dXNlcjpwYXNz") {
		t.Fatalf("Basic credential not redacted: %s", out)
	}
}

func TestRedactAuthHeader_SchemelessRedacted(t *testing.T) {
	// Scheme-less non-empty values (raw API keys, signed cookies, etc.)
	// are a real pattern in the Authorization header. Redact wholesale.
	in := "raw-api-key-no-scheme"
	out := redactAuthHeader(in)
	if out != "<redacted>" {
		t.Fatalf("expected <redacted>, got %q", out)
	}
}

func TestRedactAuthHeader_EmptyPassesThrough(t *testing.T) {
	if got := redactAuthHeader(""); got != "" {
		t.Fatalf("empty header changed: %q", got)
	}
}
