-- 20261217_001_structure_template_marker.up.sql
--
-- ADR-050. The gold copy's template datasource is a property of the gold copy, not a setting.
--
-- structure_template_app marks the ONE datasource a new tenant's structure for that app is compiled
-- from. NULL means "not a template". The partial unique index is the guarantee: at most one marked
-- row per app, in the whole database. "Exactly one" is enforced where the saga reads it: zero or
-- more than one candidate refuses the run and lists the ids found.
--
-- Only the gold-copy tenant's datasources can be marked; the trigger refuses anything else, so a
-- tenant cannot nominate its own datasource as the template whatever role wrote the row.
--
-- Additive: one nullable column, one index, one trigger. Nothing is marked by this migration;
-- marking is a deliberate act by the owner.

ALTER TABLE public.tenant_product_datasource
    ADD COLUMN IF NOT EXISTS structure_template_app text
        CHECK (structure_template_app IS NULL OR structure_template_app ~ '^[a-z][a-z0-9_]{0,31}$');

CREATE UNIQUE INDEX IF NOT EXISTS tenant_product_datasource_structure_template_app
    ON public.tenant_product_datasource (structure_template_app)
    WHERE structure_template_app IS NOT NULL;

CREATE OR REPLACE FUNCTION public.tenant_product_datasource_template_is_gold() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.structure_template_app IS NULL THEN
        RETURN NEW;
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM public.tenant_product tp
          JOIN public.tenant_instance ti ON ti.id = tp.datasource_id
          JOIN public.tenants t ON t.id = ti.tenant_id
         WHERE tp.id = NEW.tenant_product_id AND COALESCE(t.gold_copy, false)) THEN
        RAISE EXCEPTION 'structure_template_app can only be set on a gold-copy tenant datasource'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS tenant_product_datasource_template_is_gold ON public.tenant_product_datasource;
CREATE TRIGGER tenant_product_datasource_template_is_gold
    BEFORE INSERT OR UPDATE OF structure_template_app, tenant_product_id ON public.tenant_product_datasource
    FOR EACH ROW EXECUTE FUNCTION public.tenant_product_datasource_template_is_gold();
