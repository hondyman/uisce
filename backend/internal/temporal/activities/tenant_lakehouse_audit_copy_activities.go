package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
)

// Activities for copying a tenant's lakehouse audit chain into an Iceberg table in the tenant's own
// warehouse (ADR-036). alpha remains the system of record; this adds an immutable, queryable copy.
//
// The property everything here protects: the copy RESUMES FROM THE DESTINATION. Every batch begins by
// asking the Iceberg table for its highest id and ships only what comes after it. So an append whose
// outcome is unknown (the commit succeeded, the reply was lost) is simply retried: the next attempt
// sees the rows are there and ships nothing. No counter kept anywhere else can drift from the data.

// AuditCopyBatchSize is how many entries one append carries. Under Object Lock every commit's files are
// kept for the whole retention period, so batches are large, and a run with nothing to ship commits
// nothing.
const AuditCopyBatchSize = 500

// genesisHash is the prev_hash of a tenant's first audit entry.
var genesisHash = strings.Repeat("0", 64)

// AuditCopyResult is what one batch did.
type AuditCopyResult struct {
	Copied    int   // entries appended in this batch
	ThroughID int64 // the highest id now in the copy
	Done      bool  // nothing further to ship after this batch
}

const errTypeChainBreak = "LakehouseAuditChainBreak"

// PrepareAuditCopy makes sure the tenant's catalog, audit database and events table exist. The tenant's
// own storage credential is read from the secrets store here and never returned.
func (a *TenantLakehouseActivities) PrepareAuditCopy(ctx context.Context, in LakehouseProvisionInput) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	cfg, err := a.Registry.Get(ctx, id)
	if errors.Is(err, registry.ErrTenantNotFound) {
		return nonRetryable(errTypeInvalidInput, err)
	}
	if err != nil {
		return fmt.Errorf("read lakehouse registry: %w", err)
	}
	if !cfg.Provisioned {
		return nonRetryable(errTypeNotReady, fmt.Errorf("%w: there is no warehouse to copy the audit into", registry.ErrInvalidState))
	}
	key, secret, err := a.Credentials.Read(ctx, id)
	if err != nil {
		return infraErr("read storage credential", err)
	}
	if err := a.Destination.EnsureAuditDestination(ctx, infra.AuditDestinationSpec{
		TenantID: id, TenantName: cfg.TenantName,
		WarehouseName: cfg.WarehouseName, AccessKeyID: key, SecretKey: secret,
	}); err != nil {
		return infraErr("prepare the audit destination", err)
	}
	return nil
}

// CopyAuditBatch ships the next batch of audit entries to the Iceberg copy.
func (a *TenantLakehouseActivities) CopyAuditBatch(ctx context.Context, in LakehouseProvisionInput, limit int) (AuditCopyResult, error) {
	id, err := in.tenant()
	if err != nil {
		return AuditCopyResult{}, err
	}
	if limit < 1 || limit > infra.MaxAppendRows {
		return AuditCopyResult{}, nonRetryable(errTypeInvalidInput, fmt.Errorf("batch size must be between 1 and %d", infra.MaxAppendRows))
	}

	// 1. The destination says where the copy has got to.
	through, err := a.Destination.MaxAuditID(ctx, id)
	if err != nil {
		return AuditCopyResult{}, infraErr("read the audit copy's highest id", err)
	}

	// 2. What alpha has after that, oldest first.
	entries, err := a.Registry.AuditAfter(ctx, id, through, limit)
	if err != nil {
		return AuditCopyResult{}, fmt.Errorf("read audit entries: %w", err)
	}
	if len(entries) == 0 {
		// Nothing to ship, so nothing is committed.
		if through > 0 {
			if err := a.Registry.MarkAuditCopied(ctx, id, through); err != nil {
				return AuditCopyResult{}, fmt.Errorf("record the copy's progress: %w", err)
			}
		}
		return AuditCopyResult{ThroughID: through, Done: true}, nil
	}

	// 3. Continuity: the first entry must link to what the copy already ends with, and the batch must
	// link to itself. A gap or a reordered chain is never copied over.
	wantPrev := genesisHash
	if through > 0 {
		h, ok, err := a.Destination.AuditHash(ctx, id, through)
		if err != nil {
			return AuditCopyResult{}, infraErr("read the last copied entry", err)
		}
		if !ok {
			return AuditCopyResult{}, nonRetryable(errTypeChainBreak,
				fmt.Errorf("the audit copy reports highest id %d but has no such entry", through))
		}
		wantPrev = h
	}
	for _, e := range entries {
		if e.PrevHash != wantPrev {
			return AuditCopyResult{}, nonRetryable(errTypeChainBreak,
				fmt.Errorf("audit entry %d does not link to the entry before it in the copy; not copying over a break", e.ID))
		}
		wantPrev = e.Hash
	}

	// 4. One append, one commit.
	rows := make([]infra.AuditRow, len(entries))
	for i, e := range entries {
		rows[i] = infra.AuditRow{
			ID: e.ID, At: e.At, ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action,
			BeforeJSON: string(e.Before), AfterJSON: string(e.After), PrevHash: e.PrevHash, Hash: e.Hash,
		}
	}
	if err := a.Destination.AppendAudit(ctx, id, rows); err != nil {
		// The outcome may be unknown. That is fine: a retry starts from the destination's highest id.
		return AuditCopyResult{}, infraErr("append to the audit copy", err)
	}

	// 5. Do not trust the write: read it back.
	last := entries[len(entries)-1].ID
	now, err := a.Destination.MaxAuditID(ctx, id)
	if err != nil {
		return AuditCopyResult{}, infraErr("verify the audit copy", err)
	}
	if now < last {
		return AuditCopyResult{}, fmt.Errorf("the audit copy's highest id is %d after appending through %d", now, last)
	}
	if err := a.Registry.MarkAuditCopied(ctx, id, now); err != nil {
		return AuditCopyResult{}, fmt.Errorf("record the copy's progress: %w", err)
	}
	return AuditCopyResult{Copied: len(entries), ThroughID: now, Done: len(entries) < limit}, nil
}

// ListProvisionedTenants returns the tenants that have a warehouse to copy into.
func (a *TenantLakehouseActivities) ListProvisionedTenants(ctx context.Context) ([]string, error) {
	ids, err := a.Registry.ProvisionedTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("list provisioned tenants: %w", err)
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out, nil
}
