-- Migration template: adapt schema names and industry UUIDs as needed
-- Example usage: replace 'industry_a' and UUID with appropriate values

INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  '11111111-1111-1111-1111-111111111111'::uuid AS industry_id,
  m.metric_type,
  m.metric_time,
  m.value,
  m.tags::jsonb,
  to_jsonb(m) - ARRAY['metric_type','metric_time','value','tags','id','created_at','updated_at'] AS details,
  m.created_at,
  m.updated_at
FROM industry_a.metrics m;

-- If you need to preserve original IDs, add source_schema/source_id columns and insert accordingly.
