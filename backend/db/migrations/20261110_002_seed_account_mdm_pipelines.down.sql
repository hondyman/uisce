DELETE FROM public.data_pipeline_definitions
WHERE id IN (
    'a11c0001-0001-4000-8000-000000000001'::uuid,
    'a11c0001-0001-4000-8000-000000000002'::uuid
);
