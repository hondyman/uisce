package db

import (
	"encoding/json"
	"strings"
	"testing"
)

// A cloned connection must not carry the gold copy's credentials, whether
// inline in metadata or as a secret_path into the gold copy's secrets.
func TestSanitizeConnectionMetadata_DropsCredentials(t *testing.T) {
	in := []byte(`{"host":"h","port":5432,"schema":"orm","ca_cert":"CA","client_cert":"CERT",
		"auth_type":"key_pair","private_key":"-----BEGIN PRIVATE KEY-----x","api_key":"ak",
		"secret_path":"/connections/gold/conn","auth":{"basic":{"username":"u","password":"gold-pass"}}}`)
	out, err := sanitizeConnectionMetadata(in)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, leaked := range []string{"gold-pass", "PRIVATE KEY", "secret_path", `"ak"`} {
		if strings.Contains(s, leaked) {
			t.Errorf("clone metadata still contains %q: %s", leaked, s)
		}
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	if m["host"] != "h" || m["ca_cert"] != "CA" || m["schema"] != "orm" {
		t.Errorf("non-secret metadata lost: %s", s)
	}
}
