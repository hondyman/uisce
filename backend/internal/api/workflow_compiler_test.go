package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/bp"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

func TestWorkflowCompilerHandler_Unit(t *testing.T) {
	t.Run("HandleCompile returns 422 when extension anchor is missing", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		handler := NewWorkflowCompilerHandler(db, handlers.SecurityContextDeps{})

		tenantID := "11111111-1111-1111-1111-111111111111"
		baseDefID := uuid.New()

		baseSteps := []bp.WorkflowStep{
			{StepID: "b1", StepName: "Base Step 1", StepType: "init", StepOrder: 1},
		}
		baseStepsBytes, _ := json.Marshal(baseSteps)

		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config\('app\.current_tenant', \$1, true\),`).
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		baseRow := sqlmock.NewRows([]string{
			"id", "process_id", "tenant_id", "version", "name", "steps_json", "graph_json", "source_type",
		}).AddRow(baseDefID, "proc-core-acc", "00000000-0000-0000-0000-000000000001", 1, "Core Acc", baseStepsBytes, []byte("{}"), "CORE")

		mock.ExpectQuery(`SELECT id, process_id, tenant_id, version, name, steps_json, graph_json, source_type FROM public\.bp_process_definition`).
			WithArgs(baseDefID).
			WillReturnRows(baseRow)
		mock.ExpectRollback()

		compileReq := bp.CompileProcessRequest{
			TenantID:         tenantID,
			ProcessID:        "proc-custom-acc",
			Name:             "Custom Acc",
			SourceType:       "EXTENDED",
			BaseDefinitionID: &baseDefID,
			Extensions: []bp.ExtensionSpec{
				{
					AnchorID:  "non-existent-anchor",
					Operation: bp.OpInsertAfter,
					Steps:     []bp.WorkflowStep{{StepID: "s2", StepName: "Step 2"}},
				},
			},
		}

		body, _ := json.Marshal(compileReq)
		req := httptest.NewRequest("POST", "/api/bp/compile", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.HandleCompile(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "EXTENSION_ANCHOR_NOT_FOUND", resp["code"])
	})

	t.Run("HandleExecute returns 412 when process definition not found for tenant", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		handler := NewWorkflowCompilerHandler(db, handlers.SecurityContextDeps{})

		tenantID := "22222222-2222-2222-2222-222222222222"
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config\('app\.current_tenant', \$1, true\),`).
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectQuery(`SELECT id, version FROM public\.bp_process_definition WHERE process_id = \$1`).
			WithArgs("proc-missing").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		execReq := ExecuteWorkflowRequest{
			TenantID:  tenantID,
			ProcessID: "proc-missing",
		}
		body, _ := json.Marshal(execReq)
		req := httptest.NewRequest("POST", "/api/bp/execute", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.HandleExecute(rec, req)

		assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "PRECONDITION_FAILED", resp["code"])
	})
}

func getLivePostgresDBForAPI(t *testing.T) *sql.DB {
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

func TestLiveWorkflowCompilerHandler_HTTPPath_EndToEnd(t *testing.T) {
	db := getLivePostgresDBForAPI(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Resolve core tenant ID
	var coreTenantID string
	err := db.QueryRowContext(ctx, "SELECT public.get_core_tenant_id()").Scan(&coreTenantID)
	require.NoError(t, err)

	clientTenantID := uuid.New().String()
	coreProcessID := fmt.Sprintf("core-http-proc-%d", time.Now().UnixNano())
	clientProcessID := fmt.Sprintf("client-http-proc-%d", time.Now().UnixNano())

	// Demote session role to app_user (non-superuser with RLS active)
	// Clear session GUC to simulate un-initialized worker/API pool connection
	_, err = db.ExecContext(ctx, "SET ROLE app_user; SET app.current_tenant = ''; SET uisce.current_tenant = '';")
	require.NoError(t, err)

	handler := NewWorkflowCompilerHandler(db, handlers.SecurityContextDeps{})

	defer func() {
		// Cleanup with superuser
		_, _ = db.ExecContext(ctx, "SET ROLE postgres;")
		_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_process_definition WHERE process_id IN ($1, $2)", coreProcessID, clientProcessID)
		_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_trigger_subscriptions WHERE process_id IN ($1, $2)", coreProcessID, clientProcessID)
	}()

	// 2. HTTP POST /api/bp/compile for CORE workflow
	coreReqPayload := bp.CompileProcessRequest{
		TenantID:   coreTenantID,
		ProcessID:  coreProcessID,
		Name:       "Core Account Lifecycle Process",
		SourceType: "CORE",
		Steps: []bp.WorkflowStep{
			{
				StepID:    "core-step-1",
				StepName:  "Core Initialize",
				StepType:  "init",
				StepOrder: 1,
			},
			{
				StepID:    "core-step-2",
				StepName:  "Core Validate",
				StepType:  "validate",
				StepOrder: 2,
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.events",
	}

	body, _ := json.Marshal(coreReqPayload)
	req := httptest.NewRequest("POST", "/api/bp/compile", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleCompile(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "Compile CORE via HTTP handler must return 201 Created under app_user")

	var coreDef bp.ProcessDefinition
	err = json.Unmarshal(rec.Body.Bytes(), &coreDef)
	require.NoError(t, err)
	assert.Equal(t, 1, coreDef.Version)
	assert.Equal(t, "CORE", coreDef.SourceType)

	// 3. HTTP POST /api/bp/compile for EXTENDED workflow
	extReqPayload := bp.CompileProcessRequest{
		TenantID:         clientTenantID,
		ProcessID:        clientProcessID,
		Name:             "Client Custom Account Process",
		SourceType:       "EXTENDED",
		BaseDefinitionID: &coreDef.ID,
		Extensions: []bp.ExtensionSpec{
			{
				AnchorID:  "core-step-2",
				Operation: bp.OpInsertAfter,
				Steps: []bp.WorkflowStep{
					{
						StepID:   "client-aml-step",
						StepName: "Client AML Check",
						StepType: "custom",
					},
				},
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.client_events",
	}

	extBody, _ := json.Marshal(extReqPayload)
	extReq := httptest.NewRequest("POST", "/api/bp/compile", bytes.NewReader(extBody))
	extRec := httptest.NewRecorder()

	handler.HandleCompile(extRec, extReq)
	require.Equal(t, http.StatusCreated, extRec.Code, "Compile EXTENDED via HTTP handler must return 201 Created under app_user")

	var clientDef bp.ProcessDefinition
	err = json.Unmarshal(extRec.Body.Bytes(), &clientDef)
	require.NoError(t, err)
	assert.Equal(t, 1, clientDef.Version)
	assert.Equal(t, "EXTENDED", clientDef.SourceType)
	assert.Equal(t, &coreDef.ID, clientDef.BaseDefinitionID)
	assert.Equal(t, 1, *clientDef.BaseVersion)

	// 4. Verify trigger subscription row exists and was written by HTTP compile path
	subsStore := bp.NewTriggerSubscriptionStore(db)
	matches, err := subsStore.MatchSubscriptions(ctx, "oms.account.client_events", "", "")
	require.NoError(t, err)
	foundClientSub := false
	for _, m := range matches {
		if m.ProcessID == clientProcessID {
			foundClientSub = true
			break
		}
	}
	assert.True(t, foundClientSub, "bp_trigger_subscriptions must contain subscription written by HTTP compile handler")

	// 5. HTTP POST /api/bp/execute pre-flight validation
	execReqPayload := ExecuteWorkflowRequest{
		TenantID:  clientTenantID,
		ProcessID: clientProcessID,
	}
	execBody, _ := json.Marshal(execReqPayload)
	execReq := httptest.NewRequest("POST", "/api/bp/execute", bytes.NewReader(execBody))
	execRec := httptest.NewRecorder()

	handler.HandleExecute(execRec, execReq)
	require.Equal(t, http.StatusOK, execRec.Code, "Execute pre-flight must succeed for authorized tenant")

	// 6. HTTP POST /api/bp/execute for a foreign tenant without access must return 412
	foreignExecPayload := ExecuteWorkflowRequest{
		TenantID:  uuid.New().String(),
		ProcessID: clientProcessID,
	}
	foreignBody, _ := json.Marshal(foreignExecPayload)
	foreignReq := httptest.NewRequest("POST", "/api/bp/execute", bytes.NewReader(foreignBody))
	foreignRec := httptest.NewRecorder()

	handler.HandleExecute(foreignRec, foreignReq)
	require.Equal(t, http.StatusPreconditionFailed, foreignRec.Code, "Execute pre-flight for foreign tenant must return 412 Precondition Failed")
}
