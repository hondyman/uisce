-- 20260907_context_eval_harness.up.sql
-- Context CI/CD Evaluation Harness, Synthetic Test Matrix & Regression Ledger

CREATE SCHEMA IF NOT EXISTS catalog_eval;

-- 1. Master Evaluation Suites
CREATE TABLE IF NOT EXISTS catalog_eval.eval_suites (
    suite_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    suite_key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    target_bo_id UUID,
    total_test_cases INT NOT NULL DEFAULT 500,
    tolerance_epsilon NUMERIC(12, 8) NOT NULL DEFAULT 0.00000100,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Synthetic Test Case Matrix
CREATE TABLE IF NOT EXISTS catalog_eval.eval_test_cases (
    test_case_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES catalog_eval.eval_suites(suite_id) ON DELETE CASCADE,
    prompt_text TEXT NOT NULL,
    expected_ast JSONB NOT NULL,
    parameter_bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    expected_numerical_result NUMERIC(28, 6),
    requires_exact_match BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. PR Evaluation Execution Runs
CREATE TABLE IF NOT EXISTS catalog_eval.eval_execution_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES catalog_eval.eval_suites(suite_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    git_commit_sha VARCHAR(64) NOT NULL,
    git_pr_number INT,
    total_executed INT NOT NULL,
    passed_count INT NOT NULL,
    failed_count INT NOT NULL,
    ambiguity_warnings INT NOT NULL DEFAULT 0,
    status VARCHAR(30) NOT NULL, -- PASSED, REGRESSION_BLOCKED, FAILED
    merkle_eval_passport VARCHAR(64) NOT NULL,
    execution_duration_ms INT NOT NULL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Discrepancy & Regression Itemization
CREATE TABLE IF NOT EXISTS catalog_eval.eval_discrepancies (
    discrepancy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES catalog_eval.eval_execution_runs(run_id) ON DELETE CASCADE,
    test_case_id UUID NOT NULL REFERENCES catalog_eval.eval_test_cases(test_case_id) ON DELETE CASCADE,
    failure_type VARCHAR(50) NOT NULL, -- NUMERICAL_REGRESSION, GRAMMAR_PARSE_FAIL, AMBIGUOUS_JOIN, DIVIDE_BY_ZERO
    baseline_value TEXT,
    candidate_value TEXT,
    variance_delta NUMERIC(28, 6),
    diagnostic_message TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eval_runs_commit 
ON catalog_eval.eval_execution_runs (git_commit_sha, status);
