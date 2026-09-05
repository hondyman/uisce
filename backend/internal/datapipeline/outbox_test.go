package datapipeline

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestOutboxPublishTx_Rollback_RowGone(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration gate")
	}

	ctx := context.Background()
	dbx, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer dbx.Close()

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String(), "total": 100.0}

	pub := NewOutboxPublisher(dbx)

	tx, err := dbx.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	err = pub.PublishPipelineTriggerTx(ctx, tx, tenantID, pipelineID, record)
	if err != nil {
		t.Fatalf("PublishPipelineTriggerTx: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var n int
	err = dbx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM outbox WHERE payload->>'pipeline_id' = $1`,
		pipelineID.String(),
	).Scan(&n)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 outbox rows after rollback, got %d — atomicity guarantee violated", n)
	}
}

func TestOutboxPublishTx_Commit_RowVisibleToAnotherConnection(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration gate")
	}

	ctx := context.Background()
	dbx, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer dbx.Close()

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String(), "total": 100.0}

	pub := NewOutboxPublisher(dbx)

	tx, err := dbx.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	err = pub.PublishPipelineTriggerTx(ctx, tx, tenantID, pipelineID, record)
	if err != nil {
		t.Fatalf("PublishPipelineTriggerTx: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	dbx2, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("connect second connection: %v", err)
	}
	defer dbx2.Close()

	var n int
	err = dbx2.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM outbox WHERE payload->>'pipeline_id' = $1 AND payload->>'tenant_id' = $2`,
		pipelineID.String(), tenantID.String(),
	).Scan(&n)
	if err != nil {
		t.Fatalf("query outbox from second connection: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 outbox row visible from another connection after commit, got %d", n)
	}

	_, err = dbx.ExecContext(ctx, `DELETE FROM outbox WHERE payload->>'pipeline_id' = $1`, pipelineID.String())
	if err != nil {
		t.Logf("cleanup DELETE failed (non-fatal): %v", err)
	}
}

func TestOutboxPublish_LegacyAdapter_WarnsAndCompletes(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration gate")
	}

	ctx := context.Background()
	dbx, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer dbx.Close()

	pipelineID := uuid.New()
	tenantID := uuid.New()
	record := map[string]interface{}{"id": uuid.New().String()}

	var warnBuf bytes.Buffer
	log.SetOutput(&warnBuf)
	defer log.SetOutput(os.Stderr)

	pub := NewOutboxPublisher(dbx)

	err = pub.PublishPipelineTrigger(ctx, tenantID, pipelineID, record)
	if err != nil {
		t.Fatalf("legacy PublishPipelineTrigger failed: %v", err)
	}

	logOutput := warnBuf.String()
	if !strings.Contains(logOutput, "legacy") {
		t.Errorf("expected deprecation warning in log output, got: %s", logOutput)
	}

	_, err = dbx.ExecContext(ctx, `DELETE FROM outbox WHERE payload->>'pipeline_id' = $1`, pipelineID.String())
	if err != nil {
		t.Logf("cleanup DELETE failed (non-fatal): %v", err)
	}
}
