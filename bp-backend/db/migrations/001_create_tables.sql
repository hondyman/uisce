CREATE TABLE workflow_definitions (
    id uuid PRIMARY KEY,
    tenant_id uuid,
    name varchar(255),
    active_version_id uuid,
    created_at timestamptz
);

CREATE TABLE workflow_versions (
    id uuid PRIMARY KEY,
    definition_id uuid,
    version_tag varchar(100),
    definition_snapshot jsonb,
    is_published boolean,
    created_at timestamptz
);

CREATE TABLE workflow_nodes (
    id uuid PRIMARY KEY,
    version_id uuid,
    type varchar(50),
    name varchar(255),
    config_json jsonb,
    ui_metadata_json jsonb
);

CREATE TABLE workflow_edges (
    id uuid PRIMARY KEY,
    version_id uuid,
    source_node_id uuid,
    target_node_id uuid,
    condition_expression varchar(1000)
);