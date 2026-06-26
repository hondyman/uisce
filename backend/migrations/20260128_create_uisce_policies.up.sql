CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS uisce_policies (
    id SERIAL PRIMARY KEY,
    policy_key VARCHAR(100) NOT NULL, -- e.g., "global_limit_check"
    
    -- The Logic
    cue_definition TEXT NOT NULL,     -- The actual CUE code
    
    -- The Time Slice
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to   TIMESTAMPTZ,           -- NULL means "Forever" (until a new rule caps it)
    
    -- Metadata
    version_tag VARCHAR(50),          -- "v1.0", "v1.1 - Christmas Update"
    created_by VARCHAR(100),
    status VARCHAR(20) DEFAULT 'DRAFT', -- DRAFT, ACTIVE, RETIRED
    
    -- CONSTRAINT: Ensure time ranges do not overlap for the same policy_key
    EXCLUDE USING GIST (
        policy_key WITH =,
        tstzrange(valid_from, valid_to) WITH &&
    )
);
