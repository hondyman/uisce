package workflows

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getLiveWorkflowDB(t *testing.T) *sql.DB {
	if os.Getenv("BP_LIVE_SMOKE") != "1" {
		t.Skip("skipping live postgres RLS smoke test: set BP_LIVE_SMOKE=1 to run against alpha")
	}

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	certDir := filepath.Join(home, ".uisce", "certs")
	caCertPath := filepath.Join(certDir, "ca.crt")
	clientCertPath := filepath.Join(certDir, "postgres-client.crt")
	clientKeyPath := filepath.Join(certDir, "postgres-client.key")

	caCert, err := os.ReadFile(caCertPath)
	require.NoError(t, err, "failed to read ca.crt")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	require.NoError(t, err, "failed to load postgres client key pair")

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	connStr := "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full"
	config, err := pgx.ParseConfig(connStr)
	require.NoError(t, err)
	config.TLSConfig = tlsConfig

	return stdlib.OpenDB(*config)
}

func TestLiveWorkerLoadBPStepsActivity_RLS_Regression(t *testing.T) {
	db := getLiveWorkflowDB(t)
	defer db.Close()

	ctx := context.Background()
	testTenantID := uuid.New()
	testProcessID := fmt.Sprintf("proc-%d", time.Now().UnixNano())

	// 1. Seed process definition directly on alpha
	_, err := db.ExecContext(ctx, `
		INSERT INTO public.bp_process_definition (
			id, process_id, tenant_id, version, name, steps_json, graph_json, source_type
		) VALUES (
			gen_random_uuid(), $1, $2, 1, 'Worker Activity Test Process',
			'[{"step_id": "s1", "step_name": "Step 1", "step_type": "validate", "step_order": 1}]'::jsonb,
			'{}'::jsonb, 'CORE'
		);
	`, testProcessID, testTenantID.String())
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_process_definition WHERE process_id = $1", testProcessID)
	}()

	// 2. Set database session role to app_user (non-superuser with RLS active)
	// Clear any session-level GUC to simulate un-initialized worker pool connection
	_, err = db.ExecContext(ctx, "SET ROLE app_user; SET app.current_tenant = '';")
	require.NoError(t, err)

	// 3. Test LoadBPStepsActivity through workflows package
	wfActivities := NewActivities(db)
	steps, err := wfActivities.LoadBPStepsActivity(ctx, testProcessID, testTenantID.String())
	require.NoError(t, err)
	require.Len(t, steps, 1, "LoadBPStepsActivity must find the step using its internal tx GUC scoping")
	assert.Equal(t, "s1", steps[0].StepID)
	assert.Equal(t, "Step 1", steps[0].StepName)
}
