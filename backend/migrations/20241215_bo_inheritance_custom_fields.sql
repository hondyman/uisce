-- ============================================================================
-- BUSINESS OBJECT INHERITANCE & CUSTOM FIELDS
-- Workday-style field layer: Inherited → Subtype Core → Custom
-- ============================================================================

-- Add parent_bo_id to business_objects for inheritance
-- Ensure tenants has code column (missing from 0001)
ALTER TABLE public.tenants ADD COLUMN IF NOT EXISTS code TEXT;
ALTER TABLE public.tenants ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenants_code_uniq') THEN
        ALTER TABLE public.tenants ADD CONSTRAINT tenants_code_uniq UNIQUE (code);
    END IF;
END $$;

INSERT INTO public.tenants (id, code, name, is_active)
VALUES (gen_random_uuid(), 'default-tenant', 'Default Tenant', true)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE public.business_objects 
ADD COLUMN IF NOT EXISTS parent_bo_id uuid REFERENCES public.business_objects(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS description text,
ADD COLUMN IF NOT EXISTS icon text,
ADD CONSTRAINT business_objects_tenant_key_uniq UNIQUE (tenant_id, key);

ALTER TABLE public.business_objects ALTER COLUMN tenant_datasource_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_bo_parent ON public.business_objects(parent_bo_id);

COMMENT ON COLUMN public.business_objects.parent_bo_id IS 'Points to parent BO for inheritance (e.g., Individual Investor → Client Investor)';

-- Add is_custom and is_inherited flags to bo_fields
ALTER TABLE public.bo_fields
ADD COLUMN IF NOT EXISTS is_custom BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS is_inherited BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS inherited_from_bo_id uuid REFERENCES public.business_objects(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bo_fields_custom ON public.bo_fields(is_custom);
CREATE INDEX IF NOT EXISTS idx_bo_fields_inherited ON public.bo_fields(is_inherited);

COMMENT ON COLUMN public.bo_fields.is_custom IS 'True if field is a tenant customization (not core). Set based on ADMIN_CORE env var.';
COMMENT ON COLUMN public.bo_fields.is_inherited IS 'True if field is inherited from parent BO';
COMMENT ON COLUMN public.bo_fields.inherited_from_bo_id IS 'ID of parent BO from which this field was inherited';

-- ============================================================================
-- SEED CLIENT INVESTOR HIERARCHY
-- ============================================================================

-- Parent: Client Investor
INSERT INTO public.business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, parent_bo_id)
VALUES (
    'fdc20a84-00ba-4050-8919-548c772c7201',
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'client_investor',
    'Client Investor',
    'Client Investor',
    'client_investor',
    'Base investor entity with common fields for all investor types',
    'person',
    true,
    NULL
) ON CONFLICT (tenant_id, key) DO UPDATE
  SET display_name = EXCLUDED.display_name,
      description = EXCLUDED.description;

-- Add base fields to Client Investor
INSERT INTO public.bo_fields (tenant_id, business_object_id, subtype_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
SELECT 
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'fdc20a84-00ba-4050-8919-548c772c7201',
    NULL,
    key,
    name,
    label,
    name,
    type,
    true,
    required,
    seq
FROM (VALUES
    ('investor_id', 'Investor ID', 'Investor ID', 'text', true, 1),
    ('full_name', 'Full Name', 'Full Name', 'text', true, 2),
    ('email', 'Email', 'Email Address', 'email', true, 3),
    ('phone', 'Phone', 'Phone Number', 'text', false, 4),
    ('address', 'Address', 'Mailing Address', 'text', false, 5),
    ('city', 'City', 'City', 'text', false, 6),
    ('state', 'State', 'State/Province', 'text', false, 7),
    ('postal_code', 'Postal Code', 'Postal Code', 'text', false, 8),
    ('country', 'Country', 'Country', 'text', false, 9),
    ('account_opened_date', 'Account Opened', 'Account Opened Date', 'date', false, 10),
    ('status', 'Status', 'Account Status', 'text', false, 11),
    ('risk_profile', 'Risk Profile', 'Risk Profile', 'text', false, 12)
) AS t(key, name, label, type, required, seq)
ON CONFLICT DO NOTHING;

-- Child: Individual Investor (inherits from Client Investor)
INSERT INTO public.business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, parent_bo_id)
VALUES (
    'fdc20a84-00ba-4050-8919-548c772c7202',
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'individual_investor',
    'Individual Investor',
    'Individual Investor',
    'individual_investor',
    'Individual investor with personal details (inherits from Client Investor)',
    'person_outline',
    true,
    'fdc20a84-00ba-4050-8919-548c772c7201'
) ON CONFLICT (tenant_id, key) DO UPDATE
  SET parent_bo_id = EXCLUDED.parent_bo_id,
      display_name = EXCLUDED.display_name;

-- Add Individual-specific core fields
INSERT INTO public.bo_fields (tenant_id, business_object_id, subtype_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
SELECT 
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'fdc20a84-00ba-4050-8919-548c772c7202',
    NULL,
    key,
    name,
    label,
    name,
    type,
    true,
    required,
    seq
FROM (VALUES
    ('date_of_birth', 'Date of Birth', 'Date of Birth', 'date', false, 20),
    ('ssn', 'SSN', 'Social Security Number', 'text', false, 21),
    ('employment_status', 'Employment Status', 'Employment Status', 'text', false, 22),
    ('annual_income', 'Annual Income', 'Annual Income', 'currency', false, 23),
    ('net_worth', 'Net Worth', 'Net Worth', 'currency', false, 24),
    ('investment_objective', 'Investment Objective', 'Investment Objective', 'text', false, 25),
    ('marital_status', 'Marital Status', 'Marital Status', 'text', false, 26)
) AS t(key, name, label, type, required, seq)
ON CONFLICT DO NOTHING;

-- Child: Institutional Investor (inherits from Client Investor)
INSERT INTO public.business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, parent_bo_id)
VALUES (
    'fdc20a84-00ba-4050-8919-548c772c7203',
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'institutional_investor',
    'Institutional Investor',
    'Institutional Investor',
    'institutional_investor',
    'Institutional investor with organizational details (inherits from Client Investor)',
    'business',
    true,
    'fdc20a84-00ba-4050-8919-548c772c7201'
) ON CONFLICT (tenant_id, key) DO UPDATE
  SET parent_bo_id = EXCLUDED.parent_bo_id,
      display_name = EXCLUDED.display_name;

-- Add Institutional-specific core fields
INSERT INTO public.bo_fields (tenant_id, business_object_id, subtype_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
SELECT 
    (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
    'fdc20a84-00ba-4050-8919-548c772c7203',
    NULL,
    key,
    name,
    label,
    name,
    type,
    true,
    required,
    seq
FROM (VALUES
    ('legal_name', 'Legal Name', 'Legal Entity Name', 'text', true, 20),
    ('ein', 'EIN', 'Employer Identification Number', 'text', false, 21),
    ('entity_type', 'Entity Type', 'Entity Type', 'text', false, 22),
    ('incorporation_date', 'Incorporation Date', 'Date of Incorporation', 'date', false, 23),
    ('aum', 'AUM', 'Assets Under Management', 'currency', false, 24),
    ('primary_contact', 'Primary Contact', 'Primary Contact Person', 'text', false, 25),
    ('contact_title', 'Contact Title', 'Contact Title', 'text', false, 26),
    ('investment_committee', 'Investment Committee', 'Investment Committee Members', 'text', false, 27)
) AS t(key, name, label, type, required, seq)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- FUNCTION: Get All Fields for BO (including inherited)
-- ============================================================================

CREATE OR REPLACE FUNCTION get_all_bo_fields(bo_id uuid)
RETURNS TABLE (
    field_id uuid,
    field_key varchar,
    field_name varchar,
    field_type varchar,
    is_core boolean,
    is_required boolean,
    is_custom boolean,
    is_inherited boolean,
    inherited_from varchar,
    sequence integer
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE bo_hierarchy AS (
        -- Start with the current BO
        SELECT id, parent_bo_id, key, 0 AS level
        FROM public.business_objects
        WHERE id = bo_id
        
        UNION ALL
        
        -- Recursively get parent BOs
        SELECT bo.id, bo.parent_bo_id, bo.key, bh.level + 1
        FROM public.business_objects bo
        INNER JOIN bo_hierarchy bh ON bo.id = bh.parent_bo_id
    )
    SELECT 
        f.id AS field_id,
        f.key AS field_key,
        f.name AS field_name,
        f.type AS field_type,
        f.is_core,
        f.is_required,
        f.is_custom,
        CASE WHEN f.business_object_id != bo_id THEN true ELSE false END AS is_inherited,
        CASE WHEN f.business_object_id != bo_id THEN bh.key ELSE NULL END AS inherited_from,
        f.sequence
    FROM bo_hierarchy bh
    INNER JOIN public.bo_fields f ON f.business_object_id = bh.id
    WHERE f.subtype_id IS NULL
    ORDER BY bh.level DESC, f.sequence ASC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Business Object Inheritance schema created';
    RAISE NOTICE '✓ Added parent_bo_id to business_objects';
    RAISE NOTICE '✓ Added is_custom, is_inherited flags to bo_fields';
    RAISE NOTICE '✓ Seeded: Client Investor → Individual Investor, Institutional Investor';
    RAISE NOTICE '✓ Created get_all_bo_fields() function for inheritance resolution';
END $$;
