-- 20260930_006_bo_fields_compat_view.up.sql
-- Creates a view over bo_fields that provides column names expected by boresolver.GetBODefinition.
-- Maps: field_name→key, field_type→type, display_label→display_name, display_order→sequence,
--       is_readonly→false, is_searchable→false, section_name→'', default_value→'',
--       reference_bo_id→reference_entity, picklist_values→'{}', updated_at→last_modified_at.

CREATE OR REPLACE VIEW public.bo_fields_compat AS
SELECT
    id,
    tenant_id,
    business_object_id,
    subtype_id,
    key,
    name,
    display_name,
    technical_name,
    type,
    is_core,
    is_required,
    is_system,
    description,
    reference_entity,
    sequence,
    created_at,
    created_by,
    last_modified_at,
    last_modified_by,
    core_id,
    -- Aliases for boresolver query compatibility
    key             AS field_name,
    type            AS field_type,
    display_name    AS display_label,
    sequence        AS display_order,
    false           AS is_readonly,
    false           AS is_searchable,
    ''::text        AS section_name,
    ''::text        AS default_value,
    reference_entity AS reference_bo_id,
    '{}'::jsonb     AS picklist_values,
    last_modified_at AS updated_at
FROM public.bo_fields;
