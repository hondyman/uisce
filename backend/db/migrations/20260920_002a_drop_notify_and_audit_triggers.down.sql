-- Rollback for 20260920_002a_drop_notify_and_audit_triggers.up.sql
-- Re-creates all dropped triggers and functions.
-- Authoritative function bodies saved to: /tmp/trigger_fn_backups.sql

BEGIN;

-- ============================================================================
-- Phase 1a reverse: workflow audit trigger + function
-- tgtype 29 = ROW AFTER INSERT OR UPDATE OR DELETE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.log_workflow_audit()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, new_state)
        VALUES (TG_TABLE_NAME, NEW.opportunity_id, 'CREATE', row_to_json(NEW));
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state, new_state, changed_fields)
        VALUES (
            TG_TABLE_NAME,
            NEW.opportunity_id,
            'UPDATE',
            row_to_json(OLD),
            row_to_json(NEW),
            ARRAY(SELECT key FROM jsonb_each(row_to_json(NEW)::jsonb)
                  WHERE row_to_json(NEW)::jsonb->key != row_to_json(OLD)::jsonb->key)
        );
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state)
        VALUES (TG_TABLE_NAME, OLD.opportunity_id, 'DELETE', row_to_json(OLD));
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$function$;

CREATE TRIGGER audit_investment_opportunities
    AFTER INSERT OR UPDATE OR DELETE ON public.investment_opportunities
    FOR EACH ROW EXECUTE FUNCTION public.log_workflow_audit();

-- ============================================================================
-- Phase 1b reverse: NOTIFY-only functions + triggers
-- tgtype 29 = ROW AFTER INSERT OR UPDATE OR DELETE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.notify_security_change()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    payload TEXT;
BEGIN
    IF (TG_OP = 'DELETE') THEN
        payload := json_build_object(
            'tenant_id', OLD.tenant_id,
            'user_id', OLD.user_id,
            'action', TG_OP,
            'timestamp', now()
        )::text;
    ELSE
        payload := json_build_object(
            'tenant_id', NEW.tenant_id,
            'user_id', NEW.user_id,
            'action', TG_OP,
            'timestamp', now()
        )::text;
    END IF;
    PERFORM pg_notify('security_fund_access_change', payload);
    RETURN NULL;
END;
$function$;

CREATE TRIGGER trg_notify_security_change
    AFTER INSERT OR UPDATE OR DELETE ON public.security_user_fund_access
    FOR EACH ROW EXECUTE FUNCTION public.notify_security_change();

CREATE OR REPLACE FUNCTION public.notify_metrics_registry_changed()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
  PERFORM pg_notify('metrics_registry_changed', json_build_object(
    'operation', TG_OP,
    'node_id', COALESCE(NEW.node_id, OLD.node_id),
    'schema_domain', COALESCE(NEW.schema_domain, OLD.schema_domain),
    'timestamp', NOW()
  )::text);
  RETURN COALESCE(NEW, OLD);
END;
$function$;

CREATE TRIGGER metrics_registry_notify_trigger
    AFTER INSERT OR UPDATE OR DELETE ON public.metrics_registry
    FOR EACH ROW EXECUTE FUNCTION public.notify_metrics_registry_changed();

-- ============================================================================
-- Phase 1c reverse: misnamed "audit" functions + triggers
-- Also remove the replacement touchers installed before the drop.
-- tgtype 19 = ROW BEFORE UPDATE
-- tgtype 23 = ROW BEFORE INSERT OR UPDATE
-- ============================================================================

-- Remove the replacement touchers first
DROP TRIGGER IF EXISTS update_uma_accounts_updated_at
    ON public.uma_accounts;
DROP TRIGGER IF EXISTS update_uma_rebalance_requests_updated_at
    ON public.uma_rebalance_requests;
DROP TRIGGER IF EXISTS update_validation_patterns_updated_at
    ON public.validation_patterns;

-- Restore the old misnamed functions
CREATE OR REPLACE FUNCTION public.audit_uma_accounts()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_audit_uma_accounts
    BEFORE UPDATE ON public.uma_accounts
    FOR EACH ROW EXECUTE FUNCTION public.audit_uma_accounts();

CREATE OR REPLACE FUNCTION public.audit_uma_rebalance_requests()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_audit_uma_rebalance_requests
    BEFORE UPDATE ON public.uma_rebalance_requests
    FOR EACH ROW EXECUTE FUNCTION public.audit_uma_rebalance_requests();

CREATE OR REPLACE FUNCTION public.audit_validation_patterns()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = now();
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_audit_validation_patterns
    BEFORE INSERT OR UPDATE ON public.validation_patterns
    FOR EACH ROW EXECUTE FUNCTION public.audit_validation_patterns();

COMMIT;
