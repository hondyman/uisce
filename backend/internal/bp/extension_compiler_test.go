package bp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCompiler_MaterializeExtensions(t *testing.T) {
	compiler := NewWorkflowCompiler(nil, nil)

	baseSteps := []WorkflowStep{
		{
			StepID:    "step-init",
			StepName:  "Initialize Account",
			StepType:  "init",
			StepOrder: 1,
			Config:    map[string]interface{}{"timeout_sec": 30, "env": "prod"},
		},
		{
			StepID:    "step-validate",
			StepName:  "Validate Account Schema",
			StepType:  "validate",
			StepOrder: 2,
			Config:    map[string]interface{}{"strict": true},
		},
		{
			StepID:    "step-persist",
			StepName:  "Persist Account Record",
			StepType:  "persist",
			StepOrder: 3,
			Config:    map[string]interface{}{"target_table": "oms.account"},
		},
	}

	t.Run("idempotent re-expansion produces identical results", func(t *testing.T) {
		extensions := []ExtensionSpec{
			{
				AnchorID:  "step-validate",
				Operation: OpInsertAfter,
				Steps: []WorkflowStep{
					{
						StepID:   "step-custom-aml",
						StepName: "Tenant AML Check",
						StepType: "custom_activity",
						Config:   map[string]interface{}{"provider": "lexisnexis"},
					},
				},
			},
			{
				AnchorID:  "step-init",
				Operation: OpConfigOverride,
				ConfigOverrides: map[string]interface{}{
					"timeout_sec": 60,
					"custom_flag": "enabled",
				},
			},
		}

		res1, err1 := compiler.MaterializeExtensions(baseSteps, extensions)
		require.NoError(t, err1)

		res2, err2 := compiler.MaterializeExtensions(baseSteps, extensions)
		require.NoError(t, err2)

		bytes1, _ := json.Marshal(res1)
		bytes2, _ := json.Marshal(res2)
		assert.JSONEq(t, string(bytes1), string(bytes2), "Repeated extension expansion must be 100% idempotent")

		// Verify structure
		require.Len(t, res1, 4)
		assert.Equal(t, "step-init", res1[0].StepID)
		assert.Equal(t, 1, res1[0].StepOrder)
		assert.Equal(t, 60, res1[0].Config["timeout_sec"])
		assert.Equal(t, "enabled", res1[0].Config["custom_flag"])
		assert.Equal(t, "prod", res1[0].Config["env"], "Non-overridden config must be preserved")

		assert.Equal(t, "step-validate", res1[1].StepID)
		assert.Equal(t, 2, res1[1].StepOrder)

		assert.Equal(t, "step-custom-aml", res1[2].StepID)
		assert.Equal(t, 3, res1[2].StepOrder)

		assert.Equal(t, "step-persist", res1[3].StepID)
		assert.Equal(t, 4, res1[3].StepOrder)
	})

	t.Run("missing anchor returns ErrExtensionAnchorNotFound", func(t *testing.T) {
		extensions := []ExtensionSpec{
			{
				AnchorID:  "non-existent-anchor",
				Operation: OpInsertAfter,
				Steps: []WorkflowStep{
					{
						StepID:   "step-orphan",
						StepName: "Orphan Step",
						StepType: "validate",
					},
				},
			},
		}

		res, err := compiler.MaterializeExtensions(baseSteps, extensions)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, ErrExtensionAnchorNotFound)
		assert.Contains(t, err.Error(), "non-existent-anchor")
	})

	t.Run("APPEND_BRANCH attaches branch condition and preserves branch order", func(t *testing.T) {
		cond := "input.amount > 1000000"
		extensions := []ExtensionSpec{
			{
				AnchorID:        "step-validate",
				Operation:       OpAppendBranch,
				BranchCondition: &cond,
				Steps: []WorkflowStep{
					{
						StepID:   "step-high-net-worth-approval",
						StepName: "HNW Principal Approval",
						StepType: "approval",
					},
				},
			},
		}

		res, err := compiler.MaterializeExtensions(baseSteps, extensions)
		require.NoError(t, err)
		require.Len(t, res, 4)

		assert.Equal(t, "step-high-net-worth-approval", res[2].StepID)
		assert.Equal(t, 3, res[2].StepOrder)
		assert.Equal(t, "input.amount > 1000000", res[2].Config["branch_condition"])
	})
}

func TestWorkflowCompiler_CompileAndPublish_UnitMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	compiler := NewWorkflowCompiler(db, nil)
	ctx := context.Background()

	tenantID := "11111111-1111-1111-1111-111111111111"
	baseDefID := uuid.New()

	baseSteps := []WorkflowStep{
		{StepID: "b1", StepName: "Base Step 1", StepType: "init", StepOrder: 1},
		{StepID: "b2", StepName: "Base Step 2", StepType: "persist", StepOrder: 2},
	}
	baseStepsBytes, _ := json.Marshal(baseSteps)

	// 1. Mock load base definition tx
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('app\.current_tenant', \$1, true\),`).
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	baseRow := sqlmock.NewRows([]string{
		"id", "process_id", "tenant_id", "version", "name", "steps_json", "graph_json", "source_type",
	}).AddRow(baseDefID, "proc-core-account", "00000000-0000-0000-0000-000000000001", 3, "Core Account Workflow", baseStepsBytes, []byte("{}"), "CORE")

	mock.ExpectQuery(`SELECT id, process_id, tenant_id, version, name, steps_json, graph_json, source_type FROM public\.bp_process_definition WHERE id = \$1 OR process_id = \$1::text ORDER BY version DESC LIMIT 1`).
		WithArgs(baseDefID).
		WillReturnRows(baseRow)
	mock.ExpectRollback()

	// 2. Mock compile write tx
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('app\.current_tenant', \$1, true\),`).
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(version\), 0\) \+ 1 FROM public\.bp_process_definition WHERE process_id = \$1`).
		WithArgs("proc-custom-account").
		WillReturnRows(sqlmock.NewRows([]string{"next_ver"}).AddRow(1))

	insRow := sqlmock.NewRows([]string{
		"id", "process_id", "tenant_id", "version", "name", "steps_json", "graph_json", "source_type",
		"base_definition_id", "base_version", "extensions_json", "created_at",
	}).AddRow(
		uuid.New(), "proc-custom-account", tenantID, 1, "Extended Account Workflow",
		[]byte(`[{"step_id":"b1"},{"step_id":"ext1"},{"step_id":"b2"}]`), []byte(`{}`), "EXTENDED",
		baseDefID.String(), 3, []byte(`[]`), time.Now(),
	)

	mock.ExpectQuery(`INSERT INTO public\.bp_process_definition`).
		WillReturnRows(insRow)

	mock.ExpectExec(`INSERT INTO public\.bp_trigger_subscriptions`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	req := CompileProcessRequest{
		TenantID:         tenantID,
		ProcessID:        "proc-custom-account",
		Name:             "Extended Account Workflow",
		SourceType:       "EXTENDED",
		BaseDefinitionID: &baseDefID,
		Extensions: []ExtensionSpec{
			{
				AnchorID:  "b1",
				Operation: OpInsertAfter,
				Steps: []WorkflowStep{
					{StepID: "ext1", StepName: "Extension Step 1", StepType: "custom"},
				},
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.events",
	}

	persisted, err := compiler.CompileAndPublish(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, "proc-custom-account", persisted.ProcessID)
	assert.Equal(t, "EXTENDED", persisted.SourceType)
	assert.Equal(t, &baseDefID, persisted.BaseDefinitionID)
	assert.Equal(t, 3, *persisted.BaseVersion)
}

func TestTriggerSubscriptionStore_Match(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewTriggerSubscriptionStore(db)
	ctx := context.Background()

	tenantID1 := uuid.New()
	tenantID2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"tenant_id", "process_id", "trigger_type", "topic", "event_type", "ops", "version", "updated_at",
	}).AddRow(
		tenantID1, "proc-account-lifecycle", "event", "oms.account.events", "ACCOUNT_CREATED", []byte(`["c", "u"]`), 2, time.Now(),
	).AddRow(
		tenantID2, "proc-account-audit", "event", "oms.account.events", nil, nil, 1, time.Now(),
	)

	mock.ExpectQuery(`SELECT tenant_id, process_id, trigger_type, topic, event_type, ops, version, updated_at FROM public\.bp_trigger_subscriptions WHERE topic = \$1`).
		WithArgs("oms.account.events", "ACCOUNT_CREATED").
		WillReturnRows(rows)

	matches, err := store.MatchSubscriptions(ctx, "oms.account.events", "ACCOUNT_CREATED", "c")
	require.NoError(t, err)
	require.Len(t, matches, 2)
	assert.Equal(t, tenantID1, matches[0].TenantID)
	assert.Equal(t, "proc-account-lifecycle", matches[0].ProcessID)
	assert.Equal(t, tenantID2, matches[1].TenantID)
	assert.Equal(t, "proc-account-audit", matches[1].ProcessID)
}

