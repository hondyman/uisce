-- ============================================================================
-- merge_semantic_term_twins.sql
--
-- Cleans up the semantic and business terms of the CRIMS datasource after the generator created twins and
-- wrong names (see docs: the generator now reuses an existing term with the same meaning; this fixes what it
-- already made). Run through merge_semantic_term_twins.sh, not directly.
--
--   1. dictionary   adds the abbreviations the generator had to ask an LLM about (and got wrong)
--   2. renames      fixes wrong or ugly names (SecuritiesTypeCode -> SecurityTypeCode, ...AT -> ...At, ...)
--                   - if a term with the right name already exists, the wrong one is MERGED into it
--   3. twins        merges terms that mean the same thing (TenantId / TenantIdentifier, IssuerId / IssuerIdentifier)
--
-- Only terms created by the generator run are ever merged away or renamed (created at or after :since), and they
-- are merged INTO the older existing term. Older terms are never removed here: duplicates among them are listed at
-- the end for a decision, because business objects may use both spellings (a merge would then give one business
-- object two fields on the same term).
--
-- A merge keeps the older existing term (else the one with the most edges), moves EVERYTHING that pointed at the other one
-- (every foreign key to catalog_node, and every catalog_edge, both directions), removes duplicate and self edges
-- it created, and deletes the emptied term. Business object fields keep their own names, so rules written
-- against field names are unaffected.
--
-- Everything runs in one transaction. :mode is 'plan' (default) or 'apply': plan does all of it, prints what
-- happened and ROLLS BACK; apply does the same and COMMITS. Any error aborts and rolls back everything.
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
\if :{?since}
\else
  \set since '2026-09-21 22:20:00+00'
\endif

BEGIN;
SET LOCAL lock_timeout = '20s';
SET LOCAL statement_timeout = '180s';

CREATE TEMP TABLE _scope AS SELECT public.uisce_gold_copy_tenant_id() AS tenant_id, :'ds'::uuid AS ds, :'since'::timestamptz AS since;
DO $$ BEGIN IF (SELECT tenant_id FROM _scope) IS NULL THEN RAISE EXCEPTION 'ABORT: no gold-copy tenant found'; END IF; END $$;

-- Same meaning => same key: split camelCase and separators, expand each word with the dictionary (id and cd
-- also without it), lower-case, join. This mirrors canonicalTermKey in backend/internal/api/term_reuse.go.
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

CREATE TEMP TABLE _counts_before AS
SELECT t.catalog_type_name AS kind, count(*) AS terms
FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name IN ('semantic_term', 'business_term')
WHERE n.tenant_id = (SELECT tenant_id FROM _scope) AND n.tenant_datasource_id = (SELECT ds FROM _scope) GROUP BY 1;
-- some edges already point at nodes that no longer exist (left by older runs); the check below only fails if this
-- cleanup adds to them
CREATE TEMP TABLE _dangling_before AS
SELECT count(*) AS n FROM public.catalog_edge e
WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.source_node_id) OR NOT EXISTS (SELECT 1 FROM public.catalog_node n WHERE n.id = e.target_node_id);
CREATE TEMP TABLE _mapped_before AS
SELECT count(DISTINCT e.target_node_id) AS columns_mapped
FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
WHERE c.tenant_id = (SELECT tenant_id FROM _scope) AND c.tenant_datasource_id = (SELECT ds FROM _scope);

-- 1. dictionary ----------------------------------------------------------------------------------------------
\echo
\echo '== 1. dictionary: abbreviations added (existing ones are left alone)'
WITH ins AS (
  INSERT INTO public.abbreviations (abbreviation, full_word, notes) VALUES
    ('SEC', 'Security', 'curated: an LLM guessed Securities'),
    ('SUB', 'Sub', 'curated: an LLM guessed Subordinated'),
    ('TYP', 'Type', 'curated'),
    ('SLA', 'Sla', 'curated: an LLM guessed Service_level_agreement'),
    ('URL', 'Url', 'curated: an LLM guessed Uniform_resource_locator'),
    ('AUTO', 'Auto', 'curated: matches ThresholdAutoMatch'),
    ('MAX', 'Max', 'curated: matches MaxStalenessHours'),
    ('MIN', 'Min', 'curated: matches MinConfidence')
  ON CONFLICT (upper(abbreviation)) DO NOTHING RETURNING abbreviation, full_word)
SELECT * FROM ins ORDER BY 1;

-- 2. explicit renames ----------------------------------------------------------------------------------------
CREATE TEMP TABLE _fix (kind text, old_name text, new_name text);
INSERT INTO _fix VALUES
  ('semantic_term', 'SecuritiesTypeCode',                    'SecurityTypeCode'),
  ('semantic_term', 'InternalSecuritiesTypeCode',            'InternalSecurityTypeCode'),
  ('semantic_term', 'SecuritiesSubordinatedTypeCode',        'SecuritySubTypeCode'),
  ('semantic_term', 'InternalSecuritiesSubordinatedTypeCode','InternalSecuritySubTypeCode'),
  ('semantic_term', 'VendorSubordinatedTypeCode',            'VendorSubTypeCode'),
  ('semantic_term', 'Service_level_agreementMet',            'SlaMet'),
  ('semantic_term', 'Service_level_agreementMinutes',        'SlaMinutes'),
  ('semantic_term', 'DocumentUniform_resource_locator',      'DocumentUrl'),
  ('semantic_term', 'ThresholdAutomaticMatch',               'ThresholdAutoMatch'),
  ('semantic_term', 'ThresholdNumberMatch',                  'ThresholdNoMatch'),
  ('semantic_term', 'MaximumStalenessHours',                 'MaxStalenessHours'),
  ('semantic_term', 'MinimumConfidence',                     'MinConfidence'),
  ('semantic_term', 'AckAT',                                 'AckAt'),
  ('semantic_term', 'CompletedAT',                           'CompletedAt'),
  ('semantic_term', 'DeliveredAT',                           'DeliveredAt'),
  ('semantic_term', 'ExpectedAT',                            'ExpectedAt'),
  ('semantic_term', 'ExtractedAT',                           'ExtractedAt'),
  ('semantic_term', 'FirstSeenAT',                           'FirstSeenAt'),
  ('semantic_term', 'LastConfirmedAT',                       'LastConfirmedAt'),
  ('semantic_term', 'StartedAT',                             'StartedAt'),
  ('semantic_term', 'ValidatedAT',                           'ValidatedAt'),
  ('business_term', 'Securities Type Code',                  'Security Type Code'),
  ('business_term', 'Internal Securities Type Code',         'Internal Security Type Code'),
  ('business_term', 'Securities Subordinated Type Code',     'Security Sub Type Code'),
  ('business_term', 'Internal Securities Subordinated Type Code', 'Internal Security Sub Type Code'),
  ('business_term', 'Vendor Subordinated Type Code',         'Vendor Sub Type Code'),
  ('business_term', 'Service_level_agreement Met',           'Sla Met'),
  ('business_term', 'Service_level_agreement Minutes',       'Sla Minutes'),
  ('business_term', 'Document Uniform_resource_locator',     'Document Url'),
  ('business_term', 'Threshold Automatic Match',             'Threshold Auto Match'),
  ('business_term', 'Threshold Number Match',                'Threshold No Match'),
  ('business_term', 'Maximum Staleness Hours',               'Max Staleness Hours'),
  ('business_term', 'Minimum Confidence',                    'Min Confidence');

CREATE TEMP TABLE _merge (loser_id uuid PRIMARY KEY, survivor_id uuid NOT NULL, kind text, loser_name text, survivor_name text, why text);

-- a wrong name whose right name already exists is merged into it; otherwise it is renamed in place
INSERT INTO _merge
SELECT o.id, tgt.id, f.kind, o.node_name, tgt.node_name, 'rename target already exists'
FROM _fix f
JOIN public.catalog_node_type ft ON ft.catalog_type_name = f.kind
JOIN public.catalog_node o ON o.node_type_id = ft.id AND o.node_name = f.old_name
     AND o.tenant_id = (SELECT tenant_id FROM _scope) AND o.tenant_datasource_id = (SELECT ds FROM _scope)
     AND o.created_at >= (SELECT since FROM _scope)
JOIN public.catalog_node tgt ON tgt.node_type_id = ft.id AND tgt.node_name = f.new_name
     AND tgt.tenant_id = o.tenant_id AND tgt.tenant_datasource_id = o.tenant_datasource_id
ON CONFLICT (loser_id) DO NOTHING;

\echo
\echo '== 2a. renames in place'
WITH r AS (
  UPDATE public.catalog_node o
     SET node_name = f.new_name, qualified_path = f.kind || '/' || f.new_name, updated_at = NOW()
    FROM _fix f JOIN public.catalog_node_type ft ON ft.catalog_type_name = f.kind
   WHERE o.node_type_id = ft.id AND o.node_name = f.old_name
     AND o.tenant_id = (SELECT tenant_id FROM _scope) AND o.tenant_datasource_id = (SELECT ds FROM _scope)
     AND o.created_at >= (SELECT since FROM _scope)
     AND NOT EXISTS (SELECT 1 FROM _merge m WHERE m.loser_id = o.id)
  RETURNING f.kind, f.old_name, f.new_name)
SELECT * FROM r ORDER BY 1, 2;

-- 3. the merge step, used twice: first the explicit merges above, then the twins found by meaning ---------------
CREATE OR REPLACE FUNCTION pg_temp.apply_merges() RETURNS TABLE (kind text, loser text, survivor text, why text, edges_moved bigint, refs_moved bigint) LANGUAGE plpgsql AS $$
DECLARE m record; fk record; n bigint; e1 bigint; refs bigint;
BEGIN
  FOR m IN SELECT * FROM _merge ORDER BY _merge.kind, _merge.loser_name LOOP
    refs := 0;
    -- every foreign key that points at catalog_node(id): move the loser's rows to the survivor
    FOR fk IN SELECT c.conrelid::regclass AS tbl, a.attname AS col
              FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
              WHERE c.contype = 'f' AND c.confrelid = 'public.catalog_node'::regclass AND array_length(c.conkey, 1) = 1
              ORDER BY 1, 2 LOOP
      EXECUTE format('UPDATE %s SET %I = $1 WHERE %I = $2', fk.tbl, fk.col, fk.col) USING m.survivor_id, m.loser_id;
      GET DIAGNOSTICS n = ROW_COUNT; refs := refs + n;
    END LOOP;
    -- catalog_edge has no foreign key; both directions
    UPDATE public.catalog_edge SET source_node_id = m.survivor_id WHERE source_node_id = m.loser_id;
    GET DIAGNOSTICS e1 = ROW_COUNT;
    UPDATE public.catalog_edge SET target_node_id = m.survivor_id WHERE target_node_id = m.loser_id;
    GET DIAGNOSTICS n = ROW_COUNT; e1 := e1 + n;
    -- keep the loser's definition if the survivor has none
    UPDATE public.catalog_node s SET description = l.description
      FROM public.catalog_node l WHERE s.id = m.survivor_id AND l.id = m.loser_id AND (s.description IS NULL OR s.description = '') AND l.description IS NOT NULL;
    DELETE FROM public.catalog_node WHERE id = m.loser_id;
    kind := m.kind; loser := m.loser_name; survivor := m.survivor_name; why := m.why; edges_moved := e1; refs_moved := refs;
    RETURN NEXT;
  END LOOP;
END $$;

CREATE TEMP TABLE _applied (kind text, loser text, survivor text, why text, edges_moved bigint, refs_moved bigint);
CREATE TEMP TABLE _survivors (id uuid PRIMARY KEY);

\echo
\echo '== 2b. wrong names merged into the term that already has the right name'
SELECT m.kind, m.loser_name AS wrong_name, m.survivor_name AS merged_into,
       (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = m.loser_id OR e.target_node_id = m.loser_id) AS edges
FROM _merge m ORDER BY 1, 2;
INSERT INTO _survivors SELECT DISTINCT survivor_id FROM _merge ON CONFLICT DO NOTHING;
INSERT INTO _applied SELECT * FROM pg_temp.apply_merges();
TRUNCATE _merge;

-- twins by meaning (after the renames, so renamed terms take part)
INSERT INTO _merge
SELECT loser_id, survivor_id, kind, loser_name, survivor_name, 'same meaning'
FROM (
  SELECT n.id AS loser_id, t.catalog_type_name AS kind, n.node_name AS loser_name, n.created_at AS loser_created,
         first_value(n.id) OVER w AS survivor_id, first_value(n.node_name) OVER w AS survivor_name,
         row_number() OVER w AS rn
  FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name IN ('semantic_term', 'business_term')
  WHERE n.tenant_id = (SELECT tenant_id FROM _scope) AND n.tenant_datasource_id = (SELECT ds FROM _scope)
  -- the survivor is an older term when there is one (then the one with the most edges, then the oldest)
  WINDOW w AS (PARTITION BY t.catalog_type_name, pg_temp.term_key(n.node_name)
               ORDER BY (n.created_at >= (SELECT since FROM _scope)),
                        (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = n.id OR e.target_node_id = n.id) DESC, n.created_at ASC, n.node_name)
) x WHERE rn > 1 AND loser_created >= (SELECT since FROM _scope);

\echo
\echo '== 3. twins from the generator run merged into the existing term with the same meaning'
SELECT m.kind, m.loser_name AS twin, m.survivor_name AS survivor,
       (SELECT count(*) FROM public.catalog_edge e WHERE e.source_node_id = m.loser_id OR e.target_node_id = m.loser_id) AS edges
FROM _merge m ORDER BY 1, 3, 2;
INSERT INTO _survivors SELECT DISTINCT survivor_id FROM _merge ON CONFLICT DO NOTHING;
INSERT INTO _applied SELECT * FROM pg_temp.apply_merges();

-- tidy what re-pointing can leave behind: a survivor linked to the same thing twice, or to itself ---------------
WITH dup AS (
  SELECT tableoid, ctid, row_number() OVER (PARTITION BY tenant_datasource_id, source_node_id, target_node_id, edge_type_id ORDER BY created_at, id) AS rn
  FROM public.catalog_edge WHERE source_node_id IN (SELECT id FROM _survivors) OR target_node_id IN (SELECT id FROM _survivors))
DELETE FROM public.catalog_edge e USING dup WHERE e.tableoid = dup.tableoid AND e.ctid = dup.ctid AND dup.rn > 1;
DELETE FROM public.catalog_edge WHERE source_node_id = target_node_id AND source_node_id IN (SELECT id FROM _survivors);

-- result --------------------------------------------------------------------------------------------------------
\echo
\echo '== result'
SELECT 'terms merged away' AS what, count(*)::text AS n FROM _applied
UNION ALL SELECT 'edges moved', coalesce(sum(edges_moved), 0)::text FROM _applied
UNION ALL SELECT 'other foreign-key rows moved', coalesce(sum(refs_moved), 0)::text FROM _applied
UNION ALL SELECT 'edges already pointing at a missing node (not touched)', (SELECT n FROM _dangling_before)::text;
SELECT b.kind, b.terms AS before,
       (SELECT count(*) FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name = b.kind
         WHERE n.tenant_id = (SELECT tenant_id FROM _scope) AND n.tenant_datasource_id = (SELECT ds FROM _scope)) AS after
FROM _counts_before b ORDER BY 1;
SELECT 'columns mapped to a term: before ' || (SELECT columns_mapped FROM _mapped_before) || ', after ' ||
  count(DISTINCT e.target_node_id) AS columns_still_mapped
FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
WHERE c.tenant_id = (SELECT tenant_id FROM _scope) AND c.tenant_datasource_id = (SELECT ds FROM _scope);

\echo
\echo '== not touched: duplicates among OLDER terms (a decision for you; business objects may use both spellings)'
SELECT t.catalog_type_name AS kind, count(*) AS groups
FROM (SELECT n.node_type_id, pg_temp.term_key(n.node_name) AS k, count(*) AS c
        FROM public.catalog_node n JOIN public.catalog_node_type nt ON nt.id = n.node_type_id AND nt.catalog_type_name IN ('semantic_term', 'business_term')
       WHERE n.tenant_id = (SELECT tenant_id FROM _scope) AND n.tenant_datasource_id = (SELECT ds FROM _scope)
       GROUP BY 1, 2 HAVING count(*) > 1) g
JOIN public.catalog_node_type t ON t.id = g.node_type_id GROUP BY 1 ORDER BY 1;

-- safety checks: any failure raises and rolls everything back
DO $$
DECLARE remaining bigint; dangling bigint; before_m bigint; after_m bigint;
BEGIN
  SELECT count(*) INTO remaining FROM (
    SELECT 1 FROM public.catalog_node n JOIN public.catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name IN ('semantic_term', 'business_term')
    WHERE n.tenant_id = (SELECT tenant_id FROM _scope) AND n.tenant_datasource_id = (SELECT ds FROM _scope)
    GROUP BY t.catalog_type_name, pg_temp.term_key(n.node_name)
    HAVING count(*) > 1 AND bool_or(n.created_at >= (SELECT since FROM _scope))) g;
  IF remaining > 0 THEN RAISE EXCEPTION 'CHECK FAILED: % groups of same-meaning terms still include one from the generator run', remaining; END IF;

  SELECT count(*) INTO dangling FROM public.catalog_edge e
   WHERE NOT EXISTS (SELECT 1 FROM public.catalog_node s WHERE s.id = e.source_node_id) OR NOT EXISTS (SELECT 1 FROM public.catalog_node t WHERE t.id = e.target_node_id);
  IF dangling > (SELECT n FROM _dangling_before) THEN
    RAISE EXCEPTION 'CHECK FAILED: the cleanup left % edges pointing at a missing node (there were % before)', dangling, (SELECT n FROM _dangling_before);
  END IF;

  SELECT columns_mapped INTO before_m FROM _mapped_before;
  SELECT count(DISTINCT e.target_node_id) INTO after_m
    FROM public.catalog_edge e JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
    JOIN public.catalog_node c ON c.id = e.target_node_id JOIN public.catalog_node_type ct ON ct.id = c.node_type_id AND ct.catalog_type_name = 'column'
   WHERE c.tenant_id = (SELECT tenant_id FROM _scope) AND c.tenant_datasource_id = (SELECT ds FROM _scope);
  IF after_m < before_m THEN RAISE EXCEPTION 'CHECK FAILED: columns mapped to a term dropped from % to %', before_m, after_m; END IF;
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
