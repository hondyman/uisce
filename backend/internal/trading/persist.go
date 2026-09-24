package trading

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// PersistFIXRouteActivity writes the outbound placement to crims.orm
// (the UI spine). Never writes alpha.orm.
func PersistFIXRouteActivity(ctx context.Context, input FIXOrderInput) error {
	db, err := OpenCRIMS(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	placementID := input.PlacementID
	if placementID == "" {
		placementID = uuid.NewString()
	}
	broker := input.BrokerCode
	if broker == "" {
		broker = "GSCO"
	}
	qty := input.Quantity
	_, err = db.ExecContext(ctx, `
		INSERT INTO orm.placement (
			id, order_id, broker_id, venue_id, routed_qty, executed_qty, leaves_qty,
			status, fix_clordid, tenant_id
		) VALUES ($1, $2, $3, 'XNAS', $4, 0, $4, 'ROUTED', $5, $6)
	`, placementID, input.OrderID, broker, qty, input.ClOrdID, input.TenantID)
	if err != nil {
		return fmt.Errorf("insert crims.orm.placement: %w", err)
	}
	_, err = db.ExecContext(ctx, `
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

// PersistFIXFillActivity applies an ExecutionReport to placement,
// execution, order, and remaining allocations on crims.orm. Pages never
// write fills.
func PersistFIXFillActivity(ctx context.Context, input FIXFillPersist) error {
	db, err := OpenCRIMS(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

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
	err = db.QueryRowContext(ctx, `
		SELECT id::text FROM orm.placement
		WHERE tenant_id = $1::uuid AND fix_clordid = $2
		ORDER BY id LIMIT 1
	`, input.TenantID, input.ClOrdID).Scan(&placementID)
	if err != nil {
		return fmt.Errorf("lookup placement for clordid %s: %w", input.ClOrdID, err)
	}

	if qty > 0 {
		_, err = db.ExecContext(ctx, `
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
	_, err = db.ExecContext(ctx, `
		UPDATE orm.placement
		SET executed_qty = executed_qty + $1,
		    leaves_qty = GREATEST(leaves_qty - $1, 0),
		    status = CASE WHEN GREATEST(leaves_qty - $1, 0) = 0 THEN $2 ELSE 'PARTIAL' END
		WHERE id = $3::uuid AND tenant_id = $4::uuid
	`, qty, placeStatus, placementID, input.TenantID)
	if err != nil {
		return fmt.Errorf("update crims.orm.placement fill: %w", err)
	}

	_, err = db.ExecContext(ctx, `
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
		_, _ = db.ExecContext(ctx, `
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

func OpenCRIMS(ctx context.Context) (*sql.DB, error) {
	dsn := os.Getenv("CRIMS_ORM_DSN")
	if dsn == "" {
		base := os.Getenv("DATABASE_URL")
		if base == "" {
			base = os.Getenv("POSTGRES_DSN")
		}
		if base == "" {
			return nil, fmt.Errorf("CRIMS_ORM_DSN and DATABASE_URL are unset")
		}
		dsn = swapDBName(base, "crims")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open crims: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping crims: %w", err)
	}
	return db, nil
}

func swapDBName(dsn, dbname string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		// libpq key=value form
		if strings.Contains(dsn, "dbname=") {
			return replaceKV(dsn, "dbname", dbname)
		}
		return dsn
	}
	u.Path = "/" + dbname
	return u.String()
}

func replaceKV(dsn, key, val string) string {
	parts := strings.Fields(dsn)
	out := make([]string, 0, len(parts))
	found := false
	for _, p := range parts {
		if strings.HasPrefix(p, key+"=") {
			out = append(out, key+"="+val)
			found = true
			continue
		}
		out = append(out, p)
	}
	if !found {
		out = append(out, key+"="+val)
	}
	return strings.Join(out, " ")
}
