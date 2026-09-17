package auth

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWsTicketVault_IssueAndConsume(t *testing.T) {
	vault := NewWsTicketVault(DefaultWsTicketVaultConfig())
	defer vault.Close()

	ticket, expiresIn, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error issuing ticket: %v", err)
	}
	if ticket == "" {
		t.Fatal("expected non-empty ticket")
	}
	if expiresIn != 30 {
		t.Fatalf("expected expiresIn 30, got %d", expiresIn)
	}

	tenantID, userID, err := vault.ConsumeTicket(ticket)
	if err != nil {
		t.Fatalf("unexpected error consuming ticket: %v", err)
	}
	if tenantID != "tenant-1" || userID != "user-1" {
		t.Fatalf("unexpected claims: got (%s, %s), want (tenant-1, user-1)", tenantID, userID)
	}
}

func TestWsTicketVault_SingleUseSemantics(t *testing.T) {
	vault := NewWsTicketVault(DefaultWsTicketVaultConfig())
	defer vault.Close()

	ticket, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}

	// First consume succeeds
	if _, _, err := vault.ConsumeTicket(ticket); err != nil {
		t.Fatalf("first consume failed: %v", err)
	}

	// Second consume MUST fail
	_, _, err = vault.ConsumeTicket(ticket)
	if err != ErrTicketNotFoundOrConsumed {
		t.Fatalf("expected ErrTicketNotFoundOrConsumed on second consume, got %v", err)
	}
}

func TestWsTicketVault_ExpiryAndClockSkew(t *testing.T) {
	// TTL: 100ms, SkewGrace: 50ms (total window = 150ms)
	cfg := WsTicketVaultConfig{
		TTL:             100 * time.Millisecond,
		SkewGrace:       50 * time.Millisecond,
		MaxCapacity:     100,
		RateLimitWindow: 1 * time.Second,
		RateLimitMax:    10,
		JanitorInterval: 1 * time.Second,
	}
	vault := NewWsTicketVault(cfg)
	defer vault.Close()

	ticket, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}

	// Wait 70ms (within 100ms TTL) -> should succeed
	time.Sleep(70 * time.Millisecond)
	if _, _, err := vault.ConsumeTicket(ticket); err != nil {
		t.Fatalf("expected ticket valid at 70ms, got err: %v", err)
	}

	// Issue new ticket, wait 120ms (expired TTL 100ms, but within 100ms + 50ms skew grace) -> should succeed
	ticketGrace, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, _, err := vault.ConsumeTicket(ticketGrace); err != nil {
		t.Fatalf("expected ticket valid during skew grace at 120ms, got err: %v", err)
	}

	// Issue new ticket, wait 180ms (>150ms total window) -> should fail with ErrTicketExpired
	ticketExpired, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}
	time.Sleep(180 * time.Millisecond)
	_, _, err = vault.ConsumeTicket(ticketExpired)
	if err != ErrTicketExpired {
		t.Fatalf("expected ErrTicketExpired past skew grace, got %v", err)
	}

	// Verify that consuming an expired ticket also consumed (deleted) it from the store
	_, _, err = vault.ConsumeTicket(ticketExpired)
	if err != ErrTicketNotFoundOrConsumed {
		t.Fatalf("expected ErrTicketNotFoundOrConsumed on second attempt of expired ticket, got %v", err)
	}
}

func TestWsTicketVault_ConcurrentConsumeRace(t *testing.T) {
	vault := NewWsTicketVault(DefaultWsTicketVaultConfig())
	defer vault.Close()

	ticket, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}

	concurrency := 50
	var successCount int32
	var wg sync.WaitGroup
	startCh := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh
			_, _, err := vault.ConsumeTicket(ticket)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	close(startCh)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful consume in race, got %d", successCount)
	}
}

func TestWsTicketVault_RateLimitingAndWindowRecovery(t *testing.T) {
	cfg := WsTicketVaultConfig{
		TTL:             10 * time.Second,
		SkewGrace:       2 * time.Second,
		MaxCapacity:     100,
		RateLimitWindow: 100 * time.Millisecond, // short window for fast test
		RateLimitMax:    5,                      // 5 tickets max in window
		JanitorInterval: 1 * time.Second,
	}
	vault := NewWsTicketVault(cfg)
	defer vault.Close()

	// Issue 5 tickets successfully
	for i := 0; i < 5; i++ {
		if _, _, err := vault.IssueTicket("tenant-1", "user-1"); err != nil {
			t.Fatalf("ticket %d should have succeeded, got %v", i+1, err)
		}
	}

	// 6th ticket in window MUST fail with ErrRateLimitExceeded
	_, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != ErrRateLimitExceeded {
		t.Fatalf("expected ErrRateLimitExceeded on 6th ticket, got %v", err)
	}

	// Different user should still be able to issue
	if _, _, err := vault.IssueTicket("tenant-1", "user-2"); err != nil {
		t.Fatalf("user-2 should have succeeded despite user-1 rate limit, got %v", err)
	}

	// Wait for window to roll over (120ms > 100ms)
	time.Sleep(120 * time.Millisecond)

	// User-1 should now be able to issue again
	if _, _, err := vault.IssueTicket("tenant-1", "user-1"); err != nil {
		t.Fatalf("expected issuance to succeed after window recovery, got %v", err)
	}
}

func TestWsTicketVault_CapacityCeiling(t *testing.T) {
	cfg := WsTicketVaultConfig{
		TTL:             10 * time.Second,
		SkewGrace:       2 * time.Second,
		MaxCapacity:     3, // small capacity for test
		RateLimitWindow: 10 * time.Second,
		RateLimitMax:    100,
		JanitorInterval: 1 * time.Second,
	}
	vault := NewWsTicketVault(cfg)
	defer vault.Close()

	for i := 0; i < 3; i++ {
		if _, _, err := vault.IssueTicket("tenant-1", "user-1"); err != nil {
			t.Fatalf("expected ticket %d to succeed, got %v", i+1, err)
		}
	}

	if vault.Size() != 3 {
		t.Fatalf("expected size 3, got %d", vault.Size())
	}

	// 4th ticket must exceed capacity
	_, _, err := vault.IssueTicket("tenant-1", "user-1")
	if err != ErrVaultCapacityExceeded {
		t.Fatalf("expected ErrVaultCapacityExceeded, got %v", err)
	}
}

func TestWsTicketVault_JanitorSweeper(t *testing.T) {
	cfg := WsTicketVaultConfig{
		TTL:             50 * time.Millisecond,
		SkewGrace:       20 * time.Millisecond,
		MaxCapacity:     100,
		RateLimitWindow: 1 * time.Second,
		RateLimitMax:    10,
		JanitorInterval: 50 * time.Millisecond, // rapid sweeps
	}
	vault := NewWsTicketVault(cfg)
	defer vault.Close()

	if _, _, err := vault.IssueTicket("tenant-1", "user-1"); err != nil {
		t.Fatalf("issue ticket failed: %v", err)
	}
	if vault.Size() != 1 {
		t.Fatalf("expected size 1, got %d", vault.Size())
	}

	// Wait for TTL + SkewGrace + JanitorInterval (150ms > 50+20+50)
	time.Sleep(150 * time.Millisecond)

	if vault.Size() != 0 {
		t.Fatalf("expected janitor to sweep expired ticket to size 0, got %d", vault.Size())
	}
}
