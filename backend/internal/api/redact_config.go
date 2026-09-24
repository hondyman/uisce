package api

import (
	"encoding/json"
	"regexp"
	"strings"
)

// redactedValue replaces a secret. It is a non-empty string on purpose: the UI shows a
// "present" indicator by truthiness, so a redacted secret still reads as "set".
const redactedValue = "[redacted]"

// secretKeyRE matches config keys whose values are credentials. Certificates (ca_cert,
// client_cert) are public and deliberately not matched; private keys are.
var secretKeyRE = regexp.MustCompile(`(?i)(pass(word|wd|phrase)?|pwd|secret|token|private[_-]?key|api[_-]?key|access[_-]?key|credential|client[_-]?key|bearer|authorization|connection[_-]?string|dsn)`)

// redactSecretConfig returns the datasource config with every credential replaced by
// "[redacted]" so it can be sent to a browser. Tenant admin listings must never carry
// passwords or private keys: they end up in localStorage, the console and any screenshot.
//
// Redaction is by key name (secretKeyRE) and by value (any string containing a PEM private
// key, whatever its key is called). Empty values are left as they are so "unset" stays
// distinguishable. Fail-closed: a config that cannot be parsed yields an empty object.
func redactSecretConfig(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return json.RawMessage(`{}`)
	}
	out, err := json.Marshal(redactValue(v, false))
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

func redactValue(v interface{}, secret bool) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			t[k] = redactValue(child, secretKeyRE.MatchString(k))
		}
		return t
	case []interface{}:
		for i, child := range t {
			t[i] = redactValue(child, secret)
		}
		return t
	case string:
		if t == "" {
			return t
		}
		if secret || strings.Contains(t, "PRIVATE KEY") {
			return redactedValue
		}
		return t
	case nil:
		return nil
	default:
		// numbers / bools under a secret key (e.g. "token": 12345) are still secrets.
		if secret {
			return redactedValue
		}
		return t
	}
}
