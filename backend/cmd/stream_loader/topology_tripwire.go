package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// The topology tripwire.
//
// The loader routes a change event to a tenant's StarRocks database using
// after.tenant_id, because the connector captures `alpha` -- the control plane -- and
// alpha's tables carry a tenant_id column. That is the correct implementation of the
// topology as it is configured today.
//
// The target topology is different: `alpha` stays the metadata control plane, and each
// tenant's business data lives in its own Postgres database named by
// tenant_product_datasource.config. Under that shape the discriminator has to become
// source.db, and a tenant's own database need not carry a tenant_id column at all --
// which means this loader would dead-letter every row of it as unattributed.
//
// That failure is loud, not silent: rows land in the DLQ with "no tenant_id" and the
// tenant_unattributed counter climbs. Loud is the property that makes deferring the
// connector rebuild safe.
//
// What the tripwire adds is the *earlier* signal. tenant_datasource_binding is the
// control plane's own registry of per-tenant starrocks_database / starrocks_role
// (UNIQUE, so one physical database cannot bind to two tenants). It holds 0 rows today.
// The first row in it means a per-tenant analytics destination has been declared, and
// at that moment the connector is no longer the whole story -- someone has started
// building the data plane.
//
// Documenting that in a comment only helps whoever opens the file. This turns it into
// an alarm the moment the condition becomes true, which is the same pattern as the
// connector-config drift check: the condition that invalidates a design should be
// detectable when it occurs, not the next time somebody reads routing.go.

// topologyTripwireTable is the registry that flips the discriminator when it stops
// being empty.
const topologyTripwireTable = "tenant_datasource_binding"

// rowCounter is the slice of *sql.DB / *sqlx.DB the tripwire needs. Keeping it an
// interface is what lets the check be tested without a live control plane.
type rowCounter interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// bindingCounter is what checkTopologyTripwire actually depends on: one number, the
// row count of the registry. Narrower than rowCounter on purpose -- a fake that returns
// a count and an error tests the decision, while a fake that has to fabricate a
// *sql.Row only tests database/sql.
type bindingCounter interface {
	CountBindings(ctx context.Context) (int64, error)
}

// sqlBindingCounter reads the registry from the control-plane connection.
type sqlBindingCounter struct{ db rowCounter }

func (s sqlBindingCounter) CountBindings(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM "+topologyTripwireTable).Scan(&n)
	return n, err
}

// tripwireVerdict is the outcome of one check, kept separate from the logging so the
// decision is testable and so a caller can act on it.
type tripwireVerdict struct {
	bindings int64
	fired    bool
	// reason explains a firing, or says why the check could not conclude. Empty when
	// there is simply nothing to report.
	reason string
}

// checkTopologyTripwire asks whether the control plane has begun declaring per-tenant
// analytics destinations while this loader still assumes the old discriminator.
//
// It deliberately does NOT fire on a query error. A control plane that is briefly
// unreachable is not evidence of a topology change, and an alarm that cries wolf on a
// transient is one people learn to ignore -- which is the same defect class as the
// silent failures this work set out to remove.
func checkTopologyTripwire(ctx context.Context, counter bindingCounter, cfg Config) tripwireVerdict {
	if counter == nil {
		return tripwireVerdict{reason: "no control-plane connection; tripwire cannot run"}
	}
	// Without per-tenant routing enabled there is no discriminator in play at all, so
	// the registry's contents say nothing about this process.
	if cfg.TenantRoutesDir == "" {
		return tripwireVerdict{reason: "per-tenant routing not enabled; discriminator is the shared database"}
	}

	n, err := counter.CountBindings(ctx)
	if err != nil {
		return tripwireVerdict{reason: fmt.Sprintf("cannot read %s: %v", topologyTripwireTable, err)}
	}
	if n == 0 {
		return tripwireVerdict{bindings: 0}
	}

	// The discriminator is still the tenant_id column. That is only correct while the
	// connector captures the control plane, which it does when CDC_SOURCE_DB names it.
	if cfg.SourceDatabase != controlPlaneDatabase {
		return tripwireVerdict{bindings: n,
			reason: fmt.Sprintf("%s has %d row(s) and CDC_SOURCE_DB=%s, so routing is already keyed on source.db",
				topologyTripwireTable, n, cfg.SourceDatabase)}
	}

	return tripwireVerdict{bindings: n, fired: true, reason: fmt.Sprintf(
		"%s has %d row(s): per-tenant analytics destinations are being declared while the connector still "+
			"captures only %q and this loader still discriminates on after.tenant_id. A tenant whose own "+
			"Postgres database carries no tenant_id column would dead-letter every row. This is the trigger "+
			"to move the discriminator to source.db, rebuild the connector per tenant database (one logical "+
			"replication slot each), and make tenant_datasource_binding the source of truth for database and "+
			"role names. See the sourceDB seam in routing.go.",
		topologyTripwireTable, n, controlPlaneDatabase)}
}

// controlPlaneDatabase is the database that holds metadata for every tenant.
const controlPlaneDatabase = "alpha"

// reportTopologyTripwire runs one check and turns a firing into a logged alert and a
// counter, so the condition is visible on /metrics and not only in the log.
func reportTopologyTripwire(ctx context.Context, counter bindingCounter, cfg Config, metrics *GatekeeperMetrics) tripwireVerdict {
	v := checkTopologyTripwire(ctx, counter, cfg)
	if v.fired {
		metrics.TopologyTripwireFired.Add(1)
		log.Printf("[ALERT][TOPOLOGY] %s", v.reason)
	}
	return v
}

// topologyTripwireInterval is deliberately slow. This detects a topology decision, not
// an outage; a loud check that re-runs every minute is how a real outage gets missed.
const topologyTripwireInterval = 5 * time.Minute

// runTopologyTripwire checks at startup and then on an interval until ctx ends.
//
// The startup check matters most: the operator deploying a tenant data plane should
// learn that the loader cannot serve it while they are still watching, not three weeks
// later when someone asks why the numbers are wrong.
func runTopologyTripwire(ctx context.Context, counter bindingCounter, cfg Config, metrics *GatekeeperMetrics) {
	reportTopologyTripwire(ctx, counter, cfg, metrics)

	ticker := time.NewTicker(topologyTripwireInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reportTopologyTripwire(ctx, counter, cfg, metrics)
		}
	}
}
