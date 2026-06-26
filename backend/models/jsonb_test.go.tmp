package models

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func gzipCompress(t *testing.T, b []byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(b); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}
	return buf.Bytes()
}

func TestJSONBPlainJSONMarshalUnmarshal(t *testing.T) {
	orig := []byte(`{"a":1}`)
	var j JSONB
	if err := j.Scan(orig); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	out, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !bytes.Equal(out, orig) {
		t.Fatalf("expected %s got %s", string(orig), string(out))
	}
}

func TestJSONBGzipRawScan(t *testing.T) {
	payload := []byte(`{"b":"x"}`)
	gz := gzipCompress(t, payload)

	var j JSONB
	if err := j.Scan(gz); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	out, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !bytes.Equal(out, payload) {
		t.Fatalf("expected %s got %s", string(payload), string(out))
	}
}

func TestJSONBQuotedBase64GzipMarshal(t *testing.T) {
	payload := []byte(`{"c":true}`)
	gz := gzipCompress(t, payload)
	b64 := base64.StdEncoding.EncodeToString(gz)

	quoted, _ := json.Marshal(b64)

	var j JSONB
	if err := j.Scan(quoted); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	out, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !bytes.Equal(out, payload) {
		t.Fatalf("expected decompressed %s got %s", string(payload), string(out))
	}
}
