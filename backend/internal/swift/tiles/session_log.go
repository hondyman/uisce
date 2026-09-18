// Package tiles/session_log provides a pre-pipeline tile for SWIFT session
// audit logging and UETR retransmission deduplication.
//
// Architecture note: As of Sept 2026, the primary session-log path is in
// swift.Adapter.logSession() (Layer 1 / gateway), which runs independently of
// the pipeline DAG engine. This tile provides the same function for when the
// pipeline DAG engine is live (post nifty-greider-015b86 merge) and the DAG
// JSON includes "swift_session_log" at position 0.
//
// Both paths share the same idempotency contract:
//   - 23505 on (tenant_id, uetr) = retransmission → discard, no downstream work
//   - Any other insert error = log warning, pass record downstream (audit gap, non-fatal)
package tiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	pgUniqueViolation  = "23505"
	uetrUniqConstraint = "swift_session_log_uetr_tenant_uniq"
)

// NewSessionLogTile returns a tile that writes an append-only row to
// vend.swift_session_log. Must be the first tile in the pipeline DAG — before
// swift_decode — so retransmissions are discarded before any parsing work runs.
//
// Retransmission guard: 23505 on (tenant_id, uetr) → log + discard (record
// omitted from output slice). Empty output from this tile signals to the DAG
// engine that no downstream work should run for this record.
//
// IMPORTANT: verify the pipeline engine treats "tile output is empty" as
// terminal (not as "continue with stale record") before relying on this
// for correctness. See HANDOFF_SWIFT_SETTLEMENT.md §Engine Semantics.
func NewSessionLogTile(db *sql.DB) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		tctx := TenantFromContext(ctx)
		if tctx.TenantID == "" {
			return nil, nil, fmt.Errorf("session_log: no tenant in context; refusing to write to vend.swift_session_log without tenant scope")
		}

		out := make([]Record, 0, len(records))
		var errs []string

		for _, rec := range records {
			rawBytes, _ := rec["raw_bytes"].([]byte)
			uetr, _ := rec["uetr"].(string)
			msgType, _ := rec["msg_type"].(string)
			txRef, _ := rec["transaction_ref"].(string)
			bicSender, _ := rec["bic_sender"].(string)
			bicReceiver, _ := rec["bic_receiver"].(string)

			var uetrParam interface{}
			if uetr != "" {
				uetrParam = uetr
			}

			_, err := db.ExecContext(ctx, `
				INSERT INTO vend.swift_session_log (
					id, tenant_id, uetr, msg_type, transaction_ref,
					bic_sender, bic_receiver, raw_bytes, received_at, event_type
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'INBOUND')
			`,
				uuid.New().String(), tctx.TenantID, uetrParam, msgType, txRef,
				bicSender, bicReceiver, rawBytes, time.Now().UTC(),
			)
			if err != nil {
				if tileIsUETRDuplicate(err) && uetr != "" {
					log.Printf("[SWIFT] session_log tile: UETR retransmission discarded "+
						"tenant=%s uetr=%s msg_type=%s tx_ref=%s",
						tctx.TenantID, uetr, msgType, txRef)
					continue // discard — not appended to out
				}
				errs = append(errs, fmt.Sprintf("session_log: insert failed: %v", err))
				out = append(out, rec) // non-fatal: pass downstream despite log gap
				continue
			}
			out = append(out, rec)
		}

		return out, errs, nil
	}
}

// tileIsUETRDuplicate checks for 23505 on the UETR unique constraint.
// Uses lib/pq (*pq.Error) as the primary check since the tile DB uses the same
// connection as the rest of the app (lib/pq driver). Constraint-name string
// match is the belt-and-suspenders fallback.
// tileIsUETRDuplicate returns true only when err is a 23505 with an exact
// constraint-name match on the UETR index. Empty constraint name → false
// (treat as audit-gap, pass downstream). See pgerrors.go for the rationale:
// discarding on an ambiguous constraint is worse than double-processing.
func tileIsUETRDuplicate(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pgUniqueViolation &&
			pqErr.Constraint == uetrUniqConstraint
	}
	return strings.Contains(err.Error(), uetrUniqConstraint)
}
