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

const pipelineRunChannelPrefix = "pipeline_run_"

type TelemetryBus struct {
	db *sqlx.DB

	mut       sync.RWMutex
	listeners map[string]chan PipelineExecutionRun
	done      chan struct{}

	lsn     *pq.Listener
	connStr string
}

func NewTelemetryBus(db *sqlx.DB) *TelemetryBus {
	return &TelemetryBus{
		db:        db,
		listeners: make(map[string]chan PipelineExecutionRun),
		done:      make(chan struct{}),
	}
}

func (b *TelemetryBus) Start(ctx context.Context, connStr string) error {
	if b.db == nil {
		return fmt.Errorf("telemetry bus has no database")
	}

	b.connStr = connStr

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Printf("telemetry bus listener error: %v\n", err)
		}
	}

	minReconnect := 10 * time.Second
	maxReconnect := time.Minute

	b.lsn = pq.NewListener(connStr, minReconnect, maxReconnect, reportProblem)
	if b.lsn == nil {
		return fmt.Errorf("failed to create pq listener")
	}

	if err := b.lsn.Listen(pipelineRunChannelPrefix + "all"); err != nil {
		return fmt.Errorf("listen on wildcard channel: %w", err)
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
	if notification == nil {
		return
	}

	if len(notification.Extra) == 0 {
		return
	}

	var run PipelineExecutionRun
	if err := json.Unmarshal([]byte(notification.Extra), &run); err != nil {
		fmt.Printf("telemetry bus: failed to unmarshal run: %v\n", err)
		return
	}

	runID := run.RunID.String()

	b.mut.RLock()
	ch, ok := b.listeners[runID]
	b.mut.RUnlock()

	if ok {
		select {
		case ch <- run:
		default:
		}
	}
}

func (b *TelemetryBus) Listen(runID string) (chan PipelineExecutionRun, func()) {
	b.mut.Lock()
	defer b.mut.Unlock()

	ch := make(chan PipelineExecutionRun, 10)
	b.listeners[runID] = ch

	cleanup := func() {
		b.mut.Lock()
		delete(b.listeners, runID)
		close(ch)
		b.mut.Unlock()
	}
	return ch, cleanup
}

func (b *TelemetryBus) NotifyRun(run PipelineExecutionRun) error {
	if b.db == nil {
		return nil
	}

	payload, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal run for notify: %w", err)
	}

	channel := pipelineRunChannelPrefix + run.RunID.String()
	if _, err := b.db.ExecContext(context.Background(), "SELECT pg_notify($1, $2)", channel, string(payload)); err != nil {
		return fmt.Errorf("pg_notify: %w", err)
	}

	return nil
}

func (b *TelemetryBus) NotifyRunCtx(ctx context.Context, run PipelineExecutionRun) error {
	if b.db == nil {
		return nil
	}

	payload, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal run for notify: %w", err)
	}

	channel := pipelineRunChannelPrefix + run.RunID.String()
	if _, err := b.db.ExecContext(ctx, "SELECT pg_notify($1, $2)", channel, string(payload)); err != nil {
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
