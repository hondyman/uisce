-- Migration: Create semantic_types lookup table
-- Description: Adds a lookup table for semantic types with their data types and formats

-- Ensure the semantic_types lookup exists
INSERT INTO public.lookups (tenant_id, name, description)
SELECT t.id, 'semantic_types', 'Semantic types with data types and formats for nodes and edges' 
FROM tenants t 
ON CONFLICT DO NOTHING;

-- Populate semantic_types lookup with data
DO $$
DECLARE
  lkup_id uuid;
  tenant_uuid uuid;
BEGIN
  SELECT id INTO lkup_id FROM public.lookups WHERE name = 'semantic_types' LIMIT 1;
  IF lkup_id IS NULL THEN
    RETURN;
  END IF;
  SELECT tenant_id INTO tenant_uuid FROM public.lookups WHERE id = lkup_id LIMIT 1;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'lookup_id') THEN
    -- Dimension + string + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_string_default', 'Dimension (string, default)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'default', 'notes', ''))
    ON CONFLICT DO NOTHING;

    -- Dimension + string + imageUrl
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_string_imageurl', 'Dimension (string, imageUrl)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'imageUrl', 'notes', 'Dimension Format'))
    ON CONFLICT DO NOTHING;

    -- Dimension + string + link
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_string_link', 'Dimension (string, link)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'link', 'notes', 'Dimension Format'))
    ON CONFLICT DO NOTHING;

    -- Dimension + string + currency
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_string_currency', 'Dimension (string, currency)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'currency', 'notes', 'Dimension Format (If underlying type is number and formatted as string in SQL)'))
    ON CONFLICT DO NOTHING;

    -- Dimension + string + percent
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_string_percent', 'Dimension (string, percent)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'percent', 'notes', 'Dimension Format (If underlying type is number and formatted as string in SQL)'))
    ON CONFLICT DO NOTHING;

    -- Dimension + number + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_number_default', 'Dimension (number, default)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'number', 'format', 'default', 'notes', ''))
    ON CONFLICT DO NOTHING;

    -- Dimension + number + id
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_number_id', 'Dimension (number, id)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'number', 'format', 'id', 'notes', 'Dimension Format'))
    ON CONFLICT DO NOTHING;

    -- Dimension + number + currency
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_number_currency', 'Dimension (number, currency)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'number', 'format', 'currency', 'notes', 'Dimension Format'))
    ON CONFLICT DO NOTHING;

    -- Dimension + number + percent
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_number_percent', 'Dimension (number, percent)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'number', 'format', 'percent', 'notes', 'Dimension Format'))
    ON CONFLICT DO NOTHING;

    -- Dimension + boolean + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_boolean_default', 'Dimension (boolean, default)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'boolean', 'format', 'default', 'notes', ''))
    ON CONFLICT DO NOTHING;

    -- Dimension + time + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_time_default', 'Dimension (time, default)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'time', 'format', 'default', 'notes', ''))
    ON CONFLICT DO NOTHING;

    -- Dimension + geo + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'dimension_geo_default', 'Dimension (geo, default)', 
            jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'geo', 'format', 'default', 'notes', ''))
    ON CONFLICT DO NOTHING;

    -- Measure + string + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_string_default', 'Measure (string, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'string', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + time + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_time_default', 'Measure (time, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'time', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + boolean + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_boolean_default', 'Measure (boolean, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'boolean', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + number + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_default', 'Measure (number, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + number + percent
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_percent', 'Measure (number, percent)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number', 'format', 'percent', 'notes', 'Measure Format'))
    ON CONFLICT DO NOTHING;

    -- Measure + number + currency
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_currency', 'Measure (number, currency)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number', 'format', 'currency', 'notes', 'Measure Format'))
    ON CONFLICT DO NOTHING;

    -- Measure + number_agg + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_agg_default', 'Measure (number_agg, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number_agg', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + number_agg + percent
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_agg_percent', 'Measure (number_agg, percent)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number_agg', 'format', 'percent', 'notes', 'Measure Format'))
    ON CONFLICT DO NOTHING;

    -- Measure + number_agg + currency
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_number_agg_currency', 'Measure (number_agg, currency)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'number_agg', 'format', 'currency', 'notes', 'Measure Format'))
    ON CONFLICT DO NOTHING;

    -- Measure + count + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_count_default', 'Measure (count, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'count', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + count_distinct + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_count_distinct_default', 'Measure (count_distinct, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'count_distinct', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + count_distinct_approx + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_count_distinct_approx_default', 'Measure (count_distinct_approx, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'count_distinct_approx', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + sum + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_sum_default', 'Measure (sum, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'sum', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + sum + currency
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_sum_currency', 'Measure (sum, currency)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'sum', 'format', 'currency', 'notes', 'Measure Format'))
    ON CONFLICT DO NOTHING;

    -- Measure + avg + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_avg_default', 'Measure (avg, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'avg', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + min + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_min_default', 'Measure (min, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'min', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Measure + max + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'measure_max_default', 'Measure (max, default)', 
            jsonb_build_object('semantic_type', 'Measure', 'data_type', 'max', 'format', 'default', 'notes', 'Measure Type'))
    ON CONFLICT DO NOTHING;

    -- Time + time + default
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, metadata)
    VALUES (lkup_id, tenant_uuid, 'time_time_default', 'Time (time, default)', 
            jsonb_build_object('semantic_type', 'Time', 'data_type', 'time', 'format', 'default', 'notes', 'Dedicated Semantic Time Object'))
    ON CONFLICT DO NOTHING;
  ELSE
    -- Fallback: insert metadata-aware values into lookup_values using lookup_type
    INSERT INTO public.lookup_values (lookup_type, value, label, metadata)
    VALUES
      ('semantic_types', 'dimension_string_default', 'Dimension (string, default)', jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'default')),
      ('semantic_types', 'dimension_string_imageurl', 'Dimension (string, imageUrl)', jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'imageUrl')),
      ('semantic_types', 'dimension_string_link', 'Dimension (string, link)', jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'link')),
      ('semantic_types', 'dimension_string_currency', 'Dimension (string, currency)', jsonb_build_object('semantic_type', 'Dimension', 'data_type', 'string', 'format', 'currency'))
    ON CONFLICT DO NOTHING;
  END IF;
END$$;

-- Create indexes for semantic_types lookup for better query performance
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'value') THEN
    CREATE INDEX IF NOT EXISTS idx_semantic_types_lookup_value ON public.lookup_values(value);
  END IF;
END$$;
