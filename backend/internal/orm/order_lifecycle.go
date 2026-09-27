package orm

import (
	"context"
	"database/sql"
	"time"
)

// OrderEvent is one row of orm.order_event.
type OrderEvent struct {
	ID          string
	OrderID     string
	EventType   string
	FromStatus  sql.NullString
	ToStatus    string
	EventTime   time.Time
	EventSource sql.NullString
	MessageID   sql.NullString
	ReasonCd    sql.NullString
	ReasonText  sql.NullString
}

// InsertOrderEvent appends an immutable order event.
func InsertOrderEvent(ctx context.Context, tx *sql.Tx, tenantID string, e *OrderEvent) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.order_event (
			order_id, event_type, from_status, to_status,
			event_time, event_source, message_id,
			reason_cd, reason_text, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, e.OrderID, e.EventType, e.FromStatus, e.ToStatus,
		e.EventTime, e.EventSource, e.MessageID,
		e.ReasonCd, e.ReasonText, tenantID)
	return err
}

// InsertOrderAmendment records a field-level change on an order.
func InsertOrderAmendment(ctx context.Context, tx *sql.Tx, tenantID string,
	orderID string, num int, field, prior, next, reason string,
	amendedBy string, clientDirected bool) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.order_amendment (
			order_id, amendment_num, amendment_time,
			field_changed, prior_value, new_value,
			reason, amended_by, is_client_directed, tenant_id
		) VALUES ($1,$2,now(),$3,$4,$5,$6,$7,$8,$9)
	`, orderID, num, field, prior, next, reason, amendedBy, clientDirected, tenantID)
	return err
}

// InsertOrderReject records a rejection with structured reason code.
func InsertOrderReject(ctx context.Context, tx *sql.Tx, tenantID,
	orderID, placementID, source, code, reason, text string,
	retriable bool, retriedAsOrderID sql.NullString) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.order_reject (
			order_id, placement_id, reject_time,
			reject_source, reject_code, reject_reason, reject_text,
			is_retriable, retried_as_order_id, tenant_id
		) VALUES ($1,$2,now(),$3,$4,$5,$6,$7,$8,$9)
	`, orderID, nullIfEmpty(placementID), source, code, reason, text,
		retriable, retriedAsOrderID, tenantID)
	return err
}
