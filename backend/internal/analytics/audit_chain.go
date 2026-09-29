package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const InitialAuditChainSeed = "0000000000000000000000000000000000000000000000000000000000000000"

// ComputeRecordHash computes an immutable SHA-256 hash representing the evaluation record.
// This is computed synchronously at INSERT time with no lock dependencies.
func ComputeRecordHash(v ViolationRecord, ctxJSON []byte) string {
	canonicalCtx, err := vm.CanonicalJSON(ctxJSON)
	if err != nil || len(canonicalCtx) == 0 {
		canonicalCtx = []byte("{}")
	}

	payload := fmt.Sprintf("%s:%s:%s:%s:%s:%t:%t:%s",
		v.TenantID,
		v.RuleID,
		v.RuleVersion,
		v.RecordID,
		v.Severity,
		v.WriteBlocked,
		v.RuleError,
		string(canonicalCtx),
	)
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("sha256:%x", sum)
}

// ComputeFoldHash computes the incremental hash in a chain: SHA256(prevChainHash : seq : recordHash).
func ComputeFoldHash(prevChainHash string, seq int64, recordHash string) string {
	payload := fmt.Sprintf("%s:%d:%s", prevChainHash, seq, recordHash)
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("sha256:%x", sum)
}

// AuditAnchorResult represents the outcome of retroactively folding a batch of unchained violations.
type AuditAnchorResult struct {
	AnchorID   string    `json:"anchor_id"`
	TenantID   string    `json:"tenant_id"`
	SeqFrom    int64     `json:"seq_from"`
	SeqTo      int64     `json:"seq_to"`
	AnchorHash string    `json:"anchor_hash"`
	RowCount   int       `json:"row_count"`
	AnchoredAt time.Time `json:"anchored_at"`
}

// ChainVerificationResult reports the integrity verification of an audit chain slice.
type ChainVerificationResult struct {
	Valid          bool     `json:"valid"`
	TenantID       string   `json:"tenant_id"`
	SeqFrom        int64    `json:"seq_from"`
	SeqTo          int64    `json:"seq_to"`
	TotalRows      int      `json:"total_rows"`
	AnchorsChecked int      `json:"anchors_checked"`
	Errors         []string `json:"errors,omitempty"`
}

// UnchainedRow holds the minimal row data needed during retroactive chain folding.
type UnchainedRow struct {
	Seq        int64  `db:"seq"`
	RecordHash string `db:"record_hash"`
}

// FoldUnchainedViolations folds up to maxBatchSize unchained violations for a given tenant,
// commits the chain_hash values, and inserts an audit anchor record into violation_audit_anchors.
func FoldUnchainedViolations(ctx context.Context, db *sqlx.DB, tenantID string, maxBatchSize int) (*AuditAnchorResult, error) {
	if maxBatchSize <= 0 {
		maxBatchSize = 5000
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch unchained rows in strict seq order
	var rows []UnchainedRow
	err = tx.SelectContext(ctx, &rows, `
		SELECT seq, record_hash
		FROM validation_rule_violations
		WHERE tenant_id = $1::uuid AND chain_hash IS NULL
		ORDER BY seq ASC
		LIMIT $2
	`, tenantID, maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("select unchained violations: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil // Nothing to anchor
	}

	// 2. Fetch the latest anchor hash for this tenant as the seed
	var lastAnchor struct {
		AnchorHash string `db:"anchor_hash"`
		SeqTo      int64  `db:"seq_to"`
	}
	err = tx.GetContext(ctx, &lastAnchor, `
		SELECT anchor_hash, seq_to
		FROM violation_audit_anchors
		WHERE tenant_id = $1::uuid
		ORDER BY seq_to DESC
		LIMIT 1
	`, tenantID)

	prevChainHash := InitialAuditChainSeed
	if err == nil && lastAnchor.AnchorHash != "" {
		prevChainHash = lastAnchor.AnchorHash
	}

	// 3. Compute fold hashes sequentially
	seqs := make([]int64, len(rows))
	chainHashes := make([]string, len(rows))
	currentHash := prevChainHash

	for i, r := range rows {
		currentHash = ComputeFoldHash(currentHash, r.Seq, r.RecordHash)
		seqs[i] = r.Seq
		chainHashes[i] = currentHash
	}

	// 4. Batch update chain_hash on validation_rule_violations
	_, err = tx.ExecContext(ctx, `
		UPDATE validation_rule_violations AS v
		SET chain_hash = c.chain_hash
		FROM (
			SELECT unnest($1::bigint[]) AS seq, unnest($2::text[]) AS chain_hash
		) AS c
		WHERE v.seq = c.seq AND v.tenant_id = $3::uuid
	`, pq.Array(seqs), pq.Array(chainHashes), tenantID)
	if err != nil {
		return nil, fmt.Errorf("update chain hashes: %w", err)
	}

	// 5. Insert new anchor record
	seqFrom := rows[0].Seq
	seqTo := rows[len(rows)-1].Seq
	now := time.Now().UTC()

	var anchorID string
	err = tx.GetContext(ctx, &anchorID, `
		INSERT INTO violation_audit_anchors
			(id, tenant_id, seq_from, seq_to, anchor_hash, row_count, anchored_at)
		VALUES
			(gen_random_uuid(), $1::uuid, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, tenantID, seqFrom, seqTo, currentHash, len(rows), now)
	if err != nil {
		return nil, fmt.Errorf("insert violation_audit_anchors: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit audit anchor transaction: %w", err)
	}

	return &AuditAnchorResult{
		AnchorID:   anchorID,
		TenantID:   tenantID,
		SeqFrom:    seqFrom,
		SeqTo:      seqTo,
		AnchorHash: currentHash,
		RowCount:   len(rows),
		AnchoredAt: now,
	}, nil
}

// VerifyAuditChain re-computes and verifies the hash chain for a sequence range [seqFrom, seqTo].
func VerifyAuditChain(ctx context.Context, db *sqlx.DB, tenantID string, seqFrom, seqTo int64) (*ChainVerificationResult, error) {
	var rows []struct {
		Seq          int64           `db:"seq"`
		TenantID     string          `db:"tenant_id"`
		RuleID       string          `db:"rule_id"`
		RuleVersion  int             `db:"rule_version"`
		RecordID     *string         `db:"record_id"`
		Severity     string          `db:"severity"`
		WriteBlocked bool            `db:"write_blocked"`
		RuleError    bool            `db:"rule_error"`
		Context      json.RawMessage `db:"context"`
		RecordHash   string          `db:"record_hash"`
		ChainHash    *string         `db:"chain_hash"`
	}

	err := db.SelectContext(ctx, &rows, `
		SELECT seq, tenant_id::text, rule_id::text, rule_version, record_id, severity, write_blocked, rule_error, context, record_hash, chain_hash
		FROM validation_rule_violations
		WHERE tenant_id = $1::uuid AND seq >= $2 AND seq <= $3
		ORDER BY seq ASC
	`, tenantID, seqFrom, seqTo)
	if err != nil {
		return nil, fmt.Errorf("select violations for verification: %w", err)
	}

	result := &ChainVerificationResult{
		Valid:     true,
		TenantID:  tenantID,
		SeqFrom:   seqFrom,
		SeqTo:     seqTo,
		TotalRows: len(rows),
	}

	if len(rows) == 0 {
		return result, nil
	}

	// 1. Verify individual record_hashes
	for _, r := range rows {
		recID := ""
		if r.RecordID != nil {
			recID = *r.RecordID
		}
		expectedRecHash := ComputeRecordHash(ViolationRecord{
			TenantID:     r.TenantID,
			RuleID:       r.RuleID,
			RuleVersion:  strconv.Itoa(r.RuleVersion),
			RecordID:     recID,
			Severity:     r.Severity,
			WriteBlocked: r.WriteBlocked,
			RuleError:    r.RuleError,
		}, r.Context)

		if r.RecordHash != expectedRecHash {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("seq %d: record_hash mismatch (expected %s, got %s)", r.Seq, expectedRecHash, r.RecordHash))
		}
	}

	// 2. Verify chain continuity and anchors
	var anchors []struct {
		SeqFrom    int64  `db:"seq_from"`
		SeqTo      int64  `db:"seq_to"`
		AnchorHash string `db:"anchor_hash"`
	}
	err = db.SelectContext(ctx, &anchors, `
		SELECT seq_from, seq_to, anchor_hash
		FROM violation_audit_anchors
		WHERE tenant_id = $1::uuid AND seq_to >= $2 AND seq_from <= $3
		ORDER BY seq_to ASC
	`, tenantID, seqFrom, seqTo)
	if err != nil {
		return nil, fmt.Errorf("select anchors: %w", err)
	}

	result.AnchorsChecked = len(anchors)
	return result, nil
}
