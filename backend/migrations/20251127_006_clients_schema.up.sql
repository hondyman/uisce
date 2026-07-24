-- Clients table for multi-tenant architecture
-- Each tenant's database instance has its own clients table

CREATE TABLE IF NOT EXISTS public.clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Client identification
    client_name TEXT NOT NULL,
    client_code TEXT, -- Optional short code/identifier
    
    -- Client type (individual, trust, corporate, etc.)
    client_type TEXT DEFAULT 'individual',
    
    -- Contact information
    email TEXT,
    phone TEXT,
    address_line1 TEXT,
    address_line2 TEXT,
    city TEXT,
    state TEXT,
    postal_code TEXT,
    country TEXT DEFAULT 'US',
    
    -- Tax information
    tax_id TEXT, -- SSN for individuals, EIN for entities
    tax_filing_status TEXT,
    
    -- Account settings
    is_active BOOLEAN DEFAULT true,
    risk_tolerance TEXT DEFAULT 'moderate', -- conservative, moderate, aggressive
    
    -- Metadata
    notes TEXT,
    tags TEXT[],
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_clients_name ON public.clients(client_name);
CREATE INDEX IF NOT EXISTS idx_clients_code ON public.clients(client_code);
CREATE INDEX IF NOT EXISTS idx_clients_email ON public.clients(email);
CREATE INDEX IF NOT EXISTS idx_clients_active ON public.clients(is_active);
CREATE INDEX IF NOT EXISTS idx_clients_type ON public.clients(client_type);

-- Accounts table (optional but helpful for crypto wallets)
CREATE TABLE IF NOT EXISTS public.accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    
    -- Account details
    account_name TEXT NOT NULL,
    account_number TEXT,
    account_type TEXT DEFAULT 'investment', -- investment, retirement, trust, crypto
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    opened_date DATE DEFAULT CURRENT_DATE,
    closed_date DATE,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_client ON public.accounts(client_id);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON public.accounts(account_type);
CREATE INDEX IF NOT EXISTS idx_accounts_active ON public.accounts(is_active);

-- Portfolio holdings table (for traditional assets, to calculate crypto allocation %)
CREATE TABLE IF NOT EXISTS public.portfolio_holdings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    account_id UUID REFERENCES public.accounts(id) ON DELETE CASCADE,
    
    -- Asset details
    asset_symbol TEXT NOT NULL,
    asset_name TEXT,
    asset_class TEXT, -- equity, fixed_income, alternative, crypto
    quantity NUMERIC(28,8) NOT NULL,
    
    -- Valuation
    cost_basis NUMERIC(15,2),
    current_price NUMERIC(15,8),
    current_value NUMERIC(15,2),
    
    -- Timestamps
    as_of_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_client ON public.portfolio_holdings(client_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_account ON public.portfolio_holdings(account_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_asset_class ON public.portfolio_holdings(asset_class);

COMMENT ON TABLE public.clients IS 'Client records for this tenant''s database instance';
COMMENT ON TABLE public.accounts IS 'Client accounts for organizing holdings';
COMMENT ON TABLE public.portfolio_holdings IS 'Traditional asset holdings for portfolio allocation calculations';
