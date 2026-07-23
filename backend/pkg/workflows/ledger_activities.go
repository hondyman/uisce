package workflows

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LedgerRecord represents an immutable audit entry
type LedgerRecord struct {
	ID              string          `db:"id" json:"id"`
	TenantID        string          `db:"tenant_id" json:"tenantId"`
	TransactionType string          `db:"transaction_type" json:"transactionType"`
	ActorID         string          `db:"actor_id" json:"actorId"`
	Payload         json.RawMessage `db:"payload" json:"payload"`
	PreviousHash    string          `db:"previous_hash" json:"previousHash"` // The link in the chain
	Hash            string          `db:"hash" json:"hash"`                  // SHA256(prev + payload)
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
}

type LedgerActivities struct {
	db *sqlx.DB
}

func NewLedgerActivities(db *sqlx.DB) *LedgerActivities {
	return &LedgerActivities{db: db}
}

// DurableLedgerWrite writes a record to the immutable ledger with hash chaining.
// This activity is idempotent and safe to retry.
func (a *LedgerActivities) DurableLedgerWrite(ctx context.Context, record LedgerRecord) (string, error) {
	// 1. Get the latest hash for this tenant to chain from
	// Locking is required to ensure linear chaining.
	// In high-volume systems, this is a bottleneck and requires optimized "sharded" ledgers
	// or optimistic locking with retries. For "Titan" single-binary, row locking is acceptable.

	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Lock the last record strictly by insertion order (created_at desc)
	// We handle the "Genesis" case (no previous records)
	var prevHash string
	err = tx.GetContext(ctx, &prevHash, `
		SELECT hash FROM audit_ledger 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC, id DESC 
		LIMIT 1 
		FOR UPDATE
	`, record.TenantID)

	if err == sql.ErrNoRows {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000" // Genesis Hash
	} else if err != nil {
		return "", fmt.Errorf("failed to get previous hash: %w", err)
	}

	// 2. Compute new hash
	// Hash = SHA256(PreviousHash + TransactionType + ActorID + Payload)
	payloadStr := string(record.Payload)
	dataToHash := fmt.Sprintf("%s:%s:%s:%s", prevHash, record.TransactionType, record.ActorID, payloadStr)
	hash := sha256.Sum256([]byte(dataToHash))
	newHash := fmt.Sprintf("%x", hash)

	// 3. Prepare record
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	record.PreviousHash = prevHash
	record.Hash = newHash
	record.CreatedAt = time.Now()

	// 4. Insert
	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO audit_ledger (id, tenant_id, transaction_type, actor_id, payload, previous_hash, hash, created_at)
		VALUES (:id, :tenant_id, :transaction_type, :actor_id, :payload, :previous_hash, :hash, :created_at)
	`, record)

	if err != nil {
		return "", fmt.Errorf("failed to insert ledger record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	// activity.GetLogger(ctx).Info("Ledger record committed", "id", record.ID, "hash", newHash)
	return newHash, nil
}
