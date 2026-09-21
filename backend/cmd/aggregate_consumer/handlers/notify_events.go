package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	kafka "github.com/segmentio/kafka-go"
)

// NotifyEvents is a stateless pass-through. The original triggers
// notify_security_change and notify_metrics_registry_changed emitted
// pg_notify('security_fund_access_change', ...) and pg_notify('metrics_registry_changed', ...).
// This consumer republishes those events to Kafka topics so downstream
// consumers (audit/observability tooling) can subscribe via the CDC pipeline.
//
// Topics consumed:
//   alpha_trg.public.security_user_fund_access
//   alpha_trg.public.metrics_registry
//
// Topics produced:
//   agg.security_fund_access_change
//   agg.metrics_registry_changed
type NotifyEvents struct {
	Pool     *pgxpool.Pool
	Writer   *kafka.Writer
	tableMap map[string]string // source table name -> output agg topic
}

func NewNotifyEvents(pool *pgxpool.Pool, writer *kafka.Writer) *NotifyEvents {
	return &NotifyEvents{
		Pool:   pool,
		Writer: writer,
		tableMap: map[string]string{
			"security_user_fund_access": "agg.security_fund_access_change",
			"metrics_registry":          "agg.metrics_registry_changed",
		},
	}
}

// Topics returns the source CDC topics this handler subscribes to. Used by
// the multi-topic registration path in main.go.
func (h *NotifyEvents) Topics() []string {
	return []string{
		"alpha_trg.public.security_user_fund_access",
		"alpha_trg.public.metrics_registry",
	}
}

func (h *NotifyEvents) Handle(ctx context.Context, evt Event) error {
	out, ok := h.tableMap[evt.Table]
	if !ok {
		// Unknown source table; not our problem, just skip.
		return nil
	}

	payload := map[string]any{
		"op":         evt.Op,
		"table":      evt.Table,
		"ts_ms":      evt.TS,
		"source_lsn": evt.LSN,
		"after":      evt.After,
	}
	if evt.Before != nil {
		payload["before"] = evt.Before
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Topic: out,
		Key:   lsnKey(evt.LSN),
		Value: body,
		Headers: []kafka.Header{
			{Key: "x-source-table", Value: []byte(evt.Table)},
			{Key: "x-source-lsn", Value: []byte(evt.LSN)},
		},
	}
	if err := h.Writer.WriteMessages(ctx, msg); err != nil {
		return err
	}
	log.Printf("[notify_events] %s op=%s lsn=%s -> %s", evt.Table, evt.Op, evt.LSN, out)
	return nil
}

func lsnKey(lsn string) []byte {
	if lsn == "" {
		return []byte("unknown")
	}
	return []byte(lsn)
}
