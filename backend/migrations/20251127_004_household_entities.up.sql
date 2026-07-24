-- Household Complexity Management Schema
-- Phase 4: Multi-Entity Household Management

-- ===========================
-- ENTITIES TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS entities (
    entity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN (
        'INDIVIDUAL',
        'JOINT',
        'TRUST',
        'LLC',
        'FOUNDATION',
        'ESTATE',
        'PARTNERSHIP',
        'CORPORATION'
    )),
    entity_name TEXT NOT NULL,
    tax_id VARCHAR(20),
    
    -- Hierarchy
    parent_entity_id UUID REFERENCES entities(entity_id),
    household_id UUID NOT NULL,
    
    -- Trust-specific fields
    trust_type VARCHAR(50) CHECK (trust_type IN (
        'REVOCABLE',
        'IRREVOCABLE',
        'CHARITABLE',
        'DYNASTY',
        'SPECIAL_NEEDS',
        'QTIP',
        'GRAT',
        'SLAT'
    )),
    trustee_ids UUID[],
    grantor_id UUID,
    beneficiary_ids UUID[],
    trust_termination_date DATE,
    
    -- Foundation-specific fields
    foundation_type VARCHAR(50) CHECK (foundation_type IN (
        'PRIVATE',
        'DONOR_ADVISED_FUND',
        'SUPPORTING_ORGANIZATION'
    )),
    annual_distribution_requirement DECIMAL(5,4), -- e.g., 0.05 = 5% for private foundations
    
    -- LLC-specific fields
    ownership_structure JSONB, -- {"member1": 50, "member2": 50}
    operating_agreement_url TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_entities_household ON entities(household_id);
CREATE INDEX idx_entities_type ON entities(entity_type);
CREATE INDEX idx_entities_parent ON entities(parent_entity_id);

-- ===========================
-- INTER-ENTITY TRANSFERS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS inter_entity_transfers (
    transfer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_entity_id UUID NOT NULL REFERENCES entities(entity_id),
    to_entity_id UUID NOT NULL REFERENCES entities(entity_id),
    transfer_date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    asset_description TEXT,
    transfer_reason VARCHAR(50) CHECK (transfer_reason IN (
        'GIFT',
        'LOAN',
        'DISTRIBUTION',
        'CONTRIBUTION',
        'SALE',
        'EXCHANGE'
    )),
    
    -- Tax implications
    gift_tax_return_required BOOLEAN DEFAULT FALSE,
    generation_skipping_transfer BOOLEAN DEFAULT FALSE,
    
    advisor_notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_inter_entity_from ON inter_entity_transfers(from_entity_id);
CREATE INDEX idx_inter_entity_to ON inter_entity_transfers(to_entity_id);
CREATE INDEX idx_inter_entity_date ON inter_entity_transfers(transfer_date);

-- ===========================
-- HOUSEHOLD CONSOLIDATED VIEW
-- ===========================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'accounts') THEN
    EXECUTE $exec$
      CREATE OR REPLACE VIEW household_consolidated AS
      SELECT 
          e.household_id,
          COUNT(DISTINCT e.entity_id) as entity_count,
          ARRAY_AGG(DISTINCT e.entity_type) as entity_types,
          SUM(COALESCE(a.balance, 0)) as total_net_worth
      FROM entities e
      LEFT JOIN accounts a ON e.entity_id = a.owner_entity_id
      WHERE e.is_active = TRUE
      GROUP BY e.household_id;
    $exec$;
  ELSE
    EXECUTE $exec$
      CREATE OR REPLACE VIEW household_consolidated AS
      SELECT e.household_id, COUNT(DISTINCT e.entity_id) as entity_count, ARRAY_AGG(DISTINCT e.entity_type) as entity_types, 0::numeric as total_net_worth
      FROM entities e
      WHERE e.is_active = TRUE
      GROUP BY e.household_id;
    $exec$;
  END IF;
END$$;
COMMENT ON TABLE entities IS 'Multi-entity household structure with trusts, LLCs, and foundations';
COMMENT ON TABLE inter_entity_transfers IS 'Transfers between household entities with tax flagging';
