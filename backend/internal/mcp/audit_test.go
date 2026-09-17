package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRedactArgs_SensitiveKeys(t *testing.T) {
	raw := json.RawMessage(`{"password":"s3cret","token":"abc","bo_key":"order"}`)
	out := redactArgs("get_bo_schema", raw)
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["password"] != redactedSentinel || m["token"] != redactedSentinel {
		t.Fatalf("expected redaction: %#v", m)
	}
	if m["bo_key"] != "order" {
		t.Fatalf("bo_key should survive: %#v", m)
	}
}

func TestRedactArgs_PromptTruncate(t *testing.T) {
	long := strings.Repeat("字", AuditPromptMaxRunes+50)
	raw, _ := json.Marshal(map[string]string{"prompt": long})
	out := redactArgs("text_to_semantic_ast", raw)
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	prompt, _ := m["prompt"].(string)
	if utf8.RuneCountInString(prompt) != AuditPromptMaxRunes {
		t.Fatalf("prompt runes=%d want %d", utf8.RuneCountInString(prompt), AuditPromptMaxRunes)
	}
	if int(m["prompt_len"].(float64)) != AuditPromptMaxRunes+50 {
		t.Fatalf("prompt_len=%v", m["prompt_len"])
	}
}

func TestTruncateRunes(t *testing.T) {
	if truncateRunes("abc", 10) != "abc" {
		t.Fatal("short")
	}
	if truncateRunes("abcdef", 3) != "abc" {
		t.Fatal("ascii")
	}
}
