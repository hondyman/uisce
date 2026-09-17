package manager

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenVault_GenerateAndConsume(t *testing.T) {
	vault := NewTokenVault(1 * time.Second)
	defer vault.Stop()

	payload := "mock_jwt_payload_12345"
	token, err := vault.GenerateToken(payload)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	gotPayload, ok := vault.ConsumeToken(token)
	if !ok {
		t.Fatalf("ConsumeToken returned false on first read")
	}
	if gotPayload != payload {
		t.Fatalf("expected payload %q, got %q", payload, gotPayload)
	}
}

func TestTokenVault_SingleUseSemantics(t *testing.T) {
	vault := NewTokenVault(1 * time.Second)
	defer vault.Stop()

	token, _ := vault.GenerateToken("secret_token")

	// First consumption succeeds
	_, ok1 := vault.ConsumeToken(token)
	if !ok1 {
		t.Fatalf("expected first consumption to succeed")
	}

	// Second consumption fails
	_, ok2 := vault.ConsumeToken(token)
	if ok2 {
		t.Fatalf("expected second consumption to fail (single-use violated)")
	}

	// Third consumption also fails
	_, ok3 := vault.ConsumeToken(token)
	if ok3 {
		t.Fatalf("expected third consumption to fail")
	}
}

func TestTokenVault_Expiry(t *testing.T) {
	// 50ms TTL
	vault := NewTokenVault(50 * time.Millisecond)
	defer vault.Stop()

	token, _ := vault.GenerateToken("ephemeral_data")

	// Wait 80ms to exceed TTL
	time.Sleep(80 * time.Millisecond)

	_, ok := vault.ConsumeToken(token)
	if ok {
		t.Fatalf("expected expired token consumption to fail")
	}
}

func TestTokenVault_ConcurrentConsumeRace(t *testing.T) {
	vault := NewTokenVault(2 * time.Second)
	defer vault.Stop()

	token, _ := vault.GenerateToken("concurrent_target")

	const goroutines = 100
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	startBarrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier // wait for all goroutines to be ready

			_, ok := vault.ConsumeToken(token)
			if ok {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}()
	}

	// Release all goroutines simultaneously
	close(startBarrier)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful consumption, got %d", successCount)
	}
	if failCount != goroutines-1 {
		t.Fatalf("expected exactly %d failed consumptions, got %d", goroutines-1, failCount)
	}
}

func TestTokenVault_JanitorPurge(t *testing.T) {
	// Vault with very short TTL
	vault := NewTokenVault(40 * time.Millisecond)
	defer vault.Stop()

	for i := 0; i < 10; i++ {
		_, _ = vault.GenerateToken("val")
	}

	if vault.Count() != 10 {
		t.Fatalf("expected count 10, got %d", vault.Count())
	}

	// Wait for janitor to run and purge
	time.Sleep(120 * time.Millisecond)

	if vault.Count() != 0 {
		t.Fatalf("expected janitor to have purged expired tokens, remaining: %d", vault.Count())
	}
}

func TestTokenVault_StopIdempotent(t *testing.T) {
	vault := NewTokenVault(1 * time.Second)
	vault.Stop()
	// Calling Stop again should not panic
	vault.Stop()
}
