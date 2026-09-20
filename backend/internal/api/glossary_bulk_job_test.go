package api

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// TestGenerateTermsAsyncDispatch verifies that items > syncThreshold dispatch
// a job and the job reaches completed status through the real worker pool.
func TestGenerateTermsAsyncDispatch(t *testing.T) {
	// Use regexp matcher with ^-anchored patterns so queries are matched by their
	// leading SQL keyword + table, not by a generic substring.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewInMemoryJobStore()
	defer store.Stop()
	svc := NewGlossaryService(context.Background(), db, nil, store)

	tenantID := uuid.New().String()
	colID := uuid.New().String()
	qualifiedPath := "public.test_table.test_column"
	semTypeID := uuid.New().String()
	btTypeID := uuid.New().String()
	semID := uuid.New().String()
	btID := uuid.New().String()
	mapEdgeID := uuid.New().String()
	hasEdgeID := uuid.New().String()

	// Column node lookup
	mock.ExpectQuery(`^SELECT node_name, COALESCE`).WithArgs(colID).WillReturnRows(
		sqlmock.NewRows([]string{"node_name", "qualified_path"}).AddRow("test_column", qualifiedPath))

	// Sibling columns (LIKE pattern)
	mock.ExpectQuery(`^SELECT node_name FROM catalog_node WHERE tenant_id`).WithArgs(tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"node_name"}))

	// resolveOrCreateNodeType semantic_term
	mock.ExpectQuery(`^SELECT id FROM catalog_node_type WHERE catalog_type_name = \$1 LIMIT 1`).WithArgs("semantic_term").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(semTypeID))

	// findOrCreateTermNode semantic — pre-check
	mock.ExpectQuery(`^SELECT id FROM catalog_node WHERE node_type_id = \$1 AND tenant_id = \$2 AND`).WithArgs(semTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// findOrCreateTermNode semantic — INSERT
	mock.ExpectQuery(`^INSERT INTO catalog_node .+ RETURNING id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(semID))

	// resolveOrCreateEdgeType MAPS_TO
	mock.ExpectQuery(`^SELECT id FROM catalog_edge_type WHERE edge_type_name = \$1`).WithArgs("MAPS_TO").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(mapEdgeID))

	// ensureEdge MAPS_TO — pre-check
	mock.ExpectQuery(`^SELECT id FROM catalog_edge WHERE source_node_id = \$1 AND target_node_id = \$2 AND edge_type_id = \$3 LIMIT 1`).WithArgs(semID, colID, mapEdgeID).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// ensureEdge MAPS_TO — INSERT
	mock.ExpectExec(`^INSERT INTO catalog_edge`).WillReturnResult(sqlmock.NewResult(1, 1))

	// resolveOrCreateNodeType business_term
	mock.ExpectQuery(`^SELECT id FROM catalog_node_type WHERE catalog_type_name = \$1 LIMIT 1`).WithArgs("business_term").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(btTypeID))

	// findOrCreateTermNode business — pre-check
	mock.ExpectQuery(`^SELECT id FROM catalog_node WHERE node_type_id = \$1 AND tenant_id = \$2 AND`).WithArgs(btTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// findOrCreateTermNode business — INSERT
	mock.ExpectQuery(`^INSERT INTO catalog_node .+ RETURNING id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(btID))

	// resolveOrCreateEdgeType has_semantic_context
	mock.ExpectQuery(`^SELECT id FROM catalog_edge_type WHERE edge_type_name = \$1`).WithArgs("has_semantic_context").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(hasEdgeID))

	// ensureEdge has_semantic_context — pre-check
	mock.ExpectQuery(`^SELECT id FROM catalog_edge WHERE source_node_id = \$1 AND target_node_id = \$2 AND edge_type_id = \$3 LIMIT 1`).WithArgs(btID, semID, hasEdgeID).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// ensureEdge has_semantic_context — INSERT
	mock.ExpectExec(`^INSERT INTO catalog_edge`).WillReturnResult(sqlmock.NewResult(1, 1))

	// Dispatch 30 items — above threshold → async.
	// All items share the same colID so all goroutines hit the same mocked queries.
	items := make([]generateTermItem, 30)
	for i := 0; i < 30; i++ {
		items[i] = generateTermItem{Name: "TestTerm", ColumnIDs: []string{colID}}
	}

	resp := svc.generateTerms(context.Background(), tenantID, "none", items)
	if resp.JobID == "" {
		t.Fatal("JobID empty; expected async dispatch for 30 items (>25 threshold)")
	}

	// Poll for completion
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var finalJob Job
	for {
		job, found := store.Get(tenantID, resp.JobID)
		if found && (job.Status == JobStatusCompleted || job.Status == JobStatusFailed) {
			finalJob = job
			break
		}
		if ctx.Err() != nil {
			job, _ := store.Get(tenantID, resp.JobID)
			t.Fatalf("timed out: status=%q done=%d failed=%d errors=%v",
				job.Status, job.Done, job.Failed, job.Errors)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 1 goroutine succeeded (mocked colID), 29 failed (no mocks) → completed, done=1
	if finalJob.Status != JobStatusCompleted {
		t.Errorf("status=%q; want completed (failed goroutines: %d)", finalJob.Status, finalJob.Failed)
	}
	if finalJob.Done != 1 {
		t.Errorf("done=%d; want 1 (only the mocked goroutine)", finalJob.Done)
	}
}

func TestGenerateTermsSyncPath(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewInMemoryJobStore()
	defer store.Stop()
	svc := NewGlossaryService(context.Background(), db, nil, store)

	tenantID := uuid.New().String()
	colID := uuid.New().String()
	qualifiedPath := "public.test_table.test_col"
	semTypeID := uuid.New().String()
	btTypeID := uuid.New().String()
	semID := uuid.New().String()
	btID := uuid.New().String()
	mapEdgeID := uuid.New().String()
	hasEdgeID := uuid.New().String()

	mock.ExpectQuery("SELECT node_name, COALESCE\\(qualified_path, ''\\) FROM catalog_node WHERE id = \\$1").
		WithArgs(colID).WillReturnRows(
		sqlmock.NewRows([]string{"node_name", "qualified_path"}).AddRow("test_col", qualifiedPath))

	mock.ExpectQuery("SELECT node_name FROM catalog_node WHERE tenant_id = \\$1 AND qualified_path LIKE \\$2 AND qualified_path != \\$3 LIMIT 40").
		WithArgs(tenantID, "public.test_table.%", qualifiedPath).
		WillReturnRows(sqlmock.NewRows([]string{"node_name"}))

	mock.ExpectQuery("SELECT id FROM catalog_node_type WHERE catalog_type_name = \\$1 LIMIT 1").
		WithArgs("semantic_term").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(semTypeID))

	mock.ExpectQuery("SELECT id FROM catalog_node WHERE node_type_id = \\$1 AND tenant_id = \\$2 AND .* LIMIT 1").
		WithArgs(semTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery("INSERT INTO catalog_node .* ON CONFLICT .* RETURNING id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(semID))

	mock.ExpectQuery("SELECT id FROM catalog_edge_type WHERE edge_type_name = \\$1").
		WithArgs("MAPS_TO").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(mapEdgeID))

	mock.ExpectQuery("SELECT id FROM catalog_edge WHERE source_node_id = \\$1 AND target_node_id = \\$2 AND edge_type_id = \\$3 LIMIT 1").
		WithArgs(semID, colID, mapEdgeID).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectExec("INSERT INTO catalog_edge").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id FROM catalog_node_type WHERE catalog_type_name = \\$1 LIMIT 1").
		WithArgs("business_term").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(btTypeID))

	mock.ExpectQuery("SELECT id FROM catalog_node WHERE node_type_id = \\$1 AND tenant_id = \\$2 AND .* LIMIT 1").
		WithArgs(btTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery("INSERT INTO catalog_node .* ON CONFLICT .* RETURNING id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(btID))

	mock.ExpectQuery("SELECT id FROM catalog_edge_type WHERE edge_type_name = \\$1").
		WithArgs("has_semantic_context").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(hasEdgeID))

	mock.ExpectQuery("SELECT id FROM catalog_edge WHERE source_node_id = \\$1 AND target_node_id = \\$2 AND edge_type_id = \\$3 LIMIT 1").
		WithArgs(btID, semID, hasEdgeID).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectExec("INSERT INTO catalog_edge").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp := svc.generateTerms(context.Background(), tenantID, "none",
		[]generateTermItem{{Name: "SyncTerm", ColumnIDs: []string{colID}}})
	if resp.JobID != "" {
		t.Errorf("JobID=%q; want empty for sync path (1 item)", resp.JobID)
	}
	if !resp.Success {
		t.Error("Success=false; want true")
	}
	if resp.TotalRequested != 1 {
		t.Errorf("TotalRequested=%d; want 1", resp.TotalRequested)
	}
}

func TestInMemoryJobStore_Eviction(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	for i := 0; i < 120; i++ {
		job := &Job{
			TenantID:     "tenant-1",
			DatasourceID: "ds-1",
			Status:       JobStatusCompleted,
			Total:        10,
			Done:         10,
			Failed:       0,
			StartedAt:    time.Now(),
			FinishedAt:   time.Now(),
		}
		store.Create(job)
	}

	store.evict()

	store.mu.Lock()
	count := len(store.jobs)
	store.mu.Unlock()
	if count > maxCompletedJobs {
		t.Errorf("store.jobs count=%d; want <=%d after eviction", count, maxCompletedJobs)
	}
}

func TestInMemoryJobStore_Get_TenantMismatch(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        10,
		Done:         0,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	store.Create(job)

	_, found := store.Get("tenant-1", job.ID)
	if !found {
		t.Error("correct tenant: found=false; want true")
	}

	_, found = store.Get("tenant-2", job.ID)
	if found {
		t.Error("wrong tenant: found=true; want false (404 semantics)")
	}
}
