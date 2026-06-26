-- Entity structure modeling
CREATE TABLE IF NOT EXISTS entities (
    entity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL, -- 'INDIVIDUAL', 'JOINT', 'TRUST', 'LLC', 'FOUNDATION', 'ESTATE'
    entity_name TEXT NOT NULL,
    tax_id VARCHAR(20),
    
    -- Hierarchy
    parent_entity_id UUID REFERENCES entities(entity_id), -- For trusts owned by other trusts
    household_id UUID NOT NULL, -- Groups related entities. Assuming households table exists or using a UUID for logical grouping.
    
    -- Trust-specific
    trust_type VARCHAR(50), -- 'REVOCABLE', 'IRREVOCABLE', 'CHARITABLE', 'DYNASTY'
    trustee_ids UUID[], -- Array of person IDs (users or other entities)
    grantor_id UUID,
    beneficiary_ids UUID[],
    trust_termination_date DATE,
    
    -- Foundation-specific
    foundation_type VARCHAR(50), -- 'PRIVATE', 'DONOR_ADVISED_FUND'
    annual_distribution_requirement DECIMAL(5,4), -- 5% for private foundations
    
    -- LLC-specific
    ownership_structure JSONB, -- {"member1": 50, "member2": 50}
    operating_agreement_url TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entities_household ON entities(household_id);
CREATE INDEX IF NOT EXISTS idx_entities_parent ON entities(parent_entity_id);

-- Inter-entity transactions (e.g., gift to trust)
CREATE TABLE IF NOT EXISTS inter_entity_transfers (
    transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_entity_id UUID REFERENCES entities(entity_id),
    to_entity_id UUID REFERENCES entities(entity_id),
    transfer_date DATE NOT NULL,
    amount DECIMAL(15,2),
    asset_description TEXT,
    transfer_reason VARCHAR(50), -- 'GIFT', 'LOAN', 'DISTRIBUTION', 'CONTRIBUTION'
    
    -- Tax implications
    gift_tax_return_required BOOLEAN DEFAULT FALSE,
    generation_skipping_transfer BOOLEAN DEFAULT FALSE,
    
    advisor_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_from ON inter_entity_transfers(from_entity_id);
CREATE INDEX IF NOT EXISTS idx_transfers_to ON inter_entity_transfers(to_entity_id);

-- Consolidated household view
-- Note: This view assumes existence of 'households' and 'accounts' tables. 
-- If they don't exist yet in the schema, I will comment this out or create placeholders.
-- Based on previous context, 'clients' table exists. 'households' might not.
-- I will create a simple households table if it doesn't exist to make this robust.

CREATE TABLE IF NOT EXISTS households (
    household_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_name TEXT NOT NULL,
    primary_contact_id UUID, -- REFERENCES users(id)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Assuming 'accounts' table exists and has 'entity_id' or 'client_id'.
-- If 'accounts' links to 'client_id', we might need to migrate it to link to 'entity_id' or add a column.
-- For now, I'll assume we can link via entity_id.

/*
CREATE VIEW household_consolidated AS
SELECT 
    h.household_id,
    h.household_name,
    -- SUM(a.balance) as total_net_worth, -- Requires accounts table
    COUNT(DISTINCT e.entity_id) as entity_count,
    -- COUNT(DISTINCT a.account_id) as account_count,
    JSONB_AGG(DISTINCT e.entity_type) as entity_types
FROM households h
JOIN entities e ON h.household_id = e.household_id
-- LEFT JOIN accounts a ON e.entity_id = a.entity_id
GROUP BY h.household_id;
*/
