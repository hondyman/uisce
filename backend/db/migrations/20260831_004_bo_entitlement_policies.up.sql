-- Business-Object-level entitlement policies: role gate, row filter, and
-- field masking. Keyed by BO name (not by API endpoint) so the same policy
-- is enforced no matter how the BO is reached — API Studio REST/GraphQL
-- today, and any future access path (pages, reports, JDBC) that routes
-- through backend/internal/analytics.BOContextResolver + cbo.Planner.
--
-- Consumed by cbo.DBEntitlementRepository.GetPoliciesForBO and enforced in
-- cbo.Planner.Plan (see PLAN_STUDIO_EVENTS_AUDIT.md, API Studio entitlements).
CREATE TABLE IF NOT EXISTS semantic.bo_entitlement_policies (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env               TEXT NOT NULL DEFAULT 'default',
    tenant_id         UUID,
    bo_name           TEXT NOT NULL,
    name              TEXT NOT NULL,
    strategy          TEXT NOT NULL DEFAULT 'inline', -- join | prefilter | inline (cost-model hint only)
    -- Role gate: caller must hold at least one of these JWT roles to access
    -- the BO at all. Empty/null = no role restriction.
    required_roles    TEXT[] NOT NULL DEFAULT '{}',
    -- Row filter: when both set, a "<row_filter_column> = <claim value>"
    -- predicate is injected into every query against this BO. row_filter_claim
    -- must be one of a fixed, safe set of caller claims (see
    -- cbo.allowedRowFilterClaims) — never an arbitrary column/expression.
    row_filter_column TEXT,
    row_filter_claim  TEXT CHECK (row_filter_claim IS NULL OR row_filter_claim IN ('tenant_id', 'organization_id', 'user_id')),
    -- Field masking: JSON object of {"field_name": ["role_a", "role_b"]}.
    -- A field listed here is stripped from every response row unless the
    -- caller holds at least one of the listed roles.
    masked_fields     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by        TEXT NOT NULL DEFAULT 'system',
    UNIQUE (env, tenant_id, bo_name, name)
);

CREATE INDEX IF NOT EXISTS idx_bo_entitlement_policies_lookup
    ON semantic.bo_entitlement_policies (env, tenant_id, bo_name);
