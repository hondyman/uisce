package semantic_bridge

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LedgerBreach describes a tenant whose sync-log hash chain failed
// verification — i.e. a row was edited, deleted, or reordered after it was
// written.
type LedgerBreach struct {
	TenantID    uuid.UUID
	FirstBadLog uuid.UUID
	DetectedAt  time.Time
}

// AlertFunc is how LedgerMonitor reports a breach — kept as a plain function
// type rather than depending on internal/observability directly, so this
// package doesn't need to know about Slack/PagerDuty/email wiring. Callers
// (e.g. cmd/server/main.go) pass in whatever dispatcher they've configured.
type AlertFunc func(ctx context.Context, breach LedgerBreach)

// LedgerMonitor periodically verifies every active tenant's tamper-evident
// sync-log chain (see ledger.go) and reports any breach. Without this,
// VerifyLedger existing as an on-demand endpoint means nobody actually
// checks it — a tamper-evident chain nobody watches is just a chain.
type LedgerMonitor struct {
	db     *sqlx.DB
	ledger *Ledger
	alert  AlertFunc
}

func NewLedgerMonitor(db *sqlx.DB, ledger *Ledger, alert AlertFunc) *LedgerMonitor {
	return &LedgerMonitor{db: db, ledger: ledger, alert: alert}
}

// RunOnce checks every tenant that has at least one ai_bridge_targets row
// (active or not — a tampered log is worth flagging even for a
// since-deactivated target) and returns any breaches found. Safe to call on
// its own for a manual/CLI check, independent of the ticking loop.
func (m *LedgerMonitor) RunOnce(ctx context.Context) ([]LedgerBreach, error) {
	var tenantIDs []uuid.UUID
	err := m.db.SelectContext(ctx, &tenantIDs, `
		SELECT DISTINCT tenant_id FROM catalog_ai.ai_bridge_targets`)
	if err != nil {
		return nil, fmt.Errorf("listing tenants for ledger verification: %w", err)
	}

	var breaches []LedgerBreach
	for _, tenantID := range tenantIDs {
		brokenAt, err := m.ledger.Verify(ctx, tenantID)
		if err != nil {
			log.Printf("[LedgerMonitor] verify failed for tenant %s: %v", tenantID, err)
			continue
		}
		if brokenAt != uuid.Nil {
			breach := LedgerBreach{TenantID: tenantID, FirstBadLog: brokenAt, DetectedAt: time.Now()}
			breaches = append(breaches, breach)
			log.Printf("[LedgerMonitor] TAMPER DETECTED: tenant=%s first_broken_log=%s", tenantID, brokenAt)
			if m.alert != nil {
				m.alert(ctx, breach)
			}
		}
	}
	return breaches, nil
}

// Start runs RunOnce on a ticker until ctx is cancelled. Intended to be
// launched with `go monitor.Start(ctx, interval)` from the server's main.
func (m *LedgerMonitor) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Check once immediately on startup rather than waiting a full interval.
	if _, err := m.RunOnce(ctx); err != nil {
		log.Printf("[LedgerMonitor] initial check failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if _, err := m.RunOnce(ctx); err != nil {
				log.Printf("[LedgerMonitor] periodic check failed: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
