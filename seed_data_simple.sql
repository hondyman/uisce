-- Simplified Seed Script for Lookups and Abbreviations

-- Seed Abbreviations (one at a time to avoid syntax issues)
INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'ACCT', 'ACCOUNT', 'Account identifier', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'AMT', 'AMOUNT', 'Monetary amount', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'BAL', 'BALANCE', 'Account balance', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'CUST', 'CUSTOMER', 'Customer identifier', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'DESC', 'DESCRIPTION', 'Description field', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'DT', 'DATE', 'Date field', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'ID', 'IDENTIFIER', 'Unique identifier', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'NUM', 'NUMBER', 'Numeric value', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'QTY', 'QUANTITY', 'Quantity amount', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'REF', 'REFERENCE', 'Reference number', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'SRC', 'SOURCE', 'Source system', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'TXN', 'TRANSACTION', 'Transaction record', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
SELECT 'VAL', 'VALUE', 'Value field', id::varchar FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT (tenant_id, abbreviation) DO NOTHING;

-- Seed Lookups
INSERT INTO public.lookups (tenant_id, name, description)
SELECT id, 'domains', 'Hierarchical domain taxonomy' FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT DO NOTHING;

INSERT INTO public.lookups (tenant_id, name, description)
SELECT id, 'iso_countries', 'ISO 3166 Country Codes' FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT DO NOTHING;

INSERT INTO public.lookups (tenant_id, name, description)
SELECT id, 'iso_currencies', 'ISO 4217 Currency Codes' FROM tenants WHERE gold_copy = true LIMIT 1
ON CONFLICT DO NOTHING;

-- Seed Lookup Values (Domains)
INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'finance', 'Finance' 
FROM lookups l WHERE l.name = 'domains' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;

INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'operations', 'Operations' 
FROM lookups l WHERE l.name = 'domains' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;

-- Seed Lookup Values (Countries)
INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'US', 'United States' 
FROM lookups l WHERE l.name = 'iso_countries' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;

INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'GB', 'United Kingdom' 
FROM lookups l WHERE l.name = 'iso_countries' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;

-- Seed Lookup Values (Currencies)
INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'USD', 'US Dollar' 
FROM lookups l WHERE l.name = 'iso_currencies' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;

INSERT INTO public.lookup_values (lookup_id, tenant_id, value, label)
SELECT l.id, l.tenant_id, 'EUR', 'Euro' 
FROM lookups l WHERE l.name = 'iso_currencies' AND l.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)
ON CONFLICT DO NOTHING;
