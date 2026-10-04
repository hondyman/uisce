package trading

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
)

// ORMApp is the app code of a tenant's ORM database. It is NOT CoreApp ("core",
// the wealth/business-object database): provisioning records App = "orm" for this
// one, and the router looks the tenant's single datasource up by that code. A
// wrong code is a runtime refusal (ErrBadApp / ErrUnbound), not a compile error.
const ORMApp = "orm"

// Resolver is the slice of tenantdb.Router these activities need. It is an
// interface so the tests can drive the refusal paths without a registry.
type Resolver interface {
	ResolveApp(ctx context.Context, app string) (*tenantdb.Pool, error)
}

// ErrNoResolver is returned when the process never installed one. It is an
// error, never a fallback: this package used to read CRIMS_ORM_DSN and open one
// shared database for every tenant, and the whole point of the move is that no
// code path may reach a tenant's ORM data any other way (ADR-030).
var ErrNoResolver = errors.New("trading: no tenantdb resolver installed; the ORM is only reachable through tenantdb (ADR-030)")

var (
	resolverMu sync.RWMutex
	resolver   Resolver
)

// SetResolver installs the process's router. Call it once at start-up, before
// the worker registers activities. There is deliberately no default: an
// uninstalled resolver fails closed.
func SetResolver(r Resolver) {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	resolver = r
}

func currentResolver() (Resolver, error) {
	resolverMu.RLock()
	defer resolverMu.RUnlock()
	if resolver == nil {
		return nil, ErrNoResolver
	}
	return resolver, nil
}

// ormDB resolves the calling tenant's own ORM database.
//
// These are Temporal activities: they have no request context, so the tenant
// comes from the activity INPUT and is placed in the context as the caller
// tenant (db.WithTenantContextToCtx). That makes the datasource have to belong
// to THAT tenant — it is what stops a Fill for tenant A being written into
// tenant B's database — but the authenticity of input.TenantID remains the
// caller's responsibility, exactly as TenantDBManager documents.
func ormDB(ctx context.Context, tenantID uuid.UUID) (*sql.DB, error) {
	r, err := currentResolver()
	if err != nil {
		return nil, err
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("trading: activity input carries no tenant id")
	}
	pool, err := r.ResolveApp(db.WithTenantContextToCtx(ctx, tenantID.String()), ORMApp)
	if err != nil {
		return nil, fmt.Errorf("resolve the orm database for tenant %s: %w", tenantID, err)
	}
	return pool.SQLDB(), nil
}

// PersistFIXRouteActivity writes the outbound placement to the tenant's own
// orm.placement (the UI spine). Every write goes to the database the registry
// binds to the caller's tenant, so a route can no longer land in a shared
// database next to another tenant's orders.
func PersistFIXRouteActivity(ctx context.Context, input FIXOrderInput) error {
	conn, err := ormDB(ctx, input.TenantID)
	if err != nil {
		return err
	}
	defer conn.Close()

	placementID := input.PlacementID
	if placementID == "" {
		placementID = uuid.NewString()
	}
	broker := input.BrokerCode
	if broker == "" {
		broker = "GSCO"
	}
	qty := input.Quantity
	_, err = conn.ExecContext(ctx, `
		INSERT INTO orm.placement (
			id, order_id, broker_id, venue_id, routed_qty, executed_qty, leaves_qty,
			status, fix_clordid, tenant_id
		) VALUES ($1, $2, $3, 'XNAS', $4, 0, $4, 'ROUTED', $5, $6)
	`, placementID, input.OrderID, broker, qty, input.ClOrdID, input.TenantID)
	if err != nil {
		return fmt.Errorf("insert crims.orm.placement: %w", err)
	}
	_, err = conn.ExecContext(ctx, `
		UPDATE orm."order"
		SET status = CASE WHEN status IN ('NEW','DRAFT') THEN 'ROUTED' ELSE status END,
		    updated_at = NOW()
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, input.OrderID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update crims.orm.order route: %w", err)
	}
	return nil
}

// PersistFIXFillActivity applies an ExecutionReport to placement, execution,
// order, and remaining allocations in the tenant's own orm database. Pages never
// write fills.
func PersistFIXFillActivity(ctx context.Context, input FIXFillPersist) error {
	conn, err := ormDB(ctx, input.TenantID)
	if err != nil {
		return err
	}
	defer conn.Close()

	qty := input.LastQty
	if qty <= 0 {
		qty = input.Quantity
	}
	px := input.LastPx
	execID := input.ExecID
	if execID == "" {
		execID = uuid.NewString()
	}
	status := "FILLED"
	if strings.EqualFold(input.OrdStatus, "4") || strings.EqualFold(input.Status, "Canceled") {
		status = "CANCELED"
		qty = 0
	}

	var placementID string
	err = conn.QueryRowContext(ctx, `
		SELECT id::text FROM orm.placement
		WHERE tenant_id = $1::uuid AND fix_clordid = $2
		ORDER BY id LIMIT 1
	`, input.TenantID, input.ClOrdID).Scan(&placementID)
	if err != nil {
		return fmt.Errorf("lookup placement for clordid %s: %w", input.ClOrdID, err)
	}

	if qty > 0 {
		_, err = conn.ExecContext(ctx, `
			INSERT INTO orm.execution (
				id, placement_id, order_id, exec_qty, exec_price, exec_time,
				broker_exec_id, last_capacity, tenant_id
			) VALUES ($1, $2, $3, $4, $5, NOW(), $6, 'A', $7)
		`, uuid.NewString(), placementID, input.OrderID, qty, px, execID, input.TenantID)
		if err != nil {
			return fmt.Errorf("insert crims.orm.execution: %w", err)
		}
	}

	placeStatus := status
	if status == "CANCELED" {
		placeStatus = "CANCELED"
	}
	_, err = conn.ExecContext(ctx, `
		UPDATE orm.placement
		SET executed_qty = executed_qty + $1,
		    leaves_qty = GREATEST(leaves_qty - $1, 0),
		    status = CASE WHEN GREATEST(leaves_qty - $1, 0) = 0 THEN $2 ELSE 'PARTIAL' END
		WHERE id = $3::uuid AND tenant_id = $4::uuid
	`, qty, placeStatus, placementID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update crims.orm.placement fill: %w", err)
	}

	_, err = conn.ExecContext(ctx, `
		UPDATE orm."order"
		SET executed_qty = executed_qty + $1,
		    leaves_qty = GREATEST(leaves_qty - $1, 0),
		    avg_price = CASE WHEN executed_qty + $1 > 0
		      THEN ((COALESCE(avg_price,0) * executed_qty) + ($2 * $1)) / (executed_qty + $1)
		      ELSE avg_price END,
		    status = CASE
		      WHEN $3 = 'CANCELED' THEN 'CANCELED'
		      WHEN GREATEST(leaves_qty - $1, 0) = 0 THEN 'FILLED'
		      ELSE 'PARTIAL'
		    END,
		    updated_at = NOW()
		WHERE id = $4::uuid AND tenant_id = $5::uuid
	`, qty, px, status, input.OrderID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update crims.orm.order fill: %w", err)
	}

	if qty > 0 {
		_, _ = conn.ExecContext(ctx, `
			UPDATE orm.order_allocation
			SET allocated_qty = LEAST(target_qty, allocated_qty + $1),
			    status = CASE WHEN allocated_qty + $1 >= target_qty THEN 'ALLOCATED' ELSE 'PARTIAL' END
			WHERE id = (
			  SELECT id FROM orm.order_allocation
			  WHERE tenant_id = $2::uuid AND order_id = $3::uuid AND allocated_qty < target_qty
			  ORDER BY id LIMIT 1
			)
		`, qty, input.TenantID, input.OrderID)
	}
	return nil
}

type FIXFillPersist struct {
	FIXOrderInput
	LastQty   float64 `json:"last_qty"`
	LastPx    float64 `json:"last_px"`
	ExecID    string  `json:"exec_id"`
	OrdStatus string  `json:"ord_status"`
}
