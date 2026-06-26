-- Migration to add tables for Hyper-Personalized Direct Indexing (Values Schema)

-- 1. Value Themes
CREATE TABLE value_themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Value Signal Sources
CREATE TABLE value_signal_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL, -- e.g., 'VENDOR', 'NEWS', 'INTERNAL'
    reliability_score NUMERIC(5, 4) DEFAULT 1.0, -- 0.0 to 1.0
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Value Signals
CREATE TABLE value_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id TEXT NOT NULL, -- References an external issuer ID (e.g., ticker or internal ID)
    instrument_id TEXT, -- Optional, if specific to an instrument
    theme_id UUID NOT NULL REFERENCES value_themes(id),
    source_id UUID NOT NULL REFERENCES value_signal_sources(id),
    score NUMERIC(5, 2) NOT NULL, -- -100 to +100
    summary TEXT,
    evidence_refs JSONB, -- Array of objects {url, date, type, summary}
    status TEXT NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'UNDER_REVIEW', 'EXPIRED'
    confidence NUMERIC(5, 4), -- 0.0 to 1.0
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_value_signals_issuer ON value_signals(issuer_id);
CREATE INDEX idx_value_signals_theme ON value_signals(theme_id);

-- 4. Strategy Templates
CREATE TABLE strategy_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    base_policy_ids JSONB, -- List of default constraint IDs or policy references
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Client Values Profiles
CREATE TABLE client_values_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id TEXT NOT NULL UNIQUE, -- References the client in the main system
    strategy_template_id UUID REFERENCES strategy_templates(id),
    preferences JSONB, -- General preferences
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Constraints
CREATE TABLE constraints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_values_profile_id UUID REFERENCES client_values_profiles(id),
    strategy_template_id UUID REFERENCES strategy_templates(id), -- Can belong to a template or a specific profile
    name TEXT,
    scope JSONB NOT NULL, -- {benchmarkId, region, sector, issuer, instrumentId}
    operator TEXT NOT NULL, -- 'EXCLUDE', 'UNDERWEIGHT', 'OVERWEIGHT', 'REQUIRE', 'CAP_EXPOSURE'
    condition JSONB NOT NULL, -- Composable condition tree
    severity TEXT NOT NULL DEFAULT 'MEDIUM', -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (client_values_profile_id IS NOT NULL OR strategy_template_id IS NOT NULL)
);

CREATE INDEX idx_constraints_profile ON constraints(client_values_profile_id);
CREATE INDEX idx_constraints_template ON constraints(strategy_template_id);
