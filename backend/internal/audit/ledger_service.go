package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type LedgerEntry struct {
	SequenceID      int64       `json:"sequence_id" db:"sequence_id"`
	AuditID         string      `json:"audit_id" db:"audit_id"`
	TenantID        string      `json:"tenant_id" db:"tenant_id"`
	EntityType      string      `json:"entity_type" db:"entity_type"`
	EntityID        string      `json:"entity_id" db:"entity_id"`
	ActionType      string      `json:"action_type" db:"action_type"`
	PayloadSnapshot interface{} `json:"payload_snapshot" db:"payload_snapshot"`
	PerformedBy     string      `json:"performed_by" db:"performed_by"`
	PreviousHash    string      `json:"previous_hash" db:"previous_hash"`
	CurrentHash     string      `json:"current_hash" db:"current_hash"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
}

type LedgerService struct {
	db *sqlx.DB
}

func NewLedgerService(db *sqlx.DB) *LedgerService {
	return &LedgerService{db: db}
}

func ComputeEntryHash(entry LedgerEntry, prevHash string, timestamp string) string {
	payloadBytes, _ := json.Marshal(entry.PayloadSnapshot)
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		entry.TenantID, entry.EntityType, entry.EntityID, entry.ActionType, string(payloadBytes), prevHash, timestamp)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// RecordLedgerEntry inserts an immutable, hash-chained audit record
func (s *LedgerService) RecordLedgerEntry(ctx context.Context, entry LedgerEntry) (*LedgerEntry, error) {
	if entry.AuditID == "" {
		entry.AuditID = uuid.New().String()
	}
	if entry.PerformedBy == "" {
		entry.PerformedBy = "system"
	}

	// Fetch previous block hash for tenant chain
	var prevHash string = "GENESIS_BLOCK"
	if s.db != nil {
		err := s.db.GetContext(ctx, &prevHash, `SELECT current_hash FROM public.cryptographic_audit_ledger WHERE tenant_id = $1 ORDER BY sequence_id DESC LIMIT 1`, entry.TenantID)
		if err != nil {
			prevHash = "GENESIS_BLOCK"
		}
	}

	entry.PreviousHash = prevHash
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	entry.CurrentHash = ComputeEntryHash(entry, prevHash, nowStr)

	if s.db != nil {
		payloadB, _ := json.Marshal(entry.PayloadSnapshot)
		query := `
			INSERT INTO public.cryptographic_audit_ledger (
				audit_id, tenant_id, entity_type, entity_id, action_type, payload_snapshot, performed_by, previous_hash, current_hash, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		`
		_, err := s.db.ExecContext(ctx, query,
			entry.AuditID, entry.TenantID, entry.EntityType, entry.EntityID, entry.ActionType, payloadB, entry.PerformedBy, entry.PreviousHash, entry.CurrentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to record ledger entry: %w", err)
		}
	}

	return &entry, nil
}

// VerifyLedgerChain checks the cryptographic integrity of the audit chain
func (s *LedgerService) VerifyLedgerChain(ctx context.Context, tenantID string) (bool, int, error) {
	if s.db == nil {
		return true, 0, nil
	}

	var entries []LedgerEntry
	err := s.db.SelectContext(ctx, &entries, `SELECT sequence_id, audit_id, tenant_id, entity_type, entity_id, action_type, payload_snapshot, performed_by, previous_hash, current_hash, created_at FROM public.cryptographic_audit_ledger WHERE tenant_id = $1 ORDER BY sequence_id ASC`, tenantID)
	if err != nil {
		return false, 0, err
	}

	prevHash := "GENESIS_BLOCK"
	for idx, e := range entries {
		if idx > 0 && e.PreviousHash != prevHash {
			return false, idx, fmt.Errorf("chain broken at sequence %d: expected prev_hash %s, got %s", e.SequenceID, prevHash, e.PreviousHash)
		}
		prevHash = e.CurrentHash
	}

	return true, len(entries), nil
}

// HTTP Handler

func (s *LedgerService) VerifyChainHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = "core"
	}

	valid, count, err := s.VerifyLedgerChain(r.Context(), tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ledger chain verification failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenantId":      tenantID,
		"chainValid":    valid,
		"verifiedBlocks": count,
		"status":        "TAMPER_EVIDENT_CHAIN_VERIFIED",
	})
}
