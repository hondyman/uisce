-- Create table for storing user feedback on NLQ responses
CREATE TABLE IF NOT EXISTS nlq_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    query_id UUID NOT NULL, -- ID of the query/response being rated (client-generated or returned by API)
    tenant_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5), -- 1 (bad) to 5 (good)
    comment TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create table for storing golden test cases for evaluation
CREATE TABLE IF NOT EXISTS nlq_eval_cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    category TEXT, -- e.g., "calculation", "lookup", "aggregation"
    question TEXT NOT NULL,
    expected_sql TEXT,
    expected_answer TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create table for storing evaluation run results
CREATE TABLE IF NOT EXISTS nlq_eval_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL, -- Group results by a single execution run
    case_id UUID REFERENCES nlq_eval_cases(id),
    actual_answer TEXT,
    actual_sql TEXT,
    is_correct BOOLEAN,
    similarity_score FLOAT, -- Semantic similarity if applicable
    latency_ms INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_nlq_feedback_query_id ON nlq_feedback(query_id);
CREATE INDEX IF NOT EXISTS idx_nlq_eval_results_run_id ON nlq_eval_results(run_id);
