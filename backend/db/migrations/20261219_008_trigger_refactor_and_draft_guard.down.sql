-- 20261219_008_trigger_refactor_and_draft_guard.down.sql
--
-- Revert Tier 2 Trigger Refactor & Draft Mutation Guard

-- 1. Drop draft guard trigger
DROP TRIGGER IF EXISTS trg_guard_regulatory_draft_mutation ON compliance.regulatory_draft_rule;
DROP FUNCTION IF EXISTS compliance.guard_regulatory_draft_mutation();

-- 2. Recreate compliance.to_jcs and compliance.compute_rule_content_hash
CREATE OR REPLACE FUNCTION compliance.to_jcs(val jsonb) RETURNS text AS $$
DECLARE
    t text;
    keys text[];
    k text;
    elems text[];
BEGIN
    t := jsonb_typeof(val);
    IF t = 'null' THEN
        RETURN 'null';
    ELSIF t = 'boolean' OR t = 'number' THEN
        RETURN val::text;
    ELSIF t = 'string' THEN
        RETURN to_json(val#>>'{}')::text;
    ELSIF t = 'object' THEN
        SELECT array_agg(to_json(key)::text || ':' || compliance.to_jcs(value) ORDER BY key::bytea)
        INTO elems
        FROM jsonb_each(val);
        RETURN '{' || COALESCE(array_to_string(elems, ','), '') || '}';
    ELSIF t = 'array' THEN
        SELECT array_agg(compliance.to_jcs(elem))
        INTO elems
        FROM jsonb_array_elements(val);
        RETURN '[' || COALESCE(array_to_string(elems, ','), '') || ']';
    ELSE
        RETURN val::text;
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION compliance.compute_rule_content_hash(
    p_ast jsonb,
    p_params jsonb,
    p_citation text
) RETURNS text AS $$
BEGIN
    RETURN encode(sha256(('v1|' || compliance.to_jcs(COALESCE(p_ast, '{}'::jsonb)) || '|' || compliance.to_jcs(COALESCE(p_params, '{}'::jsonb)) || '|' || COALESCE(p_citation, ''))::bytea), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 3. Restore enforce_rule_version_snapshot with hash check
CREATE OR REPLACE FUNCTION compliance.enforce_rule_version_snapshot() RETURNS trigger AS $$
BEGIN
    IF OLD.ast_condition IS DISTINCT FROM NEW.ast_condition
       OR OLD.parameter_thresholds IS DISTINCT FROM NEW.parameter_thresholds
       OR OLD.citation IS DISTINCT FROM NEW.citation
       OR OLD.current_version IS DISTINCT FROM NEW.current_version THEN

        IF NOT EXISTS (
            SELECT 1 FROM compliance.compliance_rule_version
            WHERE rule_id = NEW.id 
              AND version = COALESCE(NEW.current_version, 1)
              AND content_hash = compliance.compute_rule_content_hash(NEW.ast_condition, NEW.parameter_thresholds, NEW.citation)
        ) THEN
            RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule (id=%, version=%) disallowed without inserting matching compliance_rule_version snapshot in the same transaction.', NEW.id, COALESCE(NEW.current_version, 1);
        END IF;
    END IF;

    RETURN NEW;
END $$ LANGUAGE plpgsql;
