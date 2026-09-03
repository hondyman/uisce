package semantic_bridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Ledger appends tamper-evident rows to catalog_ai.ai_bridge_sync_logs.
//
// Each row's hmac_signature is HMAC-SHA256(serverKey, prev_hmac || payload_hash)
// where prev_hmac is the previous row's hmac_signature for that tenant. That
// makes the log a hash chain: recomputing any row's signature requires both
// the server's secret key and the (unbroken) chain of everything before it,
// so editing or deleting a row breaks verification for every row after it —
// unlike the plain SHA-256(payload) checksum this replaces, which anyone
// with DB access could recompute and reinsert unnoticed.
type Ledger struct {
	db      *sqlx.DB
	hmacKey []byte
}

func NewLedger(db *sqlx.DB, hmacKey []byte) *Ledger {
	return &Ledger{db: db, hmacKey: hmacKey}
}

// signChainEntry computes HMAC-SHA256(key, prevHMAC||payloadHash) — the pure
// math behind each ledger row's tamper-evident signature. Factored out of
// Append/Verify so it can be unit-tested directly against fabricated rows,
// without a live Postgres connection.
func signChainEntry(key []byte, prevHMAC, payloadHash string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(prevHMAC))
	mac.Write([]byte(payloadHash))
	return hex.EncodeToString(mac.Sum(nil))
}

// ledgerRow is the subset of a catalog_ai.ai_bridge_sync_logs row needed to
// verify one link in the hash chain.
type ledgerRow struct {
	ID          uuid.UUID
	PayloadHash string
	PrevHMAC    string
	HMACSig     string
}

// verifyChain walks rows in order and confirms every hmac_signature matches
// what it should be given the previous row and the key. Returns the id of
// the first row that fails, or uuid.Nil if the whole chain is intact.
func verifyChain(key []byte, rows []ledgerRow) uuid.UUID {
	expectedPrev := ""
	for _, r := range rows {
		want := signChainEntry(key, expectedPrev, r.PayloadHash)
		if r.PrevHMAC != expectedPrev || r.HMACSig != want {
			return r.ID
		}
		expectedPrev = r.HMACSig
	}
	return uuid.Nil
}

// Append hashes payload with SHA-256 for payload_hash (display/dedup), then
// signs prev_hmac||payload_hash with HMAC-SHA256 for hmac_signature, and
// inserts the row. Returns the inserted log id.
func (l *Ledger) Append(ctx context.Context, tenantID uuid.UUID, targetID *uuid.UUID, vendorType, action string, payload []byte, status string, httpStatus int, responseBody string, durationMs int) (uuid.UUID, error) {
	sum := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(sum[:])

	prevHMAC, err := l.lastHMAC(ctx, tenantID)
	if err != nil {
		return uuid.Nil, err
	}

	hmacSig := signChainEntry(l.hmacKey, prevHMAC, payloadHash)

	id := uuid.New()
	_, err = l.db.ExecContext(ctx, `
		INSERT INTO catalog_ai.ai_bridge_sync_logs (
			id, tenant_id, target_id, vendor_type, action, payload_hash, artifact_payload,
			status, http_status, response_body, execution_time_ms, prev_hmac, hmac_signature, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		id, tenantID, targetID, vendorType, action, payloadHash, string(payload),
		status, httpStatus, responseBody, durationMs, prevHMAC, hmacSig, time.Now(),
	)
	return id, err
}

func (l *Ledger) lastHMAC(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var prev string
	err := l.db.GetContext(ctx, &prev, `
		SELECT hmac_signature FROM catalog_ai.ai_bridge_sync_logs
		WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, tenantID)
	if err != nil {
		return "", nil // no prior row for this tenant — chain starts empty
	}
	return prev, nil
}

// Verify walks a tenant's chain in order and confirms every hmac_signature
// matches what it should be given the previous row and the server key.
// Returns the id of the first row that fails to verify, or uuid.Nil if the
// whole chain is intact.
func (l *Ledger) Verify(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, error) {
	var rows []struct {
		ID          uuid.UUID `db:"id"`
		PayloadHash string    `db:"payload_hash"`
		PrevHMAC    string    `db:"prev_hmac"`
		HMACSig     string    `db:"hmac_signature"`
	}
	err := l.db.SelectContext(ctx, &rows, `
		SELECT id, payload_hash, prev_hmac, hmac_signature FROM catalog_ai.ai_bridge_sync_logs
		WHERE tenant_id = $1 ORDER BY created_at ASC, id ASC`, tenantID)
	if err != nil {
		return uuid.Nil, err
	}

	converted := make([]ledgerRow, len(rows))
	for i, r := range rows {
		converted[i] = ledgerRow{ID: r.ID, PayloadHash: r.PayloadHash, PrevHMAC: r.PrevHMAC, HMACSig: r.HMACSig}
	}
	return verifyChain(l.hmacKey, converted), nil
}
