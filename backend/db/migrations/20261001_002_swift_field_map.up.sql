CREATE TABLE IF NOT EXISTS swift_field_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    swift_version TEXT NOT NULL,
    msg_type TEXT NOT NULL,
    field_tag TEXT NOT NULL,
    semantic_field TEXT NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    transform_fn TEXT,
    UNIQUE (tenant_id, swift_version, msg_type, field_tag)
);

CREATE INDEX IF NOT EXISTS swift_field_map_lookup ON swift_field_map (tenant_id, swift_version, msg_type);

ALTER TABLE swift_field_map ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS swift_field_map_isolation ON swift_field_map;

CREATE POLICY swift_field_map_isolation ON swift_field_map
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    );

COMMENT ON TABLE swift_field_map IS 'SWIFT field tag to semantic field mapping. Gold-copy rows provide defaults; per-tenant rows override. Tenant row wins. No code change needed to add/modify mappings.';
