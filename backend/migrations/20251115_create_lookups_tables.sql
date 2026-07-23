-- Migration: Create lookups and lookup_values tables

CREATE TABLE IF NOT EXISTS public.lookups (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  description text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_lookups_tenant ON public.lookups (tenant_id);

CREATE TABLE IF NOT EXISTS public.lookup_values (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  lookup_id uuid NOT NULL,
  tenant_id uuid NOT NULL,
  value text NOT NULL,
  label text NOT NULL,
  parent_id uuid,
  metadata jsonb DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'lookup_id') THEN
    CREATE INDEX IF NOT EXISTS idx_lookup_values_lookup_id ON public.lookup_values (lookup_id);
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'parent_id') THEN
    CREATE INDEX IF NOT EXISTS idx_lookup_values_parent_id ON public.lookup_values (parent_id);
  END IF;
END$$;

-- A small seed for a hierarchical domains lookup as an example
INSERT INTO public.lookups (tenant_id, name, description)
SELECT t.id, 'domains', 'Hierarchical domain taxonomy' FROM tenants t LIMIT 1
ON CONFLICT DO NOTHING;

-- If the lookup 'domains' was created above, insert hierarchical values for levels
DO $$
DECLARE
  lkup_id uuid;
  tenant_uuid uuid;
BEGIN
  SELECT id INTO lkup_id FROM public.lookups WHERE name = 'domains' LIMIT 1;
  IF lkup_id IS NULL THEN
    RETURN;
  END IF;
  SELECT tenant_id INTO tenant_uuid FROM public.lookups WHERE id = lkup_id LIMIT 1;

  -- Top-level example values (idempotent)
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'lookup_id') THEN
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES (lkup_id, tenant_uuid, 'finance', 'Finance'), (lkup_id, tenant_uuid, 'operations', 'Operations')
    ON CONFLICT DO NOTHING;

    -- Example second-level (associate with the finance parent)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'lookup_values' AND column_name = 'parent_id') THEN
      INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, parent_id)
      SELECT lkup_id, tenant_uuid, 'capital_markets', 'Capital Markets', lv.id
      FROM public.lookup_values lv
      WHERE lv.lookup_id = lkup_id AND lv.value = 'finance'
      LIMIT 1
      ON CONFLICT DO NOTHING;

      -- Example third-level (associate with the capital_markets parent)
      INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label, parent_id)
      SELECT lkup_id, tenant_uuid, 'equities', 'Equities', lv.id
      FROM public.lookup_values lv
      WHERE lv.lookup_id = lkup_id AND lv.value = 'capital_markets'
      LIMIT 1
      ON CONFLICT DO NOTHING;
    END IF;

    -- ISO Countries (sample subset). If you want the full ISO 3166 list, we can expand this.
    INSERT INTO public.lookups (tenant_id, name, description)
    SELECT tenant_uuid, 'iso_countries', 'ISO 3166 Country Codes' WHERE NOT EXISTS (SELECT 1 FROM public.lookups WHERE name = 'iso_countries' AND tenant_id = tenant_uuid);

    PERFORM (SELECT 1) FROM public.lookups WHERE name = 'iso_countries' AND tenant_id = tenant_uuid;
    lkup_id := (SELECT id FROM public.lookups WHERE name = 'iso_countries' AND tenant_id = tenant_uuid LIMIT 1);

    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES
      (lkup_id, tenant_uuid, 'US', 'United States'),
      (lkup_id, tenant_uuid, 'GB', 'United Kingdom'),
      (lkup_id, tenant_uuid, 'CA', 'Canada'),
      (lkup_id, tenant_uuid, 'FR', 'France'),
      (lkup_id, tenant_uuid, 'DE', 'Germany'),
      (lkup_id, tenant_uuid, 'CN', 'China'),
      (lkup_id, tenant_uuid, 'IN', 'India'),
      (lkup_id, tenant_uuid, 'JP', 'Japan'),
      (lkup_id, tenant_uuid, 'AU', 'Australia'),
      (lkup_id, tenant_uuid, 'BR', 'Brazil'),
      (lkup_id, tenant_uuid, 'ZA', 'South Africa'),
      (lkup_id, tenant_uuid, 'RU', 'Russia')
    ON CONFLICT DO NOTHING;

    -- ISO Currencies (sample subset). Add more currencies as needed.
    INSERT INTO public.lookups (tenant_id, name, description)
    SELECT tenant_uuid, 'iso_currencies', 'ISO 4217 Currency Codes' WHERE NOT EXISTS (SELECT 1 FROM public.lookups WHERE name = 'iso_currencies' AND tenant_id = tenant_uuid);

    lkup_id := (SELECT id FROM public.lookups WHERE name = 'iso_currencies' AND tenant_id = tenant_uuid LIMIT 1);

    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES
      (lkup_id, tenant_uuid, 'USD', 'US Dollar'),
      (lkup_id, tenant_uuid, 'EUR', 'Euro'),
      (lkup_id, tenant_uuid, 'GBP', 'British Pound'),
      (lkup_id, tenant_uuid, 'JPY', 'Japanese Yen'),
      (lkup_id, tenant_uuid, 'CNY', 'Chinese Yuan'),
      (lkup_id, tenant_uuid, 'INR', 'Indian Rupee'),
      (lkup_id, tenant_uuid, 'AUD', 'Australian Dollar'),
      (lkup_id, tenant_uuid, 'CAD', 'Canadian Dollar'),
      (lkup_id, tenant_uuid, 'BRL', 'Brazilian Real'),
      (lkup_id, tenant_uuid, 'ZAR', 'South African Rand'),
      (lkup_id, tenant_uuid, 'RUB', 'Russian Ruble')
    ON CONFLICT DO NOTHING;
  ELSE
    -- Fallback for environments that already have a different lookup_values schema (e.g., lookup_type instead of lookup_id)
    INSERT INTO public.lookup_values (lookup_type, value, label)
    VALUES ('domains', 'finance', 'Finance'), ('domains', 'operations', 'Operations')
    ON CONFLICT DO NOTHING;

    -- ISO Countries fallback
    INSERT INTO public.lookups (tenant_id, name, description)
    SELECT tenant_uuid, 'iso_countries', 'ISO 3166 Country Codes' WHERE NOT EXISTS (SELECT 1 FROM public.lookups WHERE name = 'iso_countries' AND tenant_id = tenant_uuid);

    INSERT INTO public.lookup_values (lookup_type, value, label)
    VALUES
      ('iso_countries', 'US', 'United States'),
      ('iso_countries', 'GB', 'United Kingdom'),
      ('iso_countries', 'CA', 'Canada')
    ON CONFLICT DO NOTHING;

    -- ISO Currencies fallback
    INSERT INTO public.lookups (tenant_id, name, description)
    SELECT tenant_uuid, 'iso_currencies', 'ISO 4217 Currency Codes' WHERE NOT EXISTS (SELECT 1 FROM public.lookups WHERE name = 'iso_currencies' AND tenant_id = tenant_uuid);

    INSERT INTO public.lookup_values (lookup_type, value, label)
    VALUES
      ('iso_currencies', 'USD', 'US Dollar'),
      ('iso_currencies', 'EUR', 'Euro'),
      ('iso_currencies', 'GBP', 'British Pound')
    ON CONFLICT DO NOTHING;
  END IF;
END$$;
