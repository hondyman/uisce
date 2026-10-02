-- 20261205_001_guardrail_rules
--
-- Guardrails are DB-only (ADR-023). That decision made the database the single
-- source of truth for configurable SoD / certified rules, and it exposed a gap
-- that the deleted YAML fallback had been hiding: NOTHING in version control
-- created this table.
--
-- The table was created by application code, in `EnsureOptimizationSchema`
-- (internal/bundles/optimizer.go), via `CREATE TABLE IF NOT EXISTS`. That
-- function has exactly one non-test caller -- `AnalyzeAndPropose`, which calls
-- it on line 86 -- and nothing calls it at boot. So the table only ever came
-- into existence as a side effect of running the optimizer proposal flow. A
-- database provisioned purely from this repo's migrations had no
-- guardrail_rules, and `POST /bundles/guardrails/reload` returned
-- `500 pq: relation "guardrail_rules" does not exist (42P01)`.
--
-- This is the failure mode the evidence rule exists to catch: the unit and
-- integration suites were green locally because the developer's database had
-- the table, while a CI database built from the migrations never did. See
-- ADR-024.
--
-- The schema below is deliberately identical to the one optimizer.go creates,
-- so this migration and the legacy code path agree and neither can drift into
-- producing two different tables. The table lives in `public` because that is
-- where the unqualified `CREATE TABLE` in optimizer.go has always put it; moving
-- it to a namespaced schema would orphan the existing table.
--
-- No RLS is applied here on purpose. Guardrail rules are global policy
-- configuration read by the rule engine, not tenant-owned rows, and the reload
-- path deliberately reads the whole table. Adding tenant scoping here would be a
-- behaviour change disguised as a schema fix, and it is out of scope for this
-- migration.
CREATE TABLE IF NOT EXISTS public.guardrail_rules (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT,
    data JSONB
);

COMMENT ON TABLE public.guardrail_rules IS
    'Configurable SoD / certified guardrail rules. DB-only source (ADR-023); created by migration 20261205_001, not by application code (ADR-024).';
COMMENT ON COLUMN public.guardrail_rules.type IS
    'Guardrail type discriminator, e.g. SoD / certified.';
COMMENT ON COLUMN public.guardrail_rules.data IS
    'Guardrail payload. Read by the rule engine; see ValidateGuardrailData.';
