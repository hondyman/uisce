-- ============================================================================
-- merge_older_term_duplicates.sql
--
-- Merges semantic and business terms that mean the same thing WITHIN ONE DATASOURCE of the gold-copy tenant
-- (TenantId / TenantIdentifier style twins left over from earlier generations). Run through
-- merge_older_term_duplicates.sh, not directly.
--
-- Same-meaning key: split camelCase and separators, expand each word with the abbreviation dictionary, lower-case,
-- join (mirrors canonicalTermKey in backend/internal/api/term_reuse.go).
--
-- Survivor of a group: the member used by the most business-object fields, then with the most edges, then the
-- oldest. Everything that pointed at the others is moved to it (every foreign key to catalog_node and every
-- catalog_edge, both directions); duplicate and self edges that creates are removed; the emptied terms are deleted.
-- A survivor whose name is not plain PascalCase (spaces, underscores, all caps) takes the PascalCase name of a
-- member it absorbed, when that name is free.
--
-- Business object fields keep their own names (field_name), so validation rules, which reference field names, are
-- unaffected. Only their term reference moves.
--
-- NOT touched, and reported instead:
--   * groups where two members are both fields of the SAME business object (merging would collapse two fields);
--   * groups that span datasources (the term list is per datasource; merging would hide the term in one of them).
--
-- Everything runs in one transaction. :mode is 'plan' (default; does it all, prints, ROLLS BACK) or 'apply'
-- (same, COMMITS). Any failed check or error rolls everything back.
-- ============================================================================
\set ON_ERROR_STOP on
\pset pager off
\if :{?mode}
\else
  \set mode plan
\endif

BEGIN;
SET LOCAL lock_timeout = '20s';
SET LOCAL statement_timeout = '240s';

CREATE TEMP TABLE _scope AS SELECT public.uisce_gold_copy_tenant_id() AS tenant_id;
DO $$ BEGIN IF (SELECT tenant_id FROM _scope) IS NULL THEN RAISE EXCEPTION 'ABORT: no gold-copy tenant found'; END IF; END $$;

CREATE FUNCTION pg_temp.term_key(name text) RETURNS text LANGUAGE sql STABLE AS $f$
  SELECT coalesce(string_agg(
           lower(regexp_replace(coalesce(
             (SELECT a.full_word FROM public.abbreviations a WHERE upper(a.abbreviation) = upper(w.tok)),
             CASE upper(w.tok) WHEN 'ID' THEN 'IDENTIFIER' WHEN 'CD' THEN 'CODE' END,
             w.tok), '\s+', '', 'g')),
           '' ORDER BY w.ord), '')
  FROM unnest(string_to_array(trim(regexp_replace(regexp_replace(name, '([a-z0-9])([A-Z])', '\1 \2', 'g'), '[_./ -]+', ' ', 'g')), ' '))
       WITH ORDINALITY AS w(tok, ord)
  WHERE w.tok <> ''
$f$;

-- before-state, for the checks at the end
CREATE TEMP TABLE _before AS SELECT
  (SELECT count(*) FROM public.business_object_fields) AS bo_fields,
  (SELECT count(DISTINCT e.target_node_id) FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
     JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
    WHERE c.tenant_id = (SELECT tenant_id FROM _scope)) AS columns_mapped,
  (SELECT count(*) FROM public.catalog_edge e WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.source_node_id)
      OR NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.target_node_id)) AS dangling;

-- every term, with its key and how it is used
CREATE TEMP TABLE _terms AS
SELECT n.id, t.catalog_type_name AS kind, n.node_name, coalesce(n.tenant_datasource_id::text, '(none)') AS ds, n.created_at,
       pg_temp.term_key(n.node_name) AS key,
       (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = n.id OR e.target_node_id = n.id) AS edges,
       (SELECT count(*) FROM public.business_object_fields f WHERE f.term_node_id = n.id) AS bo_fields
FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name IN ('semantic_term', 'business_term')
WHERE n.tenant_id = (SELECT tenant_id FROM _scope);

-- groups within one datasource
CREATE TEMP TABLE _groups AS
SELECT kind, ds, key, count(*) AS members FROM _terms GROUP BY 1, 2, 3 HAVING count(*) > 1;

-- groups whose members are fields of the same business object: a merge would give that BO two fields on one term
CREATE TEMP TABLE _conflicts AS
SELECT DISTINCT g.kind, g.ds, g.key
FROM _groups g JOIN _terms a ON a.kind = g.kind AND a.ds = g.ds AND a.key = g.key
JOIN _terms b ON b.kind = g.kind AND b.ds = g.ds AND b.key = g.key AND b.id > a.id
JOIN public.business_object_fields fa ON fa.term_node_id = a.id
JOIN public.business_object_fields fb ON fb.term_node_id = b.id AND fb.bo_id = fa.bo_id;

CREATE TEMP TABLE _plan AS
SELECT t.*, first_value(t.id) OVER w AS survivor_id, first_value(t.node_name) OVER w AS survivor_name, row_number() OVER w AS rn
FROM _terms t JOIN _groups g ON g.kind = t.kind AND g.ds = t.ds AND g.key = t.key
WHERE NOT EXISTS (SELECT 1 FROM _conflicts c WHERE c.kind = t.kind AND c.ds = t.ds AND c.key = t.key)
WINDOW w AS (PARTITION BY t.kind, t.ds, t.key ORDER BY t.bo_fields DESC, t.edges DESC, t.created_at ASC, t.node_name);

CREATE TEMP TABLE _merge AS
SELECT id AS loser_id, survivor_id, kind, node_name AS loser_name, survivor_name, ds FROM _plan WHERE rn > 1;

-- the name each survivor should end up with: keep its own if it is plain PascalCase, else a member's that is
CREATE TEMP TABLE _rename AS
SELECT s.id AS survivor_id, s.kind, s.node_name AS old_name, best.node_name AS new_name
FROM _plan s
JOIN LATERAL (SELECT m.node_name FROM _plan m
              WHERE m.kind = s.kind AND m.ds = s.ds AND m.key = s.key
                AND m.node_name ~ '^[A-Za-z][A-Za-z0-9]*$' AND m.node_name ~ '[a-z]' AND m.node_name !~ '^[A-Z0-9]+$'
              ORDER BY (m.rn = 1) DESC, m.bo_fields DESC, m.edges DESC, m.created_at LIMIT 1) best ON true
WHERE s.rn = 1 AND s.kind = 'semantic_term'
  AND NOT (s.node_name ~ '^[A-Za-z][A-Za-z0-9]*$' AND s.node_name ~ '[a-z]' AND s.node_name !~ '^[A-Z0-9]+$')
  AND best.node_name <> s.node_name;

\echo
\echo '== groups found (within one datasource)'
SELECT g.kind, count(*) AS groups, sum(g.members) AS terms,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM _conflicts c WHERE c.kind = g.kind AND c.ds = g.ds AND c.key = g.key)) AS skipped_conflict
FROM _groups g GROUP BY 1 ORDER BY 1;

\echo
\echo '== NOT merged: both terms are fields of the same business object (a decision for you)'
SELECT o.bo_key, fa.field_name AS field_a, fb.field_name AS field_b, a.node_name AS term_a, b.node_name AS term_b
FROM _conflicts c JOIN _terms a ON a.kind = c.kind AND a.ds = c.ds AND a.key = c.key
JOIN _terms b ON b.kind = c.kind AND b.ds = c.ds AND b.key = c.key AND b.id > a.id
JOIN public.business_object_fields fa ON fa.term_node_id = a.id JOIN public.business_object_fields fb ON fb.term_node_id = b.id AND fb.bo_id = fa.bo_id
JOIN public.business_objects o ON o.id = fa.bo_id ORDER BY 1;

\echo
\echo '== to merge: the twin folded away, and the term that survives it (edges / BO fields)'
SELECT m.kind, m.loser_name AS twin, m.survivor_name AS survivor,
       (SELECT edges FROM _terms WHERE id = m.loser_id) AS twin_edges,
       (SELECT bo_fields FROM _terms WHERE id = m.loser_id) AS twin_bo_fields,
       (SELECT edges FROM _terms WHERE id = m.survivor_id) AS survivor_edges,
       (SELECT bo_fields FROM _terms WHERE id = m.survivor_id) AS survivor_bo_fields
FROM _merge m ORDER BY 1, 3, 2;

\echo
\echo '== survivors that take a well-formed name from a twin they absorb'
SELECT kind, old_name, new_name FROM _rename ORDER BY 1, 2;

CREATE OR REPLACE FUNCTION pg_temp.apply_merges() RETURNS TABLE (loser text, edges_moved bigint, refs_moved bigint) LANGUAGE plpgsql AS $$
DECLARE m record; fk record; n bigint; e1 bigint; refs bigint;
BEGIN
  FOR m IN SELECT * FROM _merge ORDER BY _merge.kind, _merge.loser_name LOOP
    refs := 0;
    FOR fk IN SELECT c.conrelid::regclass AS tbl, a.attname AS col
              FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
              WHERE c.contype = 'f' AND c.confrelid = 'public.catalog_node'::regclass AND array_length(c.conkey, 1) = 1
              ORDER BY 1, 2 LOOP
      EXECUTE format('UPDATE %s SET %I = $1 WHERE %I = $2', fk.tbl, fk.col, fk.col) USING m.survivor_id, m.loser_id;
      GET DIAGNOSTICS n = ROW_COUNT; refs := refs + n;
    END LOOP;
    UPDATE public.catalog_edge SET source_node_id = m.survivor_id WHERE source_node_id = m.loser_id;
    GET DIAGNOSTICS e1 = ROW_COUNT;
    UPDATE public.catalog_edge SET target_node_id = m.survivor_id WHERE target_node_id = m.loser_id;
    GET DIAGNOSTICS n = ROW_COUNT; e1 := e1 + n;
    UPDATE public.catalog_node s SET description = l.description
      FROM public.catalog_node l WHERE s.id = m.survivor_id AND l.id = m.loser_id AND (s.description IS NULL OR s.description = '') AND l.description IS NOT NULL;
    DELETE FROM public.catalog_node WHERE id = m.loser_id;
    loser := m.loser_name; edges_moved := e1; refs_moved := refs;
    RETURN NEXT;
  END LOOP;
END $$;

CREATE TEMP TABLE _applied AS SELECT * FROM pg_temp.apply_merges();

-- names: after the twins are gone their paths are free
UPDATE public.catalog_node n
   SET node_name = r.new_name, qualified_path = r.kind || '/' || r.new_name, updated_at = NOW()
  FROM _rename r
 WHERE n.id = r.survivor_id
   AND NOT EXISTS (SELECT 1 FROM public.catalog_node x WHERE x.tenant_id = n.tenant_id AND x.qualified_path = r.kind || '/' || r.new_name AND x.id <> n.id);

-- tidy what re-pointing can leave behind
WITH dup AS (
  SELECT tableoid, ctid, row_number() OVER (PARTITION BY tenant_datasource_id, source_node_id, target_node_id, edge_type_id ORDER BY created_at, id) AS rn
  FROM public.catalog_edge WHERE source_node_id IN (SELECT DISTINCT survivor_id FROM _merge) OR target_node_id IN (SELECT DISTINCT survivor_id FROM _merge))
DELETE FROM public.catalog_edge e USING dup WHERE e.tableoid = dup.tableoid AND e.ctid = dup.ctid AND dup.rn > 1;
DELETE FROM public.catalog_edge WHERE source_node_id = target_node_id AND source_node_id IN (SELECT DISTINCT survivor_id FROM _merge);

\echo
\echo '== result'
SELECT 'terms merged away' AS what, count(*)::text AS n FROM _applied
UNION ALL SELECT 'edges moved', coalesce(sum(edges_moved), 0)::text FROM _applied
UNION ALL SELECT 'foreign-key rows moved (business object fields and others)', coalesce(sum(refs_moved), 0)::text FROM _applied
UNION ALL SELECT 'survivors renamed', (SELECT count(*) FROM _rename r JOIN public.catalog_node n ON n.id = r.survivor_id AND n.node_name = r.new_name)::text
UNION ALL SELECT 'groups skipped (same business object)', (SELECT count(*) FROM _conflicts)::text
UNION ALL SELECT 'groups left alone (span datasources)', (SELECT count(*) FROM (SELECT kind, key FROM _terms GROUP BY 1, 2 HAVING count(DISTINCT ds) > 1 AND count(*) > 1) x)::text;

-- safety checks: any failure raises and rolls everything back
DO $$
DECLARE bof bigint; cm bigint; dg bigint; left_over bigint; b record;
BEGIN
  SELECT * INTO b FROM _before;
  SELECT count(*) INTO bof FROM public.business_object_fields;
  IF bof <> b.bo_fields THEN RAISE EXCEPTION 'CHECK FAILED: business object fields changed from % to % (they must only be re-pointed)', b.bo_fields, bof; END IF;

  SELECT count(DISTINCT e.target_node_id) INTO cm FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
    JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
   WHERE c.tenant_id = (SELECT tenant_id FROM _scope);
  IF cm < b.columns_mapped THEN RAISE EXCEPTION 'CHECK FAILED: columns mapped to a term dropped from % to %', b.columns_mapped, cm; END IF;

  SELECT count(*) INTO dg FROM public.catalog_edge e WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.source_node_id) OR NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.target_node_id);
  IF dg > b.dangling THEN RAISE EXCEPTION 'CHECK FAILED: % edges point at a missing node (there were % before)', dg, b.dangling; END IF;

  SELECT count(*) INTO left_over FROM (
    SELECT 1 FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name IN ('semantic_term', 'business_term')
     WHERE n.tenant_id = (SELECT tenant_id FROM _scope)
     GROUP BY t.catalog_type_name, coalesce(n.tenant_datasource_id::text, '(none)'), pg_temp.term_key(n.node_name) HAVING count(*) > 1) g;
  IF left_over <> (SELECT count(*) FROM _conflicts) THEN
    RAISE EXCEPTION 'CHECK FAILED: % same-datasource groups remain, but only the % skipped conflict groups should', left_over, (SELECT count(*) FROM _conflicts);
  END IF;
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
