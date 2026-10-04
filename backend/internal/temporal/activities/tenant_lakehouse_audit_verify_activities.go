package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
)

// Verification of the Iceberg audit copy against alpha (ADR-035). Nothing that depends on the copy,
// such as dropping a hot partition, may rely on it until this has passed.
//
// What "passed" means: alpha's own chain recomputes cleanly, and every row of the copy is present in
// alpha, byte-for-byte equal in every field, and linked to the row before it. Rows alpha has beyond the
// copy's highest id are PENDING, not a finding: the copy runs every 15 minutes and is allowed to lag.
// A row alpha has BELOW the copy's highest id and the copy lacks is a finding, because the copy only
// ever appends in order, so a hole means data was lost or never written.
//
// The verifier reports and never repairs. Repair is the copy workflow's job, and it refuses to copy over
// a break (errTypeChainBreak), so a finding stays a finding until an operator decides what it means.

// Finding kinds.
const (
	FindingAlphaChainBroken = "alpha_chain_broken" // alpha's own recomputation fails; the comparison would prove nothing
	FindingMissingInCopy    = "missing_in_copy"    // alpha has the entry, the copy lacks it, and the copy is past it
	FindingExtraInCopy      = "extra_in_copy"      // the copy has an entry alpha does not
	FindingDiffers          = "differs"            // both have it and a field differs
	FindingChainBreak       = "chain_break"        // the copy's entry does not link to the one before it
)

// AuditFinding is the first thing found wrong. It names an entry and a field, never a value: audit
// payloads can hold personal data and the report is logged.
type AuditFinding struct {
	Kind   string
	ID     int64
	Detail string
}

// AuditVerifyCursor is where a verification has got to. It is carried between pages by the workflow, so a
// long chain is a series of short activities and a crash resumes at the cursor.
type AuditVerifyCursor struct {
	AfterID  int64
	PrevHash string // hash of the entry at AfterID; empty before the first entry
	Rows     int64  // entries verified so far
}

// AuditVerifyPage is one step's outcome.
type AuditVerifyPage struct {
	Cursor  AuditVerifyCursor
	Done    bool
	Finding *AuditFinding
}

// AuditVerifyReport is the whole run's outcome.
type AuditVerifyReport struct {
	TenantID string
	Verified bool // true only when there is no finding
	Rows     int64
	// ThroughID is the highest id verified. Entries alpha holds after it are not yet in the copy.
	ThroughID int64
	Finding   *AuditFinding
}

// AuditVerifyPageSize is how many copy rows one page reads.
const AuditVerifyPageSize = infra.MaxAppendRows

// VerifyAlphaAuditChain recomputes alpha's chain. A mismatch is reported as a finding, not an error: it
// is a result, and retrying cannot change it.
func (a *TenantLakehouseActivities) VerifyAlphaAuditChain(ctx context.Context, in LakehouseProvisionInput) (*AuditFinding, error) {
	id, err := in.tenant()
	if err != nil {
		return nil, err
	}
	broken, err := a.Registry.VerifyAudit(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("verify alpha's audit chain: %w", err)
	}
	if broken == nil {
		return nil, nil
	}
	return &AuditFinding{Kind: FindingAlphaChainBroken, ID: *broken, Detail: "alpha's own hash chain does not recompute at this entry"}, nil
}

// VerifyAuditPage compares the next page of the copy with alpha.
func (a *TenantLakehouseActivities) VerifyAuditPage(ctx context.Context, in LakehouseProvisionInput, cur AuditVerifyCursor, limit int) (AuditVerifyPage, error) {
	id, err := in.tenant()
	if err != nil {
		return AuditVerifyPage{}, err
	}
	if limit < 1 || limit > infra.MaxAppendRows {
		return AuditVerifyPage{}, nonRetryable(errTypeInvalidInput, fmt.Errorf("page size must be between 1 and %d", infra.MaxAppendRows))
	}
	copyRows, err := a.Destination.AuditRange(ctx, id, cur.AfterID, limit)
	if err != nil {
		return AuditVerifyPage{}, infraErr("read the audit copy", err)
	}
	alphaRows, err := a.Registry.AuditAfter(ctx, id, cur.AfterID, limit)
	if err != nil {
		return AuditVerifyPage{}, fmt.Errorf("read audit entries: %w", err)
	}
	return compareAuditPage(cur, alphaRows, copyRows, limit), nil
}

// compareAuditPage is the whole judgement, kept free of I/O so every branch can be tested directly.
func compareAuditPage(cur AuditVerifyCursor, alpha []registry.AuditEntry, cp []infra.AuditRow, limit int) AuditVerifyPage {
	if len(cp) == 0 {
		// Nothing more in the copy. Whatever alpha still has is pending.
		return AuditVerifyPage{Cursor: cur, Done: true}
	}

	// Both pages are cut at `limit`, so each knows nothing past its own last id when full. Compare only
	// where both can speak: up to the copy's last id (anything alpha has beyond it is pending), and up to
	// alpha's last id when alpha's page was cut.
	horizon := cp[len(cp)-1].ID
	if len(alpha) == limit && alpha[len(alpha)-1].ID < horizon {
		horizon = alpha[len(alpha)-1].ID
	}

	prev := cur.PrevHash
	if prev == "" {
		prev = genesisHash
	}
	next := cur
	i, j := 0, 0
	for {
		var a *registry.AuditEntry
		var c *infra.AuditRow
		if i < len(alpha) && alpha[i].ID <= horizon {
			a = &alpha[i]
		}
		if j < len(cp) && cp[j].ID <= horizon {
			c = &cp[j]
		}
		if a == nil && c == nil {
			break
		}
		switch {
		case c == nil || (a != nil && a.ID < c.ID):
			return finding(next, FindingMissingInCopy, a.ID, "alpha has this entry; the copy does not, though it holds later ones")
		case a == nil || c.ID < a.ID:
			return finding(next, FindingExtraInCopy, c.ID, "the copy has this entry; alpha does not")
		}
		if field := differingField(*a, *c); field != "" {
			return finding(next, FindingDiffers, c.ID, "the copy's "+field+" differs from alpha's")
		}
		if c.PrevHash != prev {
			return finding(next, FindingChainBreak, c.ID, "prev_hash does not equal the hash of the entry before it")
		}
		prev = c.Hash
		next.AfterID, next.PrevHash = c.ID, c.Hash
		next.Rows++
		i++
		j++
	}
	// Done when the copy's page was not cut and every row of it has been compared.
	done := len(cp) < limit && next.AfterID == cp[len(cp)-1].ID
	return AuditVerifyPage{Cursor: next, Done: done}
}

func finding(cur AuditVerifyCursor, kind string, id int64, detail string) AuditVerifyPage {
	return AuditVerifyPage{Cursor: cur, Done: true, Finding: &AuditFinding{Kind: kind, ID: id, Detail: detail}}
}

// differingField names the first field that differs, or "". Timestamps are compared at the
// microsecond the database keeps; JSON is compared as data, because the copy stores text and key order
// is not meaning.
func differingField(a registry.AuditEntry, c infra.AuditRow) string {
	switch {
	case !a.At.UTC().Truncate(time.Microsecond).Equal(c.At.UTC().Truncate(time.Microsecond)):
		return "at"
	case a.ActorID != c.ActorID:
		return "actor_id"
	case a.ActorRole != c.ActorRole:
		return "actor_role"
	case a.Action != c.Action:
		return "action"
	case !sameJSON(a.Before, []byte(c.BeforeJSON)):
		return "before"
	case !sameJSON(a.After, []byte(c.AfterJSON)):
		return "after"
	case a.PrevHash != c.PrevHash:
		return "prev_hash"
	case a.Hash != c.Hash:
		return "hash"
	}
	return ""
}

func sameJSON(a, b []byte) bool {
	a, b = bytes.TrimSpace(a), bytes.TrimSpace(b)
	if isNothing(a) && isNothing(b) {
		return true
	}
	if bytes.Equal(a, b) {
		return true
	}
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false // unreadable on either side is a difference, never a pass
	}
	return reflect.DeepEqual(x, y)
}

func isNothing(b []byte) bool { return len(b) == 0 || string(b) == "null" }
