package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrTicketNotFoundOrConsumed is returned when the ticket does not exist or has already been used.
	ErrTicketNotFoundOrConsumed = errors.New("ticket not found or already consumed")
	// ErrTicketExpired is returned when the ticket has exceeded its TTL and skew grace.
	ErrTicketExpired = errors.New("ticket has expired")
	// ErrRateLimitExceeded is returned when the user exceeds the ticket issuance burst limit.
	ErrRateLimitExceeded = errors.New("rate limit exceeded: max 10 tickets per 15 seconds")
	// ErrVaultCapacityExceeded is returned when the in-memory ticket vault reaches its hard capacity ceiling.
	ErrVaultCapacityExceeded = errors.New("vault capacity ceiling reached: cannot issue new tickets")
)

// TicketEntry represents a stored ephemeral WebSocket authentication ticket.
type TicketEntry struct {
	TenantID  string
	UserID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type userRateWindow struct {
	windowStart time.Time
	count       int
}

// WsTicketVaultConfig allows parameterization of vault limits.
type WsTicketVaultConfig struct {
	TTL              time.Duration
	SkewGrace        time.Duration
	MaxCapacity      int
	RateLimitWindow  time.Duration
	RateLimitMax     int
	JanitorInterval  time.Duration
}

// DefaultWsTicketVaultConfig returns production defaults:
// 30s TTL, 2s skew grace, 10k capacity, 10 tickets per 15s window, 15s janitor interval.
func DefaultWsTicketVaultConfig() WsTicketVaultConfig {
	return WsTicketVaultConfig{
		TTL:             30 * time.Second,
		SkewGrace:       2 * time.Second,
		MaxCapacity:     10000,
		RateLimitWindow: 15 * time.Second,
		RateLimitMax:    10,
		JanitorInterval: 15 * time.Second,
	}
}

// WsTicketVault manages ephemeral, single-use WebSocket tickets in memory.
// It enforces strictly single-use semantics, clock-skew tolerance, user rate limiting,
// capacity caps, and automatic background cleanup.
type WsTicketVault struct {
	mu          sync.Mutex
	config      WsTicketVaultConfig
	tickets     map[string]TicketEntry
	userRates   map[string]*userRateWindow
	stopCh      chan struct{}
	doneCh      chan struct{}
	stopped     bool
}

// NewWsTicketVault initializes and starts the background janitor.
func NewWsTicketVault(cfg WsTicketVaultConfig) *WsTicketVault {
	if cfg.TTL <= 0 {
		cfg.TTL = 30 * time.Second
	}
	if cfg.SkewGrace < 0 {
		cfg.SkewGrace = 2 * time.Second
	}
	if cfg.MaxCapacity <= 0 {
		cfg.MaxCapacity = 10000
	}
	if cfg.RateLimitWindow <= 0 {
		cfg.RateLimitWindow = 15 * time.Second
	}
	if cfg.RateLimitMax <= 0 {
		cfg.RateLimitMax = 10
	}
	if cfg.JanitorInterval <= 0 {
		cfg.JanitorInterval = 15 * time.Second
	}

	v := &WsTicketVault{
		config:    cfg,
		tickets:   make(map[string]TicketEntry),
		userRates: make(map[string]*userRateWindow),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}

	go v.janitor()
	return v
}

// IssueTicket generates a secure 256-bit base64url ticket bound to (tenantID, userID).
// It verifies the capacity cap and enforces the per-user rate limit (10 tickets / 15s).
func (v *WsTicketVault) IssueTicket(tenantID, userID string) (ticket string, expiresIn int, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.stopped {
		return "", 0, errors.New("vault is stopped")
	}

	now := time.Now()

	// 1. Capacity check
	if len(v.tickets) >= v.config.MaxCapacity {
		return "", 0, ErrVaultCapacityExceeded
	}

	// 2. Per-user rate limiting check
	rateKey := fmt.Sprintf("%s:%s", tenantID, userID)
	rw, exists := v.userRates[rateKey]
	if !exists || now.Sub(rw.windowStart) >= v.config.RateLimitWindow {
		v.userRates[rateKey] = &userRateWindow{
			windowStart: now,
			count:       1,
		}
	} else {
		if rw.count >= v.config.RateLimitMax {
			return "", 0, ErrRateLimitExceeded
		}
		rw.count++
	}

	// 3. Generate 256 bits (32 bytes) of cryptographic randomness
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", 0, fmt.Errorf("failed to generate secure ticket: %w", err)
	}
	ticket = base64.RawURLEncoding.EncodeToString(bytes)

	// 4. Store ticket entry
	v.tickets[ticket] = TicketEntry{
		TenantID:  tenantID,
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now.Add(v.config.TTL),
	}

	return ticket, int(v.config.TTL.Seconds()), nil
}

// ConsumeTicket atomically retrieves and deletes a ticket in a single critical section.
// If the ticket does not exist or was already consumed, ErrTicketNotFoundOrConsumed is returned.
// If the ticket has passed its expiration time plus skew grace, ErrTicketExpired is returned.
// In ALL cases (success or failure), if the ticket existed in the store, it is deleted immediately.
func (v *WsTicketVault) ConsumeTicket(ticket string) (tenantID, userID string, err error) {
	if ticket == "" {
		return "", "", ErrTicketNotFoundOrConsumed
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	entry, exists := v.tickets[ticket]
	if !exists {
		return "", "", ErrTicketNotFoundOrConsumed
	}

	// Atomic deletion immediately - ticket cannot be retried or replayed under any condition
	delete(v.tickets, ticket)

	now := time.Now()
	// Validation condition: now <= ExpiresAt + SkewGrace
	if now.After(entry.ExpiresAt.Add(v.config.SkewGrace)) {
		return "", "", ErrTicketExpired
	}

	return entry.TenantID, entry.UserID, nil
}

// Size returns the count of currently outstanding tickets.
func (v *WsTicketVault) Size() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.tickets)
}

// Close stops the janitor goroutine and cleans up resources.
func (v *WsTicketVault) Close() {
	v.mu.Lock()
	if v.stopped {
		v.mu.Unlock()
		return
	}
	v.stopped = true
	close(v.stopCh)
	v.mu.Unlock()

	<-v.doneCh
}

func (v *WsTicketVault) janitor() {
	defer close(v.doneCh)
	ticker := time.NewTicker(v.config.JanitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.stopCh:
			return
		case now := <-ticker.C:
			v.sweep(now)
		}
	}
}

func (v *WsTicketVault) sweep(now time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Purge expired tickets (beyond TTL + SkewGrace)
	for ticket, entry := range v.tickets {
		if now.After(entry.ExpiresAt.Add(v.config.SkewGrace)) {
			delete(v.tickets, ticket)
		}
	}

	// Purge stale rate limit windows
	for key, rw := range v.userRates {
		if now.Sub(rw.windowStart) >= v.config.RateLimitWindow*2 {
			delete(v.userRates, key)
		}
	}
}
