-- Real persistence for pkg/workflows' WorkflowDefinition DSL. RunStoredWorkflow
-- (backend/pkg/workflows/dynamic_bp_workflow.go) previously always ran one of a
-- handful of hardcoded demo definitions selected by a Go switch on workflow_key
-- string literals ("bp_genai_demo", "bp_risk_demo", "bp_rwa_issuance", else a
-- fixed 3-node compliance+MDM demo) -- there was no way to store or author a
-- real BP definition at all. This table is the minimal real storage: one row
-- per named, versioned definition, loaded at workflow-start time via
-- ActivityLoadWorkflowDefinition (see workflow_definition_activities.go)
-- instead of being compiled into the binary.

CREATE TABLE IF NOT EXISTS workflow_definitions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NULL, -- NULL = core/shared definition, visible to every tenant
    workflow_key TEXT NOT NULL,
    name         TEXT NOT NULL,
    definition   JSONB NOT NULL, -- serialized workflows.WorkflowDefinition
    version      INTEGER NOT NULL DEFAULT 1,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One active version per (tenant scope, key): a tenant-specific row with the
-- same workflow_key as a core row shadows it (see ActivityLoadWorkflowDefinition's
-- tenant-then-core lookup order).
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_definitions_active_key
    ON workflow_definitions (workflow_key, COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'))
    WHERE is_active = TRUE;

COMMENT ON TABLE workflow_definitions IS
    'Real storage for the pkg/workflows graph DSL, loaded by ActivityLoadWorkflowDefinition at RunStoredWorkflow start instead of a hardcoded Go switch on workflow_key.';
COMMENT ON COLUMN workflow_definitions.workflow_key IS
    'Matches InterpreterInput.WorkflowID -- the id callers (pipelines_handler.go, events_api.go) pass when starting RunStoredWorkflow.';

-- Seed rows preserving the exact behavior of the demo definitions
-- RunStoredWorkflow used to hardcode, so existing callers (IngestMarketEvent's
-- "bp_margin_call_protocol"/"bp_kyc_refresh" triggers, and any caller of
-- "bp_genai_demo"/"bp_risk_demo"/"bp_rwa_issuance") keep working unchanged.
INSERT INTO workflow_definitions (tenant_id, workflow_key, name, definition, version) VALUES
-- node_2 (ActivityValidateGoldenRecord) was deliberately dropped: that
-- activity (pkg/workflows/mdm_activities.go) is fully mocked, hardcoded to
-- one fake entity ID ("CP-123") -- there is no real golden-record store
-- behind it yet (see internal/mdm/rule_validation_bridge.go's doc comment;
-- SurvivorshipEngine merges caller-supplied candidate sources, it does not
-- fetch a stored "current golden value" by entity ID). Wiring RunStoredWorkflow
-- to real storage made this the default path for every unrecognized
-- workflow_key, which would have put fake data on real traffic silently.
(NULL, '__default__', 'Compliance Check', $json$
{
    "name": "__default__",
    "startNodeId": "node_1",
    "nodes": {
        "node_1": {"id": "node_1", "type": "ACTIVITY", "config": {"activityName": "ActivityCheckCompliance"}, "nextNodeId": "node_3"},
        "node_3": {"id": "node_3", "type": "END"}
    }
}
$json$::jsonb, 1),
(NULL, 'bp_genai_demo', 'GenAI Co-pilot Demo', $json$
{
    "name": "GenAI Co-pilot Demo",
    "startNodeId": "node_analyze",
    "nodes": {
        "node_analyze": {"id": "node_analyze", "type": "ACTIVITY", "config": {
            "activityName": "ActivityGenerateContent",
            "promptTemplate": "Review the following transaction for compliance risks. Return a JSON with 'risk_level' and 'reason'. Transaction: {{.trade_details}}",
            "systemInstruction": "You are a senior compliance officer. You are strict.",
            "modelOverride": "gemini-1.5-pro"
        }, "nextNodeId": "node_end"},
        "node_end": {"id": "node_end", "type": "END"}
    }
}
$json$::jsonb, 1),
(NULL, 'bp_risk_demo', 'Settlement Risk Prediction Demo', $json$
{
    "name": "Settlement Risk Prediction Demo",
    "startNodeId": "node_predict",
    "nodes": {
        "node_predict": {"id": "node_predict", "type": "ACTIVITY", "config": {
            "activityName": "ActivityPredictSettlementRisk",
            "counterpartyName": "Unknown Entity"
        }, "nextNodeId": "node_end"},
        "node_end": {"id": "node_end", "type": "END"}
    }
}
$json$::jsonb, 1),
(NULL, 'bp_rwa_issuance', 'Digital Asset Issuance (RWA)', $json$
{
    "name": "Digital Asset Issuance (RWA)",
    "startNodeId": "node_kyc",
    "nodes": {
        "node_kyc": {"id": "node_kyc", "type": "ACTIVITY", "config": {"activityName": "ActivityPerformKYC"}, "nextNodeId": "node_mint"},
        "node_mint": {"id": "node_mint", "type": "ACTIVITY", "config": {"activityName": "ActivityMintToken"}, "nextNodeId": "node_distribute"},
        "node_distribute": {"id": "node_distribute", "type": "ACTIVITY", "config": {"activityName": "ActivityDistributeDividends"}, "nextNodeId": "node_end"},
        "node_end": {"id": "node_end", "type": "END"}
    }
}
$json$::jsonb, 1)
ON CONFLICT DO NOTHING;
