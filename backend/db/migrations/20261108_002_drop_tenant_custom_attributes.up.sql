-- Retire the broken BO-overlay custom attribute registry.
-- Replaced by public.attribute_def (control plane) + entity custom_attributes JSONB (values).

DROP TABLE IF EXISTS public.tenant_custom_attributes;
