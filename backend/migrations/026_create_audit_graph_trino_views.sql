DO $$
BEGIN
  -- Ensure required columns exist (compatible with modern catalog schema)
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_edge' AND column_name='source_id') THEN
    RAISE NOTICE 'Skipping audit Trino views: catalog_edge.source_id missing';
    RETURN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='node_type_id') THEN
    RAISE NOTICE 'Skipping audit Trino views: catalog_node.node_type_id missing';
    RETURN;
  END IF;

  -- Create unified semantic events view (use robust column names and fallbacks)
  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.semantic_events AS
  SELECT
    n.id AS event_id,
    COALESCE(n.node_type, n.node_type_id::text) AS node_type,
    n.tenant_id,
    CAST(n.properties['timestamp'] AS timestamp) AS event_time,
    CAST(n.properties['start_ts'] AS timestamp) AS start_timestamp,
    CAST(n.properties['end_ts'] AS timestamp) AS end_timestamp,
    n.properties['status'] AS event_status,
    n.properties['severity'] AS severity,
    n.properties['error_message'] AS error_message,
    e_to.entity_id,
    e_to.entity_type,
    COALESCE(n.qualified_path, n.name) AS entity_name,
    n.properties,
    n.qualified_path,
    n.created_at,
    n.updated_at
  FROM catalog_node n
  LEFT JOIN (
      SELECT
        e.source_id AS event_id,
        e.target_id AS entity_id,
        nt.catalog_type_name AS entity_type
      FROM catalog_edge e
      JOIN catalog_node n2 ON n2.id = e.target_id
      JOIN catalog_node_type nt ON nt.id = n2.node_type_id
      WHERE e.edge_type = 'event_of' OR e.edge_type_id IN (
        SELECT id FROM catalog_edge_type WHERE predicate = 'event_of'
      )
  ) e_to ON e_to.event_id = n.id
  WHERE n.node_type = 'audit_event' OR n.node_type_id IN (
    SELECT id FROM catalog_node_type WHERE catalog_type_name IN (
      'audit_event', 'job_run', 'dag_run', 'changeset_event',
      'compliance_event', 'incident', 'semantic_snapshot', 'ai_suggestion'
    )
  );
  $create$;

  -- Additional views (incident_graph, changeset_impact, compliance_context, ai_suggestions, semantic_snapshot_lineage, tenant_summary)
  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.incident_graph AS
  SELECT
    i.event_id AS incident_id,
    i.tenant_id,
    i.event_time,
    r.event_id AS job_run_id,
    r.node_type AS run_type,
    r.event_status AS run_status,
    r.error_message,
    r.properties['jobId'] AS job_id,
    r.properties['dagId'] AS dag_id
  FROM audit.semantic_events i
  JOIN catalog_edge e ON e.source_id = i.event_id AND (e.edge_type = 'causes' OR e.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'causes'))
  JOIN audit.semantic_events r ON r.event_id = e.target_id
  WHERE i.node_type = 'incident' OR i.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'incident')
    AND (r.node_type = 'job_run' OR r.node_type = 'dag_run');
  $create$;

  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.changeset_impact AS
  SELECT
    cs.event_id AS changeset_id,
    cs.tenant_id,
    cs.event_time,
    cs.properties['title'] AS changeset_title,
    cs.properties['description'] AS changeset_description,
    cs.properties['status'] AS changeset_status,
    cs.properties['source'] AS changeset_source,
    impact.entity_id,
    impact.entity_type,
    impact.entity_name
  FROM audit.semantic_events cs
  LEFT JOIN catalog_edge e ON e.source_id = cs.event_id AND (e.edge_type = 'has_impact_on' OR e.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'has_impact_on'))
  LEFT JOIN (
      SELECT
        e2.target_id AS entity_id,
        nt.catalog_type_name AS entity_type,
        COALESCE(n3.qualified_path, n3.name) AS entity_name
      FROM catalog_edge e2
      JOIN catalog_node n3 ON n3.id = e2.target_id
      JOIN catalog_node_type nt ON nt.id = n3.node_type_id
  ) impact ON impact.entity_id = e.target_id
  WHERE cs.node_type = 'changeset_event' OR cs.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'changeset_event');
  $create$;

  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.compliance_context AS
  SELECT
    ce.event_id AS compliance_event_id,
    ce.tenant_id,
    ce.event_time,
    ce.properties['violationType'] AS violation_type,
    ce.properties['severity'] AS severity,
    ce.properties['message'] AS message,
    ct.entity_id AS context_entity_id,
    ct.entity_type AS context_entity_type,
    ct.entity_name AS context_entity_name
  FROM audit.semantic_events ce
  LEFT JOIN catalog_edge ec ON ec.source_id = ce.event_id AND (ec.edge_type = 'has_compliance_context' OR ec.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'has_compliance_context'))
  LEFT JOIN (
      SELECT
        e3.target_id AS entity_id,
        nt2.catalog_type_name AS entity_type,
        COALESCE(n4.qualified_path, n4.name) AS entity_name
      FROM catalog_edge e3
      JOIN catalog_node n4 ON n4.id = e3.target_id
      JOIN catalog_node_type nt2 ON nt2.id = n4.node_type_id
  ) ct ON ct.entity_id = ec.target_id
  WHERE ce.node_type = 'compliance_event' OR ce.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'compliance_event');
  $create$;

  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.ai_suggestions AS
  SELECT
    ais.event_id AS suggestion_id,
    ais.tenant_id,
    ais.event_time,
    ais.properties['narrative'] AS narrative,
    ais.properties['rootCause'] AS root_cause,
    ais.properties['blastRadius'] AS blast_radius,
    ais.properties['recommendedFix'] AS recommended_fix,
    ais.properties['suggestedChangeSetSummary'] AS changeset_summary,
    src.event_id AS source_event_id,
    src.node_type AS source_event_type,
    src.properties['status'] AS source_event_status
  FROM audit.semantic_events ais
  LEFT JOIN catalog_edge ae ON ae.source_id = ais.event_id AND (ae.edge_type = 'has_ai_narrative' OR ae.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'has_ai_narrative'))
  LEFT JOIN audit.semantic_events src ON src.event_id = ae.target_id
  WHERE ais.node_type = 'ai_suggestion' OR ais.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'ai_suggestion');
  $create$;

  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.semantic_snapshot_lineage AS
  SELECT
    ss.event_id AS snapshot_id,
    ss.tenant_id,
    ss.event_time,
    ss.properties AS snapshot_properties,
    st.entity_id AS semantic_term_id,
    st.entity_name AS semantic_term_name,
    cs.event_id AS changeset_id,
    cs.properties['title'] AS changeset_title
  FROM audit.semantic_events ss
  LEFT JOIN (
      SELECT
        e4.target_id AS entity_id,
        COALESCE(n5.qualified_path, n5.name) AS entity_name,
        e4.source_id AS snapshot_id
      FROM catalog_edge e4
      WHERE e4.edge_type = 'version_of' OR e4.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'version_of')
  ) st ON st.snapshot_id = ss.event_id
  LEFT JOIN (
      SELECT
        e5.target_id AS snapshot_id,
        e5.source_id AS event_id,
        ce.properties
      FROM catalog_edge e5
      JOIN audit.semantic_events ce ON ce.event_id = e5.source_id
      WHERE e5.edge_type = 'applied' OR e5.edge_type_id IN (SELECT id FROM catalog_edge_type WHERE predicate = 'applied')
  ) cs ON cs.snapshot_id = ss.event_id
  WHERE ss.node_type = 'semantic_snapshot' OR ss.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_snapshot');
  $create$;

  EXECUTE $create$
  CREATE OR REPLACE VIEW audit.tenant_summary AS
  SELECT
    se.tenant_id,
    COUNT(*) FILTER (WHERE se.node_type = 'audit_event') AS total_audit_events,
    COUNT(*) FILTER (WHERE se.node_type = 'job_run') AS total_job_runs,
    COUNT(*) FILTER (WHERE se.node_type = 'job_run' AND se.event_status = 'FAILED') AS failed_job_runs,
    COUNT(*) FILTER (WHERE se.node_type = 'dag_run') AS total_dag_runs,
    COUNT(*) FILTER (WHERE se.node_type = 'dag_run' AND se.event_status = 'FAILED') AS failed_dag_runs,
    COUNT(*) FILTER (WHERE se.node_type = 'incident') AS total_incidents,
    COUNT(*) FILTER (WHERE se.node_type = 'compliance_event') AS total_compliance_events,
    COUNT(*) FILTER (WHERE se.node_type = 'changeset_event' AND se.properties['status'] = 'APPLIED') AS applied_changesets,
    COUNT(DISTINCT se.event_id) AS distinct_events,
    MAX(se.event_time) AS last_event_time
  FROM audit.semantic_events se
  GROUP BY se.tenant_id;
  $create$;
  IF EXISTS (SELECT 1 FROM information_schema.views WHERE table_schema='audit' AND table_name='incident_graph') THEN
    EXECUTE $create$
    CREATE OR REPLACE VIEW audit.cross_tenant_incidents AS
    SELECT
      i.incident_id,
      COUNT(DISTINCT i.tenant_id) AS affected_tenant_count,
      ARRAY_AGG(DISTINCT i.tenant_id) AS affected_tenants,
      COUNT(*) AS event_count,
      MAX(i.event_time) AS last_event_time,
      MIN(i.event_time) AS first_event_time
    FROM audit.incident_graph i
    GROUP BY i.incident_id
    HAVING COUNT(DISTINCT i.tenant_id) > 1;
    $create$;
  END IF;
END;
$$ LANGUAGE plpgsql;
-- 10. Cross-tenant incident view (for global ops)
-- Shows multi-tenant incidents
CREATE OR REPLACE VIEW audit.cross_tenant_incidents AS
SELECT
  i.incident_id,
  COUNT(DISTINCT i.tenant_id) AS affected_tenant_count,
  ARRAY_AGG(DISTINCT i.tenant_id) AS affected_tenants,
  COUNT(*) AS event_count,
  MAX(i.event_time) AS last_event_time,
  MIN(i.event_time) AS first_event_time
FROM audit.incident_graph i
GROUP BY i.incident_id
HAVING COUNT(DISTINCT i.tenant_id) > 1;
