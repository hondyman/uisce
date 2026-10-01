-- Notifies listeners when catalog nodes change. The payload carries the node
-- type and tenant so embedded engines can filter relevant invalidations.
-- NOTIFY is transactional: listeners fire only after the writer commits.
CREATE OR REPLACE FUNCTION public.uisce_catalog_notify() RETURNS trigger AS $$
DECLARE
    node_type text;
    tenant_id text;
BEGIN
    SELECT catalog_type_name INTO node_type FROM catalog_node_type WHERE id = COALESCE(NEW.node_type_id, OLD.node_type_id);
    tenant_id := COALESCE(NEW.tenant_id::text, OLD.tenant_id::text);
    PERFORM pg_notify('catalog_changed',
        json_build_object(
            'op',  TG_OP,
            'type', node_type,
            'tenant', tenant_id
        )::text
    );
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_catalog_node_notify ON catalog_node;
CREATE TRIGGER trg_catalog_node_notify
AFTER INSERT OR UPDATE OR DELETE ON catalog_node
FOR EACH ROW EXECUTE FUNCTION public.uisce_catalog_notify();
