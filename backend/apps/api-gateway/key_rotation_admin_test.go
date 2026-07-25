package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestRotateKeyHandler(t *testing.T) {
	km := NewKeyManager()
	oldKid, _ := km.GetCurrent()
	req := httptest.NewRequest("POST", "/api/keys/rotate", nil)
	w := httptest.NewRecorder()
	km.RotateKeyHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("rotate handler status=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["kid"] == "" || resp["kid"] == oldKid {
		t.Fatalf("expected new kid different from old kid")
	}
}
