-- Migration: Add notify_metrics_registry_changed trigger
-- This trigger fires whenever the metrics_registry table is modified
-- and sends a notification to listening applications (e.g., Semantic Sync service)

CREATE OR REPLACE FUNCTION notify_metrics_registry_changed()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('metrics_registry_changed', json_build_object(
    'operation', TG_OP,
    'node_id', COALESCE(NEW.node_id, OLD.node_id),
    'schema_domain', COALESCE(NEW.schema_domain, OLD.schema_domain),
    'timestamp', NOW()
  )::text);
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if it exists
DROP TRIGGER IF EXISTS metrics_registry_notify_trigger ON metrics_registry;

-- Create the trigger
CREATE TRIGGER metrics_registry_notify_trigger
AFTER INSERT OR UPDATE OR DELETE ON metrics_registry
FOR EACH ROW
EXECUTE FUNCTION notify_metrics_registry_changed();
