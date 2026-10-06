-- 20261222_001_report_definition_subject_pin
-- PR7 / CUBE-3.3: one-shot rewrite of legacy dataBindings.primary.cube name strings
-- to pinned subject { kind:'cube', cubeId, contractVersion } when a matching
-- data_explorer.cube_definition.name exists. Unresolved names are left as-is
-- for fail-closed runtime migrate (frontend analytical-subject/legacyCubeMigrate.ts).
--
-- Alpha scan (2026-10-05): 7/7 report_definitions had legacy cube names
-- (oms.account, oms.position, altinv.alternative_investment, cash_flow.settlement);
-- 0 had subject pins; 0 page_definitions carried legacy cube name strings.
-- No cube_definition rows were named like those legacy strings at scan time,
-- so this UPDATE is expected to rewrite 0 rows until cubes are published under
-- those names (or seeds are rewritten with explicit subject pins).

UPDATE report_definitions rd
SET
  definition = jsonb_set(
    rd.definition,
    '{dataBindings,primary}',
    (
      (COALESCE(rd.definition->'dataBindings'->'primary', '{}'::jsonb) - 'cube')
      || jsonb_build_object(
        'subject', jsonb_build_object(
          'kind', 'cube',
          'cubeId', c.id::text,
          'contractVersion', c.contract_version
        )
      )
    ),
    true
  ),
  updated_at = now()
FROM data_explorer.cube_definition c
WHERE rd.definition #>> '{dataBindings,primary,cube}' IS NOT NULL
  AND rd.definition #>> '{dataBindings,primary,subject,cubeId}' IS NULL
  AND rd.definition #>> '{dataBindings,primary,cube}' = c.name
  AND c.archived_at IS NULL;
