-- ============================================================================
-- standardize_security_id_term.sql
--
-- One term for "the security's id": SecurityId (uuid, 29 columns). The security MDM business objects were seeded with
-- SecurityIdentifier (uuid, 18 columns) for the same concept. This, in one transaction:
--
--   1. folds the SecurityIdentifier term into SecurityId (every foreign key and catalog edge that pointed at it);
--   2. renames the SecurityIdentifierA / SecurityIdentifierB terms (security_id_a / security_id_b) to
--      SecurityIdA / SecurityIdB, matching the SecurityId spelling and the issuer pair IssuerIdA / IssuerIdB;
--   3. renames the business object fields that carried those names, so a BO's vocabulary matches its term;
--   4. rewrites the stored validation rules on those BOs to the new field names. Rules reference FIELD NAMES, so
--      doing 3 without 4 would turn every one of them into a rule_error.
--
-- Deliberately NOT touched: SecId. It looks like a twin only because the dictionary reads SEC as Security, but
-- orm.position and orm.security carry it as a separate numeric legacy key (position also has the uuid security_id).
--
-- Run through standardize_security_id_term.sh. :mode is 'plan' (default; does it all, prints, ROLLS BACK) or 'apply'.
-- Any failed check or error rolls everything back.
-- ============================================================================
\set ON_ERROR_STOP on
\pset pager off
\if :{?mode}
\else
  \set mode plan
\endif
\if :{?ds}
\else
  \set ds 441f62c9-aad1-481d-9aab-62943fa11cd3
\endif

BEGIN;
SET LOCAL lock_timeout = '20s';
SET LOCAL statement_timeout = '120s';

CREATE TEMP TABLE _scope AS SELECT public.uisce_gold_copy_tenant_id() AS tenant_id, :'ds'::uuid AS ds;
DO $$ BEGIN IF (SELECT tenant_id FROM _scope) IS NULL THEN RAISE EXCEPTION 'ABORT: no gold-copy tenant found'; END IF; END $$;

-- old name -> new name (semantic term). SecurityId already exists, so its twin is folded into it; A and B do not exist,
-- so they are renamed in place.
CREATE TEMP TABLE _map (old_name text PRIMARY KEY, new_name text NOT NULL);
INSERT INTO _map VALUES ('SecurityIdentifier', 'SecurityId'), ('SecurityIdentifierA', 'SecurityIdA'), ('SecurityIdentifierB', 'SecurityIdB');

CREATE TEMP TABLE _st AS
SELECT m.old_name, m.new_name, o.id AS old_id, n.id AS new_id
FROM _map m
LEFT JOIN LATERAL (SELECT x.id FROM public.catalog_node x JOIN public.catalog_node_type t ON t.id = x.node_type_id AND t.catalog_type_name = 'semantic_term'
                    WHERE x.tenant_id = (SELECT tenant_id FROM _scope) AND x.tenant_datasource_id = (SELECT ds FROM _scope) AND x.node_name = m.old_name) o ON true
LEFT JOIN LATERAL (SELECT x.id FROM public.catalog_node x JOIN public.catalog_node_type t ON t.id = x.node_type_id AND t.catalog_type_name = 'semantic_term'
                    WHERE x.tenant_id = (SELECT tenant_id FROM _scope) AND x.tenant_datasource_id = (SELECT ds FROM _scope) AND x.node_name = m.new_name) n ON true;

DO $$
DECLARE r record; dupes bigint;
BEGIN
  FOR r IN SELECT * FROM _st LOOP
    IF r.old_id IS NULL THEN RAISE EXCEPTION 'ABORT: semantic term % not found (already standardized?)', r.old_name; END IF;
  END LOOP;
  SELECT count(*) INTO dupes FROM (SELECT node_name FROM public.catalog_node x JOIN public.catalog_node_type t ON t.id = x.node_type_id AND t.catalog_type_name = 'semantic_term'
     WHERE x.tenant_id = (SELECT tenant_id FROM _scope) AND x.tenant_datasource_id = (SELECT ds FROM _scope) AND x.node_name IN (SELECT old_name FROM _map UNION SELECT new_name FROM _map)
     GROUP BY node_name HAVING count(*) > 1) d;
  IF dupes > 0 THEN RAISE EXCEPTION 'ABORT: % of these term names exist more than once in the datasource', dupes; END IF;
  IF (SELECT new_id FROM _st WHERE old_name = 'SecurityIdentifier') IS NULL THEN RAISE EXCEPTION 'ABORT: the SecurityId term does not exist'; END IF;
END $$;

-- before-state
CREATE TEMP TABLE _before AS SELECT
  (SELECT count(*) FROM public.business_object_fields) AS bo_fields,
  (SELECT count(DISTINCT e.target_node_id) FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
     JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
    WHERE c.tenant_id = (SELECT tenant_id FROM _scope)) AS columns_mapped,
  (SELECT count(*) FROM public.catalog_edge e WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.source_node_id)
      OR NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.target_node_id)) AS dangling,
  (SELECT count(*) FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
    WHERE e.source_node_id IN (SELECT old_id FROM _st WHERE old_name = 'SecurityIdentifier') OR e.source_node_id IN (SELECT new_id FROM _st WHERE old_name = 'SecurityIdentifier')) AS security_id_columns;

\echo
\echo '== 0. what the two terms map (data types must all be uuid)'
SELECT st.node_name AS term, coalesce(c.properties->>'data_type', '?') AS data_type, count(*) AS columns
FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
JOIN public.catalog_node st ON st.id = e.source_node_id JOIN public.catalog_node c ON c.id = e.target_node_id
WHERE st.id IN (SELECT old_id FROM _st UNION SELECT new_id FROM _st WHERE new_id IS NOT NULL) GROUP BY 1, 2 ORDER BY 1, 2;
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
             JOIN public.catalog_node c ON c.id = e.target_node_id
             WHERE e.source_node_id IN (SELECT old_id FROM _st UNION SELECT new_id FROM _st WHERE new_id IS NOT NULL) AND coalesce(c.properties->>'data_type', '') <> 'uuid') THEN
    RAISE EXCEPTION 'ABORT: a column mapped to one of these terms is not a uuid, so it is not the same id';
  END IF;
END $$;

-- the business object fields that carry the old names (captured before anything moves)
CREATE TEMP TABLE _fields AS
SELECT f.id AS field_id, o.bo_key, o.id AS bo_id, f.field_name AS old_field, s.new_name AS new_field, o.tenant_id
FROM public.business_object_fields f JOIN public.business_objects o ON o.id = f.bo_id JOIN _st s ON s.old_id = f.term_node_id AND f.field_name = s.old_name
WHERE o.tenant_id = (SELECT tenant_id FROM _scope);

-- a BO that already has a field with the new name cannot take a second one
DO $$
DECLARE c record;
BEGIN
  FOR c IN SELECT f.bo_key, f.new_field FROM _fields f JOIN public.business_object_fields x ON x.bo_id = f.bo_id AND x.field_name = f.new_field LOOP
    RAISE EXCEPTION 'ABORT: BO % already has a field named %', c.bo_key, c.new_field;
  END LOOP;
  IF EXISTS (SELECT 1 FROM _fields a JOIN _fields b ON a.bo_id = b.bo_id AND a.new_field = b.new_field AND a.field_id <> b.field_id) THEN
    RAISE EXCEPTION 'ABORT: two fields of one BO would end up with the same name';
  END IF;
END $$;

\echo
\echo '== 1. terms: SecurityIdentifier folds into the existing SecurityId; the A and B terms are renamed'
SELECT s.old_name AS term, CASE WHEN s.new_id IS NOT NULL THEN 'fold into ' || s.new_name || ' (' || (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = s.new_id) || ' columns already)' ELSE 'rename to ' || s.new_name END AS action,
       (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = s.old_id) AS columns_moving,
       (SELECT count(*) FROM public.business_object_fields f WHERE f.term_node_id = s.old_id) AS bo_fields
FROM _st s ORDER BY 1;

\echo
\echo '== 2. business object fields renamed'
SELECT bo_key, old_field, new_field FROM _fields ORDER BY 1, 2;

-- the rules on those BOs (referenced by the field names being renamed)
CREATE TEMP TABLE _rules AS
SELECT r.id, r.node_name, r.properties->>'bo_name' AS bo_key
FROM public.catalog_node r JOIN public.catalog_node_type t ON t.id = r.node_type_id AND t.catalog_type_name = 'validation_rule'
WHERE r.tenant_id = (SELECT tenant_id FROM _scope) AND r.properties->>'bo_name' IN (SELECT bo_key FROM _fields)
  AND EXISTS (SELECT 1 FROM _map m WHERE r.config::text LIKE '%"' || m.old_name || '"%');

\echo
\echo '== 3. stored validation rules rewritten to the new field names'
SELECT node_name AS rule, bo_key FROM _rules ORDER BY 1;

-- do it: fields first (captured above), then the rules, then the terms
UPDATE public.business_object_fields f SET field_name = x.new_field, updated_at = NOW() FROM _fields x WHERE f.id = x.field_id;

DO $$
DECLARE m record; rr record; txt text;
BEGIN
  FOR rr IN SELECT id FROM _rules LOOP
    SELECT config::text INTO txt FROM public.catalog_node WHERE id = rr.id;
    FOR m IN SELECT * FROM _map ORDER BY length(old_name) DESC LOOP
      txt := replace(txt, '"' || m.old_name || '"', '"' || m.new_name || '"');   -- exact quoted token: SecurityIdentifierA is not touched by SecurityIdentifier
    END LOOP;
    UPDATE public.catalog_node SET config = txt::jsonb, updated_at = NOW() WHERE id = rr.id;
  END LOOP;
END $$;

-- terms: fold SecurityIdentifier into SecurityId (every FK and edge), rename A and B in place
CREATE TEMP TABLE _merge AS SELECT old_id AS loser_id, new_id AS survivor_id, old_name AS loser_name, new_name AS survivor_name FROM _st WHERE new_id IS NOT NULL;
CREATE OR REPLACE FUNCTION pg_temp.apply_merges() RETURNS TABLE (loser text, edges_moved bigint, refs_moved bigint) LANGUAGE plpgsql AS $$
DECLARE m record; fk record; n bigint; e1 bigint; refs bigint;
BEGIN
  FOR m IN SELECT * FROM _merge LOOP
    refs := 0;
    FOR fk IN SELECT c.conrelid::regclass AS tbl, a.attname AS col
              FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
              WHERE c.contype = 'f' AND c.confrelid = 'public.catalog_node'::regclass AND array_length(c.conkey, 1) = 1 ORDER BY 1, 2 LOOP
      EXECUTE format('UPDATE %s SET %I = $1 WHERE %I = $2', fk.tbl, fk.col, fk.col) USING m.survivor_id, m.loser_id;
      GET DIAGNOSTICS n = ROW_COUNT; refs := refs + n;
    END LOOP;
    UPDATE public.catalog_edge SET source_node_id = m.survivor_id WHERE source_node_id = m.loser_id;
    GET DIAGNOSTICS e1 = ROW_COUNT;
    UPDATE public.catalog_edge SET target_node_id = m.survivor_id WHERE target_node_id = m.loser_id;
    GET DIAGNOSTICS n = ROW_COUNT; e1 := e1 + n;
    UPDATE public.catalog_node s SET description = l.description FROM public.catalog_node l
     WHERE s.id = m.survivor_id AND l.id = m.loser_id AND (s.description IS NULL OR s.description = '') AND l.description IS NOT NULL;
    DELETE FROM public.catalog_node WHERE id = m.loser_id;
    loser := m.loser_name; edges_moved := e1; refs_moved := refs;
    RETURN NEXT;
  END LOOP;
END $$;
CREATE TEMP TABLE _applied AS SELECT * FROM pg_temp.apply_merges();

UPDATE public.catalog_node n SET node_name = s.new_name, qualified_path = 'semantic_term/' || s.new_name, updated_at = NOW()
  FROM _st s WHERE n.id = s.old_id AND s.new_id IS NULL;

-- tidy what re-pointing can leave behind
WITH dup AS (
  SELECT tableoid, ctid, row_number() OVER (PARTITION BY tenant_datasource_id, source_node_id, target_node_id, edge_type_id ORDER BY created_at, id) AS rn
  FROM public.catalog_edge WHERE source_node_id IN (SELECT survivor_id FROM _merge) OR target_node_id IN (SELECT survivor_id FROM _merge))
DELETE FROM public.catalog_edge e USING dup WHERE e.tableoid = dup.tableoid AND e.ctid = dup.ctid AND dup.rn > 1;
DELETE FROM public.catalog_edge WHERE source_node_id = target_node_id AND source_node_id IN (SELECT survivor_id FROM _merge);

\echo
\echo '== result'
SELECT 'edges moved onto SecurityId' AS what, coalesce(sum(edges_moved), 0)::text AS n FROM _applied
UNION ALL SELECT 'business object fields renamed', count(*)::text FROM _fields
UNION ALL SELECT 'stored rules rewritten', count(*)::text FROM _rules
UNION ALL SELECT 'columns now mapped to SecurityId (was ' || (SELECT security_id_columns FROM _before) || ' across both)',
       (SELECT count(*) FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
         WHERE e.source_node_id = (SELECT new_id FROM _st WHERE old_name = 'SecurityIdentifier'))::text;

-- safety checks: any failure raises and rolls everything back
DO $$
DECLARE b record; n bigint; bad record;
BEGIN
  SELECT * INTO b FROM _before;
  IF (SELECT count(*) FROM public.business_object_fields) <> b.bo_fields THEN RAISE EXCEPTION 'CHECK FAILED: the number of business object fields changed'; END IF;

  SELECT count(DISTINCT e.target_node_id) INTO n FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
    JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
   WHERE c.tenant_id = (SELECT tenant_id FROM _scope);
  IF n < b.columns_mapped THEN RAISE EXCEPTION 'CHECK FAILED: columns mapped to a term dropped from % to %', b.columns_mapped, n; END IF;

  SELECT count(*) INTO n FROM public.catalog_edge e WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node x WHERE x.id = e.source_node_id) OR NOT EXISTS (SELECT 1 FROM public.catalog_node x WHERE x.id = e.target_node_id);
  IF n > b.dangling THEN RAISE EXCEPTION 'CHECK FAILED: the change left % edges pointing at a missing node (there were %)', n, b.dangling; END IF;

  SELECT count(*) INTO n FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
   WHERE e.source_node_id = (SELECT new_id FROM _st WHERE old_name = 'SecurityIdentifier');
  IF n <> b.security_id_columns THEN RAISE EXCEPTION 'CHECK FAILED: SecurityId maps % columns, expected %', n, b.security_id_columns; END IF;

  IF EXISTS (SELECT 1 FROM public.business_object_fields f JOIN public.business_objects o ON o.id = f.bo_id WHERE o.tenant_id = (SELECT tenant_id FROM _scope) AND f.field_name IN (SELECT old_name FROM _map)) THEN
    RAISE EXCEPTION 'CHECK FAILED: a business object field still carries an old name';
  END IF;
  IF EXISTS (SELECT 1 FROM public.catalog_node x JOIN public.catalog_node_type t ON t.id = x.node_type_id AND t.catalog_type_name = 'semantic_term'
              WHERE x.tenant_id = (SELECT tenant_id FROM _scope) AND x.tenant_datasource_id = (SELECT ds FROM _scope) AND x.node_name IN (SELECT old_name FROM _map)) THEN
    RAISE EXCEPTION 'CHECK FAILED: an old semantic term is still there';
  END IF;

  -- every field name a rule on these BOs references must exist on that BO, or the rule becomes a rule_error at write time
  FOR bad IN
    SELECT r.node_name, ref.name FROM public.catalog_node r
    JOIN LATERAL (SELECT DISTINCT v #>> '{}' AS name FROM jsonb_path_query(r.config->'rule_ast', 'lax $.**.field') v WHERE jsonb_typeof(v) = 'string' AND v #>> '{}' <> ''
                  UNION SELECT DISTINCT v #>> '{}' FROM jsonb_path_query(r.config->'rule_ast', 'lax $.**.path') v WHERE jsonb_typeof(v) = 'string' AND v #>> '{}' <> '') ref ON true
    WHERE r.id IN (SELECT id FROM _rules)
      AND NOT EXISTS (SELECT 1 FROM public.business_object_fields f JOIN public.business_objects o ON o.id = f.bo_id
                       WHERE o.tenant_id = r.tenant_id AND o.bo_key = r.properties->>'bo_name' AND f.field_name = ref.name)
  LOOP
    RAISE EXCEPTION 'CHECK FAILED: rule % references field %, which its BO does not have', bad.node_name, bad.name;
  END LOOP;
  RAISE NOTICE 'all checks passed';
END $$;

\echo
SELECT (:'mode' = 'apply') AS will_commit \gset
\if :will_commit
  \echo '== APPLY: committing'
  COMMIT;
\else
  \echo '== PLAN: nothing was changed, rolling back'
  ROLLBACK;
\endif
