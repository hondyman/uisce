package analytics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestComputeRecordHash_Deterministic(t *testing.T) {
	v1 := ViolationRecord{
		TenantID:     "tenant-abc",
		RuleID:       "rule-123",
		RuleVersion:  "2",
		RecordID:     "rec-999",
		Severity:     "BLOCK",
		WriteBlocked: true,
		RuleError:    false,
	}
	ctxJSON1 := []byte(`{"a": 1, "b": "hello"}`)
	ctxJSON2 := []byte(`{"b": "hello", "a": 1}`) // key reordering

	hash1 := ComputeRecordHash(v1, ctxJSON1)
	hash2 := ComputeRecordHash(v1, ctxJSON2)

	assert.Equal(t, hash1, hash2, "ComputeRecordHash should produce identical hash regardless of JSON key order")
	assert.True(t, len(hash1) > 10)
}

func TestFoldUnchainedViolations_HappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New().String()

	// 1. Begin tx
	mock.ExpectBegin()

	// 2. Select unchained rows
	mock.ExpectQuery(`SELECT seq, record_hash FROM validation_rule_violations WHERE tenant_id = \$1::uuid AND chain_hash IS NULL`).
		WithArgs(tenantID, 5000).
		WillReturnRows(sqlmock.NewRows([]string{"seq", "record_hash"}).
			AddRow(1, "sha256:1111111111111111").
			AddRow(2, "sha256:2222222222222222"))

	// 3. Select last anchor
	mock.ExpectQuery(`SELECT anchor_hash, seq_to FROM violation_audit_anchors WHERE tenant_id = \$1::uuid`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"anchor_hash", "seq_to"})) // no previous anchor -> use InitialAuditChainSeed

	// 4. Update chain_hash
	mock.ExpectExec(`UPDATE validation_rule_violations AS v SET chain_hash = c\.chain_hash`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), tenantID).
		WillReturnResult(sqlmock.NewResult(2, 2))

	// 5. Insert into violation_audit_anchors
	mock.ExpectQuery(`INSERT INTO violation_audit_anchors`).
		WithArgs(tenantID, int64(1), int64(2), sqlmock.AnyArg(), 2, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))

	// 6. Commit
	mock.ExpectCommit()

	res, err := FoldUnchainedViolations(context.Background(), sqlxDB, tenantID, 5000)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(1), res.SeqFrom)
	assert.Equal(t, int64(2), res.SeqTo)
	assert.Equal(t, 2, res.RowCount)
	assert.NotEmpty(t, res.AnchorHash)
}

func TestVerifyAuditChain_TamperDetection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	tenantID := uuid.New().String()
	ruleID := uuid.New().String()

	v := ViolationRecord{
		TenantID:     tenantID,
		RuleID:       ruleID,
		RuleVersion:  "1",
		RecordID:     "rec-1",
		Severity:     "BLOCK",
		WriteBlocked: true,
		RuleError:    false,
	}
	ctxRaw := json.RawMessage(`{"field":"amount","val":500}`)
	ctxBytes, _ := json.Marshal(ctxRaw)
	correctHash := ComputeRecordHash(v, ctxBytes)

	// Tampered row with corrupted hash
	mock.ExpectQuery(`SELECT seq, tenant_id::text, rule_id::text, rule_version, record_id, severity, write_blocked, rule_error, context, record_hash, chain_hash FROM validation_rule_violations`).
		WithArgs(tenantID, int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"seq", "tenant_id", "rule_id", "rule_version", "record_id", "severity", "write_blocked", "rule_error", "context", "record_hash", "chain_hash",
		}).AddRow(
			int64(1), tenantID, ruleID, 1, "rec-1", "BLOCK", true, false, ctxBytes, "sha256:corrupted_fake_hash", "sha256:chain_1",
		))

	mock.ExpectQuery(`SELECT seq_from, seq_to, anchor_hash FROM violation_audit_anchors`).
		WithArgs(tenantID, int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"seq_from", "seq_to", "anchor_hash"}).
			AddRow(int64(1), int64(1), "sha256:chain_1"))

	res, err := VerifyAuditChain(context.Background(), sqlxDB, tenantID, 1, 1)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.False(t, res.Valid, "VerifyAuditChain must fail on tampered record_hash")
	assert.NotEmpty(t, res.Errors)
	assert.Contains(t, res.Errors[0], "record_hash mismatch")
	assert.Contains(t, res.Errors[0], correctHash)
}
