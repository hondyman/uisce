package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// The fixture mirrors the real datasource config shape with obviously fake values.
const fakeConfig = `{
  "auth": {"basic": {"username": "pguser", "password": "hunter2-FAKE"}},
  "host": "db.example.internal", "port": 5432, "database": "crims",
  "schema": "orm,vend,mdm", "sslmode": "verify-full", "auth_type": "key_pair",
  "api_key": "sk-FAKE-1234", "base_url": "https://api.example.internal",
  "ca_cert": "-----BEGIN CERTIFICATE-----\nFAKECA\n-----END CERTIFICATE-----\n",
  "client_cert": "-----BEGIN CERTIFICATE-----\nFAKECLIENT\n-----END CERTIFICATE-----\n",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\nFAKEKEYMATERIAL\n-----END RSA PRIVATE KEY-----\n"
}`

func decodeConfig(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v (%s)", err, raw)
	}
	return m
}

func TestRedactSecretConfig_RemovesCredentialsKeepsConnectionDetails(t *testing.T) {
	out := redactSecretConfig(json.RawMessage(fakeConfig))
	s := string(out)
	for _, leaked := range []string{"hunter2-FAKE", "sk-FAKE-1234", "FAKEKEYMATERIAL", "PRIVATE KEY"} {
		if strings.Contains(s, leaked) {
			t.Errorf("redacted config still contains %q: %s", leaked, s)
		}
	}
	m := decodeConfig(t, out)
	for k, want := range map[string]interface{}{"host": "db.example.internal", "database": "crims", "schema": "orm,vend,mdm", "sslmode": "verify-full", "auth_type": "key_pair"} {
		if m[k] != want {
			t.Errorf("%s = %v; want it kept as %v", k, m[k], want)
		}
	}
	if m["port"] != float64(5432) {
		t.Errorf("port = %v; want 5432", m["port"])
	}
	basic := m["auth"].(map[string]interface{})["basic"].(map[string]interface{})
	if basic["username"] != "pguser" {
		t.Errorf("username = %v; want kept", basic["username"])
	}
	if basic["password"] != redactedValue || m["api_key"] != redactedValue || m["private_key"] != redactedValue {
		t.Errorf("secrets not replaced by %q: password=%v api_key=%v private_key=%v", redactedValue, basic["password"], m["api_key"], m["private_key"])
	}
	// Certificates are public and stay (the UI shows connection identity).
	if !strings.Contains(m["client_cert"].(string), "BEGIN CERTIFICATE") || !strings.Contains(m["ca_cert"].(string), "BEGIN CERTIFICATE") {
		t.Errorf("certificates should not be redacted")
	}
}

func TestRedactSecretConfig_PEMPrivateKeyRedactedUnderAnyKeyName(t *testing.T) {
	out := redactSecretConfig(json.RawMessage(`{"nested":{"mystery":"-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----"}}`))
	if strings.Contains(string(out), "BEGIN PRIVATE KEY") {
		t.Errorf("PEM private key survived under an unrecognised key name: %s", out)
	}
}

func TestRedactSecretConfig_ArraysAndNestedObjects(t *testing.T) {
	out := redactSecretConfig(json.RawMessage(`{"replicas":[{"host":"a","password":"p1"},{"host":"b","token":"t2"}]}`))
	s := string(out)
	if strings.Contains(s, `"p1"`) || strings.Contains(s, `"t2"`) || !strings.Contains(s, `"a"`) || !strings.Contains(s, `"b"`) {
		t.Errorf("array elements not handled correctly: %s", s)
	}
}

func TestRedactSecretConfig_EmptyAndNonStringSecrets(t *testing.T) {
	out := decodeConfig(t, redactSecretConfig(json.RawMessage(`{"password":"","token":12345,"note":"hello"}`)))
	if out["password"] != "" {
		t.Errorf("empty password should stay empty (unset stays distinguishable), got %v", out["password"])
	}
	if out["token"] != redactedValue {
		t.Errorf("numeric secret not redacted: %v", out["token"])
	}
	if out["note"] != "hello" {
		t.Errorf("non-secret value changed: %v", out["note"])
	}
}

func TestRedactSecretConfig_FailsClosed(t *testing.T) {
	if got := string(redactSecretConfig(json.RawMessage(`{not json password=hunter2`))); got != `{}` {
		t.Errorf("unparseable config returned %q; want {}", got)
	}
	if got := redactSecretConfig(nil); got != nil {
		t.Errorf("nil config should stay nil so omitempty still omits it, got %q", got)
	}
}
