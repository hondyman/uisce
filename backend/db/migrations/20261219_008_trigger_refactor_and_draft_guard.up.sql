-- 20261219_008_trigger_refactor_and_draft_guard.up.sql
--
-- Tier 2 Trigger Refactor & Draft Modification Guard:
-- 1. Simplify trg_enforce_rule_version_snapshot to an existence check only
-- 2. Drop compliance.to_jcs and compliance.compute_rule_content_hash (Go is sole hash authority)
-- 3. Add trg_guard_regulatory_draft_mutation on compliance.regulatory_draft_rule

-- 1. Existence-only snapshot trigger (Go is the sole hash authority)
CREATE OR REPLACE FUNCTION compliance.enforce_rule_version_snapshot() RETURNS trigger AS $$
BEGIN
    -- If logic, thresholds, or citation changed without incrementing version, reject in-place mutation
    IF (OLD.ast_condition IS DISTINCT FROM NEW.ast_condition
        OR OLD.parameter_thresholds IS DISTINCT FROM NEW.parameter_thresholds
        OR OLD.citation IS DISTINCT FROM NEW.citation)
       AND (COALESCE(NEW.current_version, 1) <= COALESCE(OLD.current_version, 1)) THEN
        RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule logic (id=%, version=%) disallowed without incrementing current_version and inserting a companion snapshot.',
            NEW.id, COALESCE(NEW.current_version, 1);
    END IF;

    -- If current_version changed, verify that the new version snapshot exists
    IF OLD.current_version IS DISTINCT FROM NEW.current_version THEN
        IF NOT EXISTS (
            SELECT 1 FROM compliance.compliance_rule_version
            WHERE rule_id = NEW.id 
              AND version = NEW.current_version
        ) THEN
            RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule current_version (id=%, version=%) disallowed without inserting matching compliance_rule_version snapshot in the same transaction.',
                NEW.id, NEW.current_version;
        END IF;
    END IF;

    RETURN NEW;
END $$ LANGUAGE plpgsql;

-- 2. Drop SQL JCS and hash calculation functions
DROP FUNCTION IF EXISTS compliance.compute_rule_content_hash(jsonb, jsonb, text);
DROP FUNCTION IF EXISTS compliance.to_jcs(jsonb);

-- 3. Draft Modification Guard Trigger
CREATE OR REPLACE FUNCTION compliance.guard_regulatory_draft_mutation()
RETURNS trigger AS $$
BEGIN
    IF (OLD.proposed_ast IS DISTINCT FROM NEW.proposed_ast OR
        OLD.proposed_parameter_thresholds IS DISTINCT FROM NEW.proposed_parameter_thresholds OR
        OLD.proposed_citation IS DISTINCT FROM NEW.proposed_citation OR
        OLD.proposed_content_hash IS DISTINCT FROM NEW.proposed_content_hash) THEN
        
        IF OLD.is_approved THEN
            RAISE EXCEPTION 'Audit Violation: Cannot modify proposed content on an approved regulatory draft (draft_id: %). Reject or un-approve draft first.', OLD.id;
        END IF;

        -- If draft had passed corpus, modifying proposed fields invalidates corpus results
        IF OLD.corpus_results IS DISTINCT FROM '{}'::jsonb THEN
            NEW.corpus_results := '{}'::jsonb;
        END IF;
    END IF;

    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_regulatory_draft_mutation ON compliance.regulatory_draft_rule;
CREATE TRIGGER trg_guard_regulatory_draft_mutation
    BEFORE UPDATE ON compliance.regulatory_draft_rule
    FOR EACH ROW EXECUTE FUNCTION compliance.guard_regulatory_draft_mutation();
