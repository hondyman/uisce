DELETE FROM public.data_pipeline_definitions
WHERE id IN (
    'a11c0001-0001-4000-8000-000000000011'::uuid,
    'a11c0001-0001-4000-8000-000000000012'::uuid
);

DELETE FROM public.staging_bindings sb
USING public.business_objects bo
WHERE sb.bo_id = bo.id
  AND bo.bo_key = 'security'
  AND sb.staging_table = 'staging.security_data';
