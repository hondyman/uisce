package bp

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

func getLivePostgresDB(t *testing.T) *sql.DB {
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

func TestLiveWorkflowCompiler_RLS_And_Extensions_EndToEnd(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Resolve core tenant ID
	var coreTenantID string
	err := db.QueryRowContext(ctx, "SELECT public.get_core_tenant_id()").Scan(&coreTenantID)
	require.NoError(t, err)

	clientTenantID := uuid.New().String()
	coreProcessID := fmt.Sprintf("core-acc-proc-%d", time.Now().UnixNano())
	clientProcessID := fmt.Sprintf("client-acc-proc-%d", time.Now().UnixNano())

	// Demote connection session role to app_user (non-superuser with RLS active)
	// Clear any session-level GUC to simulate un-initialized worker/API pool connection
	_, err = db.ExecContext(ctx, "SET ROLE app_user; SET app.current_tenant = ''; SET uisce.current_tenant = '';")
	require.NoError(t, err)

	compiler := NewWorkflowCompiler(db, nil)

	// Clean up after test
	defer func() {
		// Restore superuser for cleanup
		_, _ = db.ExecContext(ctx, "SET ROLE postgres;")
		_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_process_definition WHERE process_id IN ($1, $2)", coreProcessID, clientProcessID)
		_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_trigger_subscriptions WHERE process_id IN ($1, $2)", coreProcessID, clientProcessID)
	}()

	// 2. Compile CORE definition under core tenant
	coreReq := CompileProcessRequest{
		TenantID:   coreTenantID,
		ProcessID:  coreProcessID,
		Name:       "Core Account Lifecycle Process",
		SourceType: "CORE",
		Steps: []WorkflowStep{
			{
				StepID:    "core-init",
				StepName:  "Initialize Core Account",
				StepType:  "init",
				StepOrder: 1,
				Config:    map[string]interface{}{"max_accounts": 100},
			},
			{
				StepID:    "core-validate",
				StepName:  "Validate Schema",
				StepType:  "validate",
				StepOrder: 2,
				Config:    map[string]interface{}{"strict": true},
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.created",
	}

	coreDef, err := compiler.CompileAndPublish(ctx, coreReq)
	require.NoError(t, err, "Compile CORE process must succeed under app_user with transaction SetRLSContext")
	require.NotNil(t, coreDef)
	assert.Equal(t, 1, coreDef.Version)
	assert.Equal(t, "CORE", coreDef.SourceType)

	// 3. Compile EXTENDED definition under client tenant attaching delta to core anchor
	extReq := CompileProcessRequest{
		TenantID:         clientTenantID,
		ProcessID:        clientProcessID,
		Name:             "Client Custom Account Process",
		SourceType:       "EXTENDED",
		BaseDefinitionID: &coreDef.ID,
		Extensions: []ExtensionSpec{
			{
				AnchorID:  "core-validate",
				Operation: OpInsertAfter,
				Steps: []WorkflowStep{
					{
						StepID:   "client-aml",
						StepName: "Client Sanctions & AML Screening",
						StepType: "custom_activity",
						Config:   map[string]interface{}{"aml_vendor": "veriff"},
					},
				},
			},
			{
				AnchorID:  "core-init",
				Operation: OpConfigOverride,
				ConfigOverrides: map[string]interface{}{
					"max_accounts": 500,
					"custom_flag":  true,
				},
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.custom_event",
	}

	clientDef, err := compiler.CompileAndPublish(ctx, extReq)
	require.NoError(t, err, "Compile EXTENDED process must succeed reading core base under RLS")
	require.NotNil(t, clientDef)
	assert.Equal(t, 1, clientDef.Version)
	assert.Equal(t, "EXTENDED", clientDef.SourceType)
	assert.Equal(t, &coreDef.ID, clientDef.BaseDefinitionID)
	assert.Equal(t, 1, *clientDef.BaseVersion)

	// 4. Verify trigger subscription store matched topics across tenants without RLS bypass
	subsStore := NewTriggerSubscriptionStore(db)
	matches, err := subsStore.MatchSubscriptions(ctx, "oms.account.created", "", "")
	require.NoError(t, err)
	foundCoreSub := false
	for _, m := range matches {
		if m.ProcessID == coreProcessID {
			foundCoreSub = true
			break
		}
	}
	assert.True(t, foundCoreSub, "Trigger subscription for core process must be indexed and matched")

	clientMatches, err := subsStore.MatchSubscriptions(ctx, "oms.account.custom_event", "", "")
	require.NoError(t, err)
	foundClientSub := false
	for _, m := range clientMatches {
		if m.ProcessID == clientProcessID {
			foundClientSub = true
			break
		}
	}
	assert.True(t, foundClientSub, "Trigger subscription for client process must be indexed and matched")
}
