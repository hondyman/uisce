package datapipeline

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOutboxSchema = "test_outbox_schema"

func testDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://postgres@localhost:5432/alpha?sslmode=disable&options=-c%20search_path%3D" + testOutboxSchema
}

func testDB(t *testing.T) *sqlx.DB {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}
	dsn := testDSN()
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return nil
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("skipping: %v", err)
		return nil
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func ensureOutboxSchema(t *testing.T, db *sqlx.DB, ctx context.Context) {
	_, _ = db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+testOutboxSchema)
	_, _ = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS `+testOutboxSchema+`.outbox (
			id uuid DEFAULT gen_random_uuid() NOT NULL,
			event_type text NOT NULL,
			payload jsonb NOT NULL,
			published boolean DEFAULT false,
			created_at timestamp with time zone DEFAULT now(),
			published_at timestamp with time zone
		)
	`)
}

func outboxCount(db *sqlx.DB, ctx context.Context, pipelineID uuid.UUID) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM `+testOutboxSchema+`.outbox `+
			`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String()).Scan(&n)
	return n, err
}

func TestOutboxPublisher_TxAtomicity_Rollback(t *testing.T) {
	db := testDB(t)
	if db == nil {
		return
	}
	ctx := context.Background()
	ensureOutboxSchema(t, db, ctx)

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String(), "name": "test-record"}

	_, _ = db.ExecContext(ctx, `DELETE FROM `+testOutboxSchema+`.outbox `+
		`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String())

	pub := NewOutboxPublisher(db)

	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	err = pub.PublishPipelineTriggerTx(ctx, tx, tenantID, pipelineID, uuid.Nil, record)
	require.NoError(t, err)

	tx.Rollback()

	n, err := outboxCount(db, ctx, pipelineID)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "outbox row must not survive a rolled-back transaction")
}

func TestOutboxPublisher_TxAtomicity_Commit(t *testing.T) {
	db := testDB(t)
	if db == nil {
		return
	}
	ctx := context.Background()
	ensureOutboxSchema(t, db, ctx)

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String(), "name": "test-record"}

	_, _ = db.ExecContext(ctx, `DELETE FROM `+testOutboxSchema+`.outbox `+
		`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String())

	pub := NewOutboxPublisher(db)

	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)

	err = pub.PublishPipelineTriggerTx(ctx, tx, tenantID, pipelineID, uuid.Nil, record)
	require.NoError(t, err)

	require.NoError(t, tx.Commit())

	n, err := outboxCount(db, ctx, pipelineID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "outbox row must exist after commit")

	_, _ = db.ExecContext(ctx, `DELETE FROM `+testOutboxSchema+`.outbox `+
		`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String())
}

func TestOutboxPublisher_Legacy_PublishTrigger(t *testing.T) {
	db := testDB(t)
	if db == nil {
		return
	}
	ctx := context.Background()
	ensureOutboxSchema(t, db, ctx)

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String(), "name": "test-record"}

	_, _ = db.ExecContext(ctx, `DELETE FROM `+testOutboxSchema+`.outbox `+
		`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String())

	pub := NewOutboxPublisher(db)

	err := pub.PublishPipelineTrigger(ctx, tenantID, pipelineID, uuid.Nil, record)
	require.NoError(t, err)

	n, err := outboxCount(db, ctx, pipelineID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "outbox row must exist after legacy publish")

	_, _ = db.ExecContext(ctx, `DELETE FROM `+testOutboxSchema+`.outbox `+
		`WHERE event_type = $1 AND payload->>'pipeline_id' = $2`,
		PipelineTriggerEventType, pipelineID.String())
}
