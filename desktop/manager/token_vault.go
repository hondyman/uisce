package manager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type tokenEntry struct {
	payload   string
	expiresAt time.Time
}

// TokenVault stores ephemeral, single-use authentication tokens in Go memory.
// When a secondary desktop window is spawned, a token is issued and appended to
// the URL. The secondary window exchanges the token immediately on mount and
// the token is consumed (deleted) atomically.
type TokenVault struct {
	mu     sync.RWMutex
	tokens map[string]tokenEntry
	ttl    time.Duration
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewTokenVault creates a new TokenVault with the specified TTL and starts
// a background janitor goroutine to clean expired entries.
func NewTokenVault(ttl time.Duration) *TokenVault {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	vault := &TokenVault{
		tokens: make(map[string]tokenEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	go vault.janitor()
	return vault
}

// GenerateToken creates a cryptographically secure random token, stores the
// payload with an expiration time, and returns the token string.
func (v *TokenVault) GenerateToken(payload string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	token := hex.EncodeToString(bytes)

	v.mu.Lock()
	defer v.mu.Unlock()

	v.tokens[token] = tokenEntry{
		payload:   payload,
		expiresAt: time.Now().Add(v.ttl),
	}

	return token, nil
}

// ConsumeToken atomically retrieves and deletes a token. If the token is valid
// and not expired, it returns the payload and true. Any subsequent attempt to
// consume the same token returns false.
func (v *TokenVault) ConsumeToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	entry, exists := v.tokens[token]
	if !exists {
		return "", false
	}

	// Delete immediately to enforce single-use semantics
	delete(v.tokens, token)

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.payload, true
}

// Count returns the number of tokens currently stored in the vault.
func (v *TokenVault) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.tokens)
}

// Stop cleanly terminates the background janitor goroutine.
func (v *TokenVault) Stop() {
	select {
	case <-v.stopCh:
		// already stopped
		return
	default:
		close(v.stopCh)
		<-v.doneCh
	}
}

func (v *TokenVault) janitor() {
	defer close(v.doneCh)

	interval := v.ttl / 2
	if interval < 50*time.Millisecond {
		interval = 50 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-v.stopCh:
			return
		case <-ticker.C:
			v.purgeExpired()
		}
	}
}

func (v *TokenVault) purgeExpired() {
	now := time.Now()

	v.mu.Lock()
	defer v.mu.Unlock()

	for token, entry := range v.tokens {
		if now.After(entry.expiresAt) {
			delete(v.tokens, token)
		}
	}
}
