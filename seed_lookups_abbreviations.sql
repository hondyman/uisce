-- Seed Lookups and Abbreviations

-- Get the Uisce tenant ID
DO $$
DECLARE
  uisce_tenant_id uuid;
  lkup_id uuid;
BEGIN
  -- Get Uisce tenant ID
  SELECT id INTO uisce_tenant_id FROM tenants WHERE name = 'uisce' OR gold_copy = true LIMIT 1;
  
  IF uisce_tenant_id IS NULL THEN
    RAISE NOTICE 'No gold copy tenant found, skipping lookup seeding';
    RETURN;
  END IF;

  -- Seed Lookups
  INSERT INTO public.lookups (tenant_id, name, description)
  VALUES 
    (uisce_tenant_id, 'domains', 'Hierarchical domain taxonomy'),
    (uisce_tenant_id, 'iso_countries', 'ISO 3166 Country Codes'),
    (uisce_tenant_id, 'iso_currencies', 'ISO 4217 Currency Codes')
  ON CONFLICT DO NOTHING;

  -- Seed Domain Lookup Values
  SELECT id INTO lkup_id FROM public.lookups WHERE name = 'domains' AND tenant_id = uisce_tenant_id LIMIT 1;
  IF lkup_id IS NOT NULL THEN
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES 
      (lkup_id, uisce_tenant_id, 'finance', 'Finance'),
      (lkup_id, uisce_tenant_id, 'operations', 'Operations'),
      (lkup_id, uisce_tenant_id, 'risk', 'Risk Management'),
      (lkup_id, uisce_tenant_id, 'compliance', 'Compliance')
    ON CONFLICT DO NOTHING;
  END IF;

  -- Seed Country Lookup Values
  SELECT id INTO lkup_id FROM public.lookups WHERE name = 'iso_countries' AND tenant_id = uisce_tenant_id LIMIT 1;
  IF lkup_id IS NOT NULL THEN
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES
      (lkup_id, uisce_tenant_id, 'US', 'United States'),
      (lkup_id, uisce_tenant_id, 'GB', 'United Kingdom'),
      (lkup_id, uisce_tenant_id, 'CA', 'Canada'),
      (lkup_id, uisce_tenant_id, 'FR', 'France'),
      (lkup_id, uisce_tenant_id, 'DE', 'Germany'),
      (lkup_id, uisce_tenant_id, 'CN', 'China'),
      (lkup_id, uisce_tenant_id, 'IN', 'India'),
      (lkup_id, uisce_tenant_id, 'JP', 'Japan')
    ON CONFLICT DO NOTHING;
  END IF;

  -- Seed Currency Lookup Values
  SELECT id INTO lkup_id FROM public.lookups WHERE name = 'iso_currencies' AND tenant_id = uisce_tenant_id LIMIT 1;
  IF lkup_id IS NOT NULL THEN
    INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
    VALUES
      (lkup_id, uisce_tenant_id, 'USD', 'US Dollar'),
      (lkup_id, uisce_tenant_id, 'EUR', 'Euro'),
      (lkup_id, uisce_tenant_id, 'GBP', 'British Pound'),
      (lkup_id, uisce_tenant_id, 'JPY', 'Japanese Yen'),
      (lkup_id, uisce_tenant_id, 'CNY', 'Chinese Yuan')
    ON CONFLICT DO NOTHING;
  END IF;

  -- Seed Common Abbreviations
  INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
  VALUES
    ('ACCT', 'ACCOUNT', 'Account identifier', uisce_tenant_id::varchar),
    ('AMT', 'AMOUNT', 'Monetary amount', uisce_tenant_id::varchar),
    ('BAL', 'BALANCE', 'Account balance', uisce_tenant_id::varchar),
    ('CUST', 'CUSTOMER', 'Customer identifier', uisce_tenant_id::varchar),
    ('DESC', 'DESCRIPTION', 'Description field', uisce_tenant_id::varchar),
    ('DT', 'DATE', 'Date field', uisce_tenant_id::varchar),
    ('ID', 'IDENTIFIER', 'Unique identifier', uisce_tenant_id::varchar),
    ('NUM', 'NUMBER', 'Numeric value', uisce_tenant_id::varchar),
    ('QTY', 'QUANTITY', 'Quantity amount', uisce_tenant_id::varchar),
    ('REF', 'REFERENCE', 'Reference number', uisce_tenant_id::varchar),
    ('SRC', 'SOURCE', 'Source system', uisce_tenant_id::varchar),
    ('TXN', 'TRANSACTION', 'Transaction record', uisce_tenant_id::varchar),
    ('VAL', 'VALUE', 'Value field', uisce_tenant_id::varchar)
  ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

END$$;
