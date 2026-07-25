package main

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryRevocationStore_RevokeAndIsRevoked(t *testing.T) {
	s := NewInMemoryRevocationStore()
	jti := "jti-123"
	exp := time.Now().Add(1 * time.Hour)
	if err := s.Revoke(context.Background(), jti, exp); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	rev, err := s.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("IsRevoked error: %v", err)
	}
	if !rev {
		t.Fatalf("expected revoked true")
	}
}

func TestRedisRevocationStore_RevokeAndIsRevoked(t *testing.T) {
	// This test runs only when REVOCATION_REDIS_ADDR is present in the environment.
	addr := ""
	if a := getEnv("REVOCATION_REDIS_ADDR", ""); a != "" {
		addr = a
	} else {
		t.Skip("REVOCATION_REDIS_ADDR not configured; skipping Redis-backed test")
	}
	r := NewRedisRevocationStore(addr)
	jti := "jti-redis-123"
	exp := time.Now().Add(1 * time.Hour)
	if err := r.Revoke(context.Background(), jti, exp); err != nil {
		t.Fatalf("Redis Revoke failed: %v", err)
	}
	rev, err := r.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("Redis IsRevoked error: %v", err)
	}
	if !rev {
		t.Fatalf("expected redis revoked true")
	}
}
