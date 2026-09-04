package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const pipelineRunChannel = "pipeline_runs"

type runNotification struct {
	RunID string                 `json:"run_id"`
	Step  map[string]interface{} `json:"step,omitempty"`
	Run   *PipelineExecutionRun  `json:"run,omitempty"`
}

type TelemetryBus struct {
	db *sqlx.DB

	mut       sync.RWMutex
	listeners map[string]chan *runNotification
	done      chan struct{}

	lsn *pq.Listener
}

func NewTelemetryBus(db *sqlx.DB) *TelemetryBus {
	return &TelemetryBus{
		db:        db,
		listeners: make(map[string]chan *runNotification),
		done:      make(chan struct{}),
	}
}

func (b *TelemetryBus) Start(ctx context.Context, connStr string) error {
	if b.db == nil {
		return fmt.Errorf("telemetry bus has no database")
	}

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Printf("telemetry bus listener error: %v\n", err)
		}
	}

	b.lsn = pq.NewListener(connStr, 10*time.Second, time.Minute, reportProblem)
	if b.lsn == nil {
		return fmt.Errorf("failed to create pq listener")
	}

	if err := b.lsn.Listen(pipelineRunChannel); err != nil {
		return fmt.Errorf("listen %s: %w", pipelineRunChannel, err)
	}

	go b.dispatchLoop(ctx)
	return nil
}

func (b *TelemetryBus) dispatchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notification, ok := <-b.lsn.Notify:
			if !ok {
				return
			}
			b.dispatch(notification)
		case <-time.After(90 * time.Second):
			if b.lsn != nil {
				b.lsn.Ping()
			}
		}
	}
}

func (b *TelemetryBus) dispatch(notification *pq.Notification) {
	if notification == nil || len(notification.Extra) == 0 {
		return
	}

	var n runNotification
	if err := json.Unmarshal([]byte(notification.Extra), &n); err != nil {
		fmt.Printf("telemetry bus: failed to unmarshal notification: %v\n", err)
		return
	}

	b.mut.RLock()
	ch, ok := b.listeners[n.RunID]
	b.mut.RUnlock()

	if ok {
		select {
		case ch <- &n:
		default:
		}
	}
}

func (b *TelemetryBus) Listen(runID string) (chan *runNotification, func()) {
	b.mut.Lock()
	defer b.mut.Unlock()

	ch := make(chan *runNotification, 10)
	b.listeners[runID] = ch

	cleanup := func() {
		b.mut.Lock()
		delete(b.listeners, runID)
		close(ch)
		b.mut.Unlock()
	}
	return ch, cleanup
}

func (b *TelemetryBus) NotifyStep(ctx context.Context, run PipelineExecutionRun, stepKey string) error {
	if b.db == nil {
		return nil
	}

	step, ok := run.StepTelemetry[stepKey]
	stepMap := map[string]interface{}{
		"node_id":       step.NodeID,
		"node_label":    step.NodeLabel,
		"node_type":     step.NodeType,
		"status":        step.Status,
		"records_in":    step.RecordsIn,
		"records_out":   step.RecordsOut,
		"records_error": step.RecordsError,
		"duration_ms":   step.Duration.Milliseconds(),
		"rows_per_sec": step.RowsPerSec,
	}
	if !ok {
		stepMap = nil
	}

	n := runNotification{
		RunID: run.RunID.String(),
		Step:  stepMap,
		Run:   &run,
	}

	payload, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	if _, err := b.db.ExecContext(ctx, "SELECT pg_notify($1, $2)", pipelineRunChannel, string(payload)); err != nil {
		return fmt.Errorf("pg_notify: %w", err)
	}

	return nil
}

func (b *TelemetryBus) Stop() error {
	if b.lsn != nil {
		return b.lsn.Close()
	}
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	return nil
}
