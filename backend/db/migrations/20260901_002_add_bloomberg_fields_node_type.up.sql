-- Add BLOOMBERG_FIELD catalog_node_type and seed for multi-tenant catalog
-- Enables Bloomberg Data License dictionary mapping into catalog_node graph

DO $$
DECLARE
    v_tenant RECORD;
BEGIN
    -- Seed BLOOMBERG_FIELD node type for all tenants
    FOR v_tenant IN SELECT id FROM tenants LOOP
        INSERT INTO public.catalog_node_type (
            id,
            tenant_id,
            catalog_type_name,
            description,
            config
        )
        VALUES (
            gen_random_uuid(),
            v_tenant.id,
            'BLOOMBERG_FIELD',
            'Bloomberg Fields - Standard Data License field dictionary definitions and asset class eligibility',
            '{
                "display_name": "Bloomberg Fields",
                "icon": "LineChart",
                "color": "#FF6D00",
                "category": "MARKET_DATA_DICTIONARY",
                "properties_schema": {
                    "field_id": {"type": "string", "description": "Bloomberg 4-character Field ID (e.g. DS62, RK90)"},
                    "mnemonic": {"type": "string", "description": "Field Mnemonic (e.g. 144A_FLAG, 10Y_ASK_CDS_SPREAD)"},
                    "description": {"type": "string", "description": "Field short descriptive label"},
                    "definition": {"type": "string", "description": "Full business and formula definition"},
                    "data_license_category": {"type": "string", "description": "Data License Category (e.g. Derived Data, Security Master)"},
                    "category": {"type": "string", "description": "Subcategory (e.g. Descriptive Info, Analytics - Risk Measures)"},
                    "field_type": {"type": "string", "description": "Data type: Real, Character, Boolean, Date, etc."},
                    "standard_width": {"type": "integer"},
                    "standard_decimal_places": {"type": "integer"},
                    "production_date": {"type": "string"},
                    "market_sectors": {
                        "type": "object",
                        "description": "Eligibility across Bloomberg yellow key market sectors",
                        "properties": {
                            "comdty": {"type": "boolean"},
                            "equity": {"type": "boolean"},
                            "muni": {"type": "boolean"},
                            "pfd": {"type": "boolean"},
                            "mmkt": {"type": "boolean"},
                            "govt": {"type": "boolean"},
                            "corp": {"type": "boolean"},
                            "index": {"type": "boolean"},
                            "curncy": {"type": "boolean"},
                            "mtge": {"type": "boolean"}
                        }
                    }
                }
            }'::jsonb
        )
        ON CONFLICT (tenant_id, catalog_type_name) DO UPDATE SET
            description = EXCLUDED.description,
            config = EXCLUDED.config;
    END LOOP;

    -- Also ensure default/fallback tenant has BLOOMBERG_FIELD
    INSERT INTO public.catalog_node_type (
        id,
        tenant_id,
        catalog_type_name,
        description,
        config
    )
    VALUES (
        gen_random_uuid(),
        '00000000-0000-0000-0000-000000000000'::UUID,
        'BLOOMBERG_FIELD',
        'Bloomberg Fields - Standard Data License field dictionary definitions and asset class eligibility',
        '{
            "display_name": "Bloomberg Fields",
            "icon": "LineChart",
            "color": "#FF6D00",
            "category": "MARKET_DATA_DICTIONARY"
        }'::jsonb
    )
    ON CONFLICT (tenant_id, catalog_type_name) DO UPDATE SET
        description = EXCLUDED.description,
        config = EXCLUDED.config;

END $$;
