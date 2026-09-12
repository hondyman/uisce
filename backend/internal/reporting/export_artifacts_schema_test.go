package reporting_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("UISCE_TEST_DB") == "" {
		t.Skip("Skipping live database integration tests. Set UISCE_TEST_DB=1 to run.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("UISCE_TEST_DB is set but DATABASE_URL is missing.")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "failed to open database")

	err = db.Ping()
	require.NoError(t, err, "failed to connect to database")

	t.Cleanup(func() { _ = db.Close() })
	return db
}

func getOrCreateTestTemplate(t *testing.T, db *sql.DB, tenantID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := db.QueryRow(`
		SELECT id FROM report_templates WHERE tenant_id = $1 LIMIT 1
	`, tenantID).Scan(&id)
	if err == nil {
		return id
	}
	if err != sql.ErrNoRows {
		t.Fatalf("template lookup failed: %v", err)
	}
	id = uuid.New()
	_, err = db.Exec(`
		INSERT INTO report_templates (id, tenant_id, template_name, is_active, is_public)
		VALUES ($1, $2, 'integration-test-template', true, false)
	`, id, tenantID)
	require.NoError(t, err, "failed to insert test template")
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM report_templates WHERE id = $1", id) })
	return id
}

func cleanupExport(db *sql.DB, exportIDs []uuid.UUID) {
	_, _ = db.Exec(`DELETE FROM export_artifact_events WHERE export_id = ANY($1)`,
		exportIDs)
	_, _ = db.Exec(`DELETE FROM export_artifacts WHERE id = ANY($1)`,
		exportIDs)
}

func setupExportArtifact(ctx context.Context, t *testing.T, db *sql.DB, tenantID uuid.UUID, templateID uuid.UUID) uuid.UUID {
	exportID := uuid.New()
	storageKey := "exports/test/" + templateID.String() + "/" + exportID.String() + ".pdf"
	exporterID := "test-exporter-" + exportID.String()[:8]

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "failed to begin tx")
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID.String())
	require.NoError(t, err, "failed to set tenant")

	_, err = tx.ExecContext(ctx, `
		INSERT INTO export_artifacts
			(id, template_id, exporter_user_id, tenant_id, status, format,
			 storage_key, export_classification)
		VALUES ($1, $2, $3, $4, 'pending', 'pdf', $5, 'internal')
	`, exportID, templateID, exporterID, tenantID, storageKey)
	require.NoError(t, err, "failed to insert export_artifacts row")

	eventID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO export_artifact_events
			(id, export_id, tenant_id, event, from_status, to_status, actor_id, detail)
		VALUES ($1, $2, $3, 'PENDING', NULL, 'pending', $4, '{}')
	`, eventID, exportID, tenantID, exporterID)
	require.NoError(t, err, "failed to insert export_artifact_events row")

	require.NoError(t, tx.Commit(), "commit failed")
	t.Cleanup(func() { cleanupExport(db, []uuid.UUID{exportID}) })
	return exportID
}

func TestExportArtifacts_RLS_WITH_CHECK_rejectsWrongTenant(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	tenantB := uuid.MustParse("87ee7d8f-7b57-4119-a46d-b01e7e405b5b")
	templateA := getOrCreateTestTemplate(t, db, tenantA)

	exportID := uuid.New()
	exporterID := "test-exporter-" + exportID.String()[:8]
	storageKey := "exports/test/" + templateA.String() + "/" + exportID.String() + ".pdf"

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO export_artifacts
			(id, template_id, exporter_user_id, tenant_id, status, format,
			 storage_key, export_classification)
		VALUES ($1, $2, $3, $4, 'pending', 'pdf', $5, 'internal')
	`, exportID, templateA, exporterID, tenantA, storageKey)
	require.NoError(t, err, "insert as tenantA should succeed")

	_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO export_artifacts
			(id, template_id, exporter_user_id, tenant_id, status, format,
			 storage_key, export_classification)
		VALUES ($1, $2, $3, $4, 'pending', 'pdf', $5, 'internal')
	`, uuid.New(), templateA, exporterID, tenantB, storageKey+"wrong")
	assert.Error(t, err, "INSERT with tenant_id=tenantB while current_tenant=tenantA must be rejected by WITH CHECK policy")
	t.Logf("WITH CHECK rejection confirmed: %v", err)

	_ = tx.Rollback()
}

func TestExportArtifactEvents_RLS_WITH_CHECK_rejectsWrongTenant(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	tenantB := uuid.MustParse("87ee7d8f-7b57-4119-a46d-b01e7e405b5b")
	templateA := getOrCreateTestTemplate(t, db, tenantA)
	exportID := setupExportArtifact(ctx, t, db, tenantA, templateA)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO export_artifact_events
			(id, export_id, tenant_id, event, from_status, to_status, actor_id, detail)
		VALUES ($1, $2, $3, 'DOWNLOADED', 'completed', 'completed', $4, '{}')
	`, uuid.New(), exportID, tenantB, "test-actor-b")
	assert.Error(t, err, "INSERT into events with tenant_id=tenantB while current_tenant=tenantA must be rejected by WITH CHECK policy")
	t.Logf("WITH CHECK rejection confirmed: %v", err)

	_ = tx.Rollback()
}

func TestExportArtifacts_ColumnScopedUPDATE_blocksStorageKey(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	templateA := getOrCreateTestTemplate(t, db, tenantA)
	exportID := setupExportArtifact(ctx, t, db, tenantA, templateA)

	{
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
		require.NoError(t, err)

		_, err = tx.ExecContext(ctx, `
			UPDATE export_artifacts SET storage_key = 'hacked/path/evil.pdf'
			WHERE id = $1
		`, exportID)
		assert.Error(t, err, "UPDATE on storage_key (no grant) must error as app_user")
		t.Logf("storage_key UPDATE blocked: %v", err)
	}

	{
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
		require.NoError(t, err)

		_, err = tx.ExecContext(ctx, `
			UPDATE export_artifacts SET status = 'completed'
			WHERE id = $1
		`, exportID)
		assert.NoError(t, err, "UPDATE on status (granted column) must succeed as app_user")
		t.Log("status UPDATE permitted: confirmed")
	}
}

func TestExportArtifacts_FK_NO_ACTION_blocksDeleteWithEvents(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	templateA := getOrCreateTestTemplate(t, db, tenantA)
	exportID := setupExportArtifact(ctx, t, db, tenantA, templateA)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM export_artifacts WHERE id = $1
	`, exportID)
	assert.Error(t, err, "DELETE on export_artifacts with existing events must fail due to FK ON DELETE NO ACTION")
	t.Logf("FK NO ACTION confirmed: %v", err)

	_ = tx.Rollback()
}

func TestExportArtifacts_TTLSweeperCandidateQuery(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	templateA := getOrCreateTestTemplate(t, db, tenantA)

	exportIDA := uuid.New()
	exportIDB := uuid.New()
	exportIDC := uuid.New()

	exportIDs := []uuid.UUID{exportIDA, exportIDB, exportIDC}
	t.Cleanup(func() { cleanupExport(db, exportIDs) })

	{
		dbPre, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
		require.NoError(t, err)
		_, _ = dbPre.Exec(`DELETE FROM export_artifact_events WHERE export_id IN (SELECT id FROM export_artifacts WHERE storage_key LIKE 'exports/test/%')`)
		_, _ = dbPre.Exec(`DELETE FROM export_artifacts WHERE storage_key LIKE 'exports/test/%'`)
		dbPre.Close()
	}

	db2, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer db2.Close()

	{
		tx, err := db2.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO export_artifacts
				(id, template_id, exporter_user_id, tenant_id, status, format,
				 storage_key, export_classification, expires_at)
			VALUES
				($1, $2, 'test-exporter', $3, 'completed', 'pdf',
				 'exports/test/a.pdf', 'internal',
				 NOW() - INTERVAL '100 days'),
				($4, $2, 'test-exporter', $3, 'completed', 'pdf',
				 'exports/test/b.pdf', 'confidential',
				 NOW() - INTERVAL '50 days'),
				($5, $2, 'test-exporter', $3, 'completed', 'pdf',
				 'exports/test/c.pdf', 'public', NULL)
		`, exportIDA, templateA, tenantA, exportIDB, exportIDC)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
	}

	{
		tx, err := db2.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO export_artifact_events
				(id, export_id, tenant_id, event, from_status, to_status, actor_id, detail)
			VALUES ($1, $2, $3, 'EXPIRED', NULL, 'completed', 'test-exporter', '{}')
		`, uuid.New(), exportIDB, tenantA)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
	}

	var candidateCount int
	db3, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer db3.Close()

	err = db3.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM export_artifacts e
		WHERE e.expires_at < NOW()
		  AND e.status = 'completed'
		  AND NOT EXISTS (
			  SELECT 1 FROM export_artifact_events ev
			  WHERE ev.export_id = e.id AND ev.event = 'EXPIRED'
		  )
	`).Scan(&candidateCount)
	require.NoError(t, err, "sweeper candidate query failed")

	assert.Equal(t, 1, candidateCount, "exactly 1 sweeper candidate expected (exportA); exportB has EXPIRED event, exportC has NULL expires_at")
	t.Logf("Sweeper candidate count: %d (expected 1)", candidateCount)

	var candidateID uuid.UUID
	err = db3.QueryRowContext(ctx, `
		SELECT e.id FROM export_artifacts e
		WHERE e.expires_at < NOW()
		  AND e.status = 'completed'
		  AND NOT EXISTS (
			  SELECT 1 FROM export_artifact_events ev
			  WHERE ev.export_id = e.id AND ev.event = 'EXPIRED'
		  )
	`).Scan(&candidateID)
	require.NoError(t, err, "sweeper candidate ID query failed")
	assert.Equal(t, exportIDA, candidateID, "sweeper candidate should be exportIDA (expired, no EXPIRED event)")
	t.Logf("Sweeper candidate ID: %s (expected %s)", candidateID, exportIDA)
}

func TestExportArtifactEvents_InsertOnly_noUPDATE_noDELETE(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	tenantA := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	templateA := getOrCreateTestTemplate(t, db, tenantA)
	exportID := setupExportArtifact(ctx, t, db, tenantA, templateA)

	var eventID uuid.UUID
	err := db.QueryRowContext(ctx, `
		SELECT id FROM export_artifact_events WHERE export_id = $1 LIMIT 1
	`, exportID).Scan(&eventID)
	require.NoError(t, err, "no events found for export")

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `SET ROLE app_user`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		UPDATE export_artifact_events SET event = 'HACKED' WHERE id = $1
	`, eventID)
	assert.Error(t, err, "UPDATE on export_artifact_events must fail (no grant)")
	t.Logf("UPDATE blocked: %v", err)

	_, err = tx.ExecContext(ctx, `
		DELETE FROM export_artifact_events WHERE id = $1
	`, eventID)
	assert.Error(t, err, "DELETE on export_artifact_events must fail (no grant)")
	t.Logf("DELETE blocked: %v", err)

	_ = tx.Rollback()
}
