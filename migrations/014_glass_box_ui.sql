-- 1. Workflow Runs (Structured Metadata for UI)
CREATE TABLE workflow_runs (
    run_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id TEXT NOT NULL,
    objective TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    status TEXT NOT NULL, -- 'initiated', 'drafting', 'review', 'approved', 'published'
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_runs_client ON workflow_runs (client_id);
CREATE INDEX idx_runs_status ON workflow_runs (status);

-- 2. Policy Hits (Structured Violations)
CREATE TABLE policy_hits (
    hit_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES workflow_runs(run_id),
    artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id),
    rule_id TEXT NOT NULL,
    severity TEXT NOT NULL, -- 'critical', 'high', 'medium', 'low'
    span_offsets JSONB NOT NULL, -- e.g., [{"start": 10, "end": 20}]
    policy_version TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_hits_run ON policy_hits (run_id);
CREATE INDEX idx_hits_artifact ON policy_hits (artifact_id);

-- 3. Decisions (Structured Outcomes)
CREATE TABLE decisions (
    decision_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES workflow_runs(run_id),
    outcome TEXT NOT NULL, -- 'approved', 'rejected', 'revised'
    input_artifact_id UUID REFERENCES artifacts(artifact_id),
    policy_eval_artifact_id UUID REFERENCES artifacts(artifact_id),
    actor_id TEXT,
    comment TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_decisions_run ON decisions (run_id);

-- 4. Replay References (For Deterministic Reconstruction)
CREATE TABLE replay_refs (
    replay_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_run_id UUID NOT NULL REFERENCES workflow_runs(run_id),
    prompt_artifact_id UUID NOT NULL,
    data_snapshot_ids JSONB NOT NULL,
    policy_version TEXT NOT NULL,
    llm_adapter_meta JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
