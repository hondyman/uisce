package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestKeyManager_RotateAndJWKS(t *testing.T) {
	km := NewKeyManager()
	oldKid, _ := km.GetCurrent()
	// rotate to a new key
	newKid, err := km.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}
	if newKid == oldKid {
		t.Fatalf("expected new kid different from old kid")
	}

	// JWKS handler should return JSON with keys
	req := httptest.NewRequest("GET", "/jwks.json", nil)
	w := httptest.NewRecorder()
	km.JWKSHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("jwks handler status=%d", w.Code)
	}
	var payload map[string][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid jwks json: %v", err)
	}
	if len(payload["keys"]) == 0 {
		t.Fatalf("expected at least one jwk in keys")
	}
}

func TestKeyManager_SignAndVerifyRS256(t *testing.T) {
	km := NewKeyManager()
	kid, _ := km.GetCurrent()
	claims := jwt.MapClaims{"foo": "bar", "exp": time.Now().Add(1 * time.Hour).Unix(), "jti": "jti-test"}
	signed, kid2, err := km.SignTokenRS256(claims)
	if err != nil {
		t.Fatalf("SignTokenRS256 failed: %v", err)
	}
	if kid != kid2 {
		t.Fatalf("expected same kid returned")
	}

	// Parse token and verify using km public key
	tok, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			t.Fatalf("unexpected alg: %v", token.Header["alg"])
		}
		k, _ := token.Header["kid"].(string)
		pub, ok := km.GetPublicKey(k)
		if !ok {
			t.Fatalf("public key not found for kid=%s", k)
		}
		return pub, nil
	})
	if err != nil || !tok.Valid {
		t.Fatalf("failed to parse/verify signed token: %v", err)
	}
}
