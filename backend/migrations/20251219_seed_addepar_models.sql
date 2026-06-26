-- Seed Addepar Business Object Definitions


        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'household',
            'Represents a household portfolio entity.',
            '{"display_name": "Household", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'person_node',
            'Represents an individual client or person.',
            '{"display_name": "Client", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'prospect',
            'Represents a prospective client.',
            '{"display_name": "Prospect", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'managed_partnership',
            'Represents a managed fund or partnership.',
            '{"display_name": "Managed fund", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'holding_company',
            'Represents a holding company entity.',
            '{"display_name": "Holding company", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'manager',
            'Represents a manager entity.',
            '{"display_name": "Manager", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'fund',
            'Represents a private fund.',
            '{"display_name": "Private fund", "ownership_type": "Value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'trust',
            'Represents a trust entity.',
            '{"display_name": "Trust", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'vehicle',
            'Represents a vehicle entity.',
            '{"display_name": "Vehicle", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'financial_account',
            'Represents a financial or holding account.',
            '{"display_name": "Holding Account", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'sleeve',
            'Represents a sleeve in a portfolio.',
            '{"display_name": "Sleeve", "ownership_type": "Percent-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'annuity',
            'Represents an annuity investment.',
            '{"display_name": "Annuity", "ownership_type": "Value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'art',
            'Represents art as an asset. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Art", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'bond',
            'Represents a bond investment.',
            '{"display_name": "Bond", "ownership_type": "Share-based", "suggested_attributes": [{"key": "cusip", "value_type": "string"}, {"key": "maturity_date", "value_type": "date"}, {"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'car',
            'Represents a car as an asset. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Car", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'certificate_of_deposit',
            'Represents a certificate of deposit.',
            '{"display_name": "Certificate of deposit", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'closed_end_fund',
            'Represents a closed-end fund.',
            '{"display_name": "Closed end fund", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'cmo',
            'Represents a collateralized mortgage obligation.',
            '{"display_name": "CMO", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'collectible',
            'Represents a collectible asset. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Collectible", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'convertible_note',
            'Represents a convertible note.',
            '{"display_name": "Convertible note", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'generic_asset',
            'Catch-all for custom or unspecified assets.',
            '{"display_name": "Custom asset, or any other custom investment type that''s not in this list", "ownership_type": "Any", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'cash',
            'Represents cash or currency holdings.',
            '{"display_name": "Currency", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'digital_asset',
            'Represents digital assets like cryptocurrencies.',
            '{"display_name": "Digital asset", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'etf',
            'Represents an exchange-traded fund.',
            '{"display_name": "ETF", "ownership_type": "Share-based", "suggested_attributes": [{"key": "cusip", "value_type": "string"}, {"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'etn',
            'Represents an exchange-traded note.',
            '{"display_name": "ETN", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'forward_contract',
            'Represents a forward contract.',
            '{"display_name": "Forward contract", "ownership_type": "Share-based", "suggested_attributes": [{"key": "underlying_type", "value_type": "string"}, {"key": "delivery_price", "value_type": "numeric"}, {"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'futures_contract',
            'Represents a futures contract.',
            '{"display_name": "Futures contract", "ownership_type": "Share-based", "suggested_attributes": [{"key": "underlying_type", "value_type": "string"}, {"key": "delivery_price", "value_type": "numeric"}, {"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'hedge_fund',
            'Represents a hedge fund. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Hedge fund", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'historical_segment',
            'Represents historical data segments.',
            '{"display_name": "Historical segment", "ownership_type": "Value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'loan',
            'Represents a loan asset.',
            '{"display_name": "Loan", "ownership_type": "Value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'master_limited_partnership',
            'Represents a master limited partnership.',
            '{"display_name": "Master limited partnership", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'money_market_fund',
            'Represents a money market fund.',
            '{"display_name": "Money market fund", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'mutual_fund',
            'Represents a mutual fund.',
            '{"display_name": "Mutual fund", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'option',
            'Represents an option contract.',
            '{"display_name": "Option", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'preferred_stock',
            'Represents preferred stock.',
            '{"display_name": "Preferred stock", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'private_equity_fund',
            'Represents a private equity fund. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Private equity fund", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'private_investment',
            'Represents a private investment. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Private investment", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'promissory_note',
            'Represents a promissory note. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Promissory note", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'real_estate',
            'Represents real estate assets. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Real estate", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'reit',
            'Represents a real estate investment trust.',
            '{"display_name": "REIT", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'stock',
            'Represents common stock.',
            '{"display_name": "Stock", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'structured_product',
            'Represents a structured product investment.',
            '{"display_name": "Structured product", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'uit',
            'Represents a unit investment trust.',
            '{"display_name": "UIT", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'unknown_security',
            'Catch-all for unknown securities.',
            '{"display_name": "Unknown security", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'venture_capital',
            'Represents venture capital investments. Available only to firms that started using Addepar on or after September 12, 2025.',
            '{"display_name": "Venture capital", "ownership_type": "Share-based or value-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            'warrant',
            'Represents a warrant.',
            '{"display_name": "Warrant", "ownership_type": "Share-based", "suggested_attributes": [{"key": "original_name", "value_type": "string"}, {"key": "display_name", "value_type": "string"}, {"key": "currency_factor", "value_type": "string"}]}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        

