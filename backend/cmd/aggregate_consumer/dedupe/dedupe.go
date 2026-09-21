package dedupe

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the consumer's at-least-once guard. Each handler marks a
// (source_lsn, table, op) tuple as processed after a successful handle; on
// replay (consumer restart, rebalance, slot reset) the same tuple is rejected
// here so the handler is invoked at most once per event.
//
// Keyed on (lsn, table, op) — not just lsn — because:
//   - One transaction can touch multiple rows of one table, all sharing the
//     same lsn; the op distinguishes them.
//   - REPLICA IDENTITY FULL emits before rows; one lsn can carry multiple
//     before/after pairs across statements. Composite key avoids the rare
//     collision where two different rows share (lsn, table).
type Store struct {
	pool *pgxpool.Pool
}

// ErrAlreadyProcessed is returned by MarkIfNew when the event has been seen
// before — handlers should treat this as "skip silently", not an error.
var ErrAlreadyProcessed = errors.New("event already processed")

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// EnsureSchema creates the dedupe table if it doesn't exist.
// Idempotent — safe to call on every startup.
func (s *Store) EnsureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS consumer_dedupe (
    handler      TEXT        NOT NULL,
    source_lsn   TEXT        NOT NULL,
    table_name   TEXT        NOT NULL,
    op           CHAR(1)     NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (handler, source_lsn, table_name, op)
);

CREATE INDEX IF NOT EXISTS idx_consumer_dedupe_processed_at
    ON consumer_dedupe (processed_at);
`
	_, err := s.pool.Exec(ctx, ddl)
	return err
}

// MarkIfNew atomically inserts the dedupe row. Returns ErrAlreadyProcessed
// if the row already exists. Returns nil + a fresh row if inserted.
func (s *Store) MarkIfNew(ctx context.Context, handler, lsn, table, op string) error {
	if lsn == "" {
		return fmt.Errorf("dedupe: empty source_lsn")
	}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO consumer_dedupe (handler, source_lsn, table_name, op)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
`, handler, lsn, table, op)
	if err != nil {
		return fmt.Errorf("dedupe insert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyProcessed
	}
	return nil
}

// Prune deletes dedupe rows older than the retention window. Run on a schedule.
// 7 days is conservative; matches Kafka offset retention for stream-loader.
func (s *Store) Prune(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM consumer_dedupe
WHERE processed_at < NOW() - ($1 || ' days')::INTERVAL
`, fmt.Sprintf("%d", retentionDays))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
