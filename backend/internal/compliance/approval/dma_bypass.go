package approval

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBypassTokenNotFound  = errors.New("dma_bypass: token not found")
	ErrBypassTokenExpired   = errors.New("dma_bypass: token has expired")
	ErrBypassTokenUsed      = errors.New("dma_bypass: token has already been consumed (single-use only)")
	ErrBypassTokenScopeFail = errors.New("dma_bypass: token scope mismatch (tenant/account/rule)")
	ErrBypassTokenInvalidSig = errors.New("dma_bypass: cryptographic signature verification failed")
)

// DMABypassToken represents a single-use pre-authorized bypass certificate
type DMABypassToken struct {
	TokenID      uuid.UUID  `json:"token_id"`
	KeyID        string     `json:"key_id"` // Key version used for HMAC signing
	TenantID     uuid.UUID  `json:"tenant_id"`
	AccountID    uuid.UUID  `json:"account_id"`
	RuleID       uuid.UUID  `json:"rule_id"`
	AuthorizedBy string     `json:"authorized_by"` // Named compliance officer
	Reason       string     `json:"reason"`
	IssuedAt     time.Time  `json:"issued_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	Used         bool       `json:"used"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	Signature    string     `json:"signature"`
}

// DMABypassAuditLog captures every token validation attempt
type DMABypassAuditLog struct {
	AttemptID   uuid.UUID `json:"attempt_id"`
	TokenID     uuid.UUID `json:"token_id"`
	KeyID       string    `json:"key_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	AccountID   uuid.UUID `json:"account_id"`
	RuleID      uuid.UUID `json:"rule_id"`
	Success     bool      `json:"success"`
	FailureCode string    `json:"failure_code,omitempty"`
	AttemptedAt time.Time `json:"attempted_at"`
}

// DurableBypassAuditLogger writes bypass audit events to immutable persistent storage
type DurableBypassAuditLogger interface {
	LogBypassAttempt(ctx context.Context, log DMABypassAuditLog) error
}

// DMABypassManager handles issuing, validating, and atomically consuming DMA bypass tokens with key rotation
type DMABypassManager struct {
	mu           sync.Mutex
	activeKeyID  string
	keyRing      map[string][]byte // keyID -> secretKey
	tokens       map[uuid.UUID]*DMABypassToken
	auditLogs    []DMABypassAuditLog
	durableLogger DurableBypassAuditLogger
}

// NewDMABypassManager creates a new bypass token manager with initial active key
func NewDMABypassManager(activeKeyID string, secretKey []byte) *DMABypassManager {
	if activeKeyID == "" {
		activeKeyID = "v1"
	}
	if len(secretKey) == 0 {
		secretKey = []byte("default-compliance-dma-secret-key-32b")
	}

	keys := make(map[string][]byte)
	keys[activeKeyID] = secretKey

	return &DMABypassManager{
		activeKeyID: activeKeyID,
		keyRing:     keys,
		tokens:      make(map[uuid.UUID]*DMABypassToken),
		auditLogs:   make([]DMABypassAuditLog, 0, 128),
	}
}

// SetDurableLogger attaches an external durable audit sink (Postgres/Redpanda)
func (m *DMABypassManager) SetDurableLogger(logger DurableBypassAuditLogger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durableLogger = logger
}

// AddKey registers an additional key into the key ring for graceful rotation
func (m *DMABypassManager) AddKey(keyID string, key []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keyRing[keyID] = key
}

// SetActiveKey promotes a registered key ID as the active signing key
func (m *DMABypassManager) SetActiveKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.keyRing[keyID]; !exists {
		return fmt.Errorf("key id %q not found in key ring", keyID)
	}
	m.activeKeyID = keyID
	return nil
}

// ComputeSignature calculates the HMAC-SHA256 signature for a token using a specific key
func (m *DMABypassManager) ComputeSignature(keyID string, tokenID, tenantID, accountID, ruleID uuid.UUID, expiresAt time.Time) (string, error) {
	key, exists := m.keyRing[keyID]
	if !exists {
		return "", fmt.Errorf("unknown key id: %s", keyID)
	}
	mac := hmac.New(sha256.New, key)
	payload := fmt.Sprintf("%s:%s:%s:%s:%s:%d",
		keyID,
		tokenID.String(),
		tenantID.String(),
		accountID.String(),
		ruleID.String(),
		expiresAt.Unix(),
	)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// IssueToken mints a cryptographically signed, single-use bypass certificate
func (m *DMABypassManager) IssueToken(
	tenantID, accountID, ruleID uuid.UUID,
	authorizedBy, reason string,
	ttl time.Duration,
) *DMABypassToken {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	expiresAt := now.Add(ttl)

	tokenID := uuid.New()
	keyID := m.activeKeyID
	sig, _ := m.ComputeSignature(keyID, tokenID, tenantID, accountID, ruleID, expiresAt)

	token := &DMABypassToken{
		TokenID:      tokenID,
		KeyID:        keyID,
		TenantID:     tenantID,
		AccountID:    accountID,
		RuleID:       ruleID,
		AuthorizedBy: authorizedBy,
		Reason:       reason,
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
		Used:         false,
		Signature:    sig,
	}

	m.tokens[tokenID] = token
	return token
}

// ValidateAndConsume atomically verifies and expends a bypass token
func (m *DMABypassManager) ValidateAndConsume(
	ctx context.Context,
	tokenID, tenantID, accountID, ruleID uuid.UUID,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	audit := DMABypassAuditLog{
		AttemptID:   uuid.New(),
		TokenID:     tokenID,
		TenantID:    tenantID,
		AccountID:   accountID,
		RuleID:      ruleID,
		AttemptedAt: now,
	}

	recordAudit := func(success bool, failureCode string, err error) error {
		audit.Success = success
		audit.FailureCode = failureCode
		m.auditLogs = append(m.auditLogs, audit)
		if m.durableLogger != nil {
			_ = m.durableLogger.LogBypassAttempt(ctx, audit)
		}
		return err
	}

	token, exists := m.tokens[tokenID]
	if !exists {
		return recordAudit(false, "TOKEN_NOT_FOUND", ErrBypassTokenNotFound)
	}
	audit.KeyID = token.KeyID

	// Verify cryptographic signature with the token's keyID
	expectedSig, err := m.ComputeSignature(token.KeyID, token.TokenID, token.TenantID, token.AccountID, token.RuleID, token.ExpiresAt)
	if err != nil || !hmac.Equal([]byte(token.Signature), []byte(expectedSig)) {
		return recordAudit(false, "INVALID_SIGNATURE", ErrBypassTokenInvalidSig)
	}

	// Verify Scope (tenant, account, rule)
	if token.TenantID != tenantID || token.AccountID != accountID || token.RuleID != ruleID {
		return recordAudit(false, "SCOPE_MISMATCH", ErrBypassTokenScopeFail)
	}

	// Verify not expired
	if now.After(token.ExpiresAt) {
		return recordAudit(false, "TOKEN_EXPIRED", ErrBypassTokenExpired)
	}

	// Verify single use (not already used)
	if token.Used {
		return recordAudit(false, "ALREADY_CONSUMED", ErrBypassTokenUsed)
	}

	// Atomically consume token
	token.Used = true
	token.UsedAt = &now

	return recordAudit(true, "", nil)
}

// GetAuditLogs returns a copy of recorded bypass attempts
func (m *DMABypassManager) GetAuditLogs() []DMABypassAuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := make([]DMABypassAuditLog, len(m.auditLogs))
	copy(res, m.auditLogs)
	return res
}
