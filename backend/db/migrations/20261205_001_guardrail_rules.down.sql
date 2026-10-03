-- 20261205_001_guardrail_rules (down)
--
-- Drops the guardrail rules table and, with it, every configured SoD /
-- certified rule it held. Guardrails are configuration, not user data, but this
-- is still destructive: a database restored after this migration has an empty
-- rule set, and the rule engine will enforce nothing until rules are
-- re-authored. The payload in `data` is the only copy -- there is no second
-- source, which is the point of ADR-023.
DROP TABLE IF EXISTS public.guardrail_rules;
