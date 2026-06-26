-- ============================================================================
-- KYC/AML VALIDATION RULES FOR CLIENT ONBOARDING
-- ============================================================================
-- These rules are imported into the validation_rules table and enforce
-- regulatory compliance during the client onboarding process.
-- They integrate with the validation rules engine and can be configured
-- via the UI or directly via these INSERT statements.

-- ============================================================================
-- STEP 1: BASIC KYC REQUIREMENTS
-- ============================================================================

-- Rule 1.1: Identification Number Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_identification_required',
  'field_format',
  'Ensure client has provided identification number (SSN, EIN, PASSPORT)',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'OR',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'identification_number', 'operator', 'IS_NULL'),
      jsonb_build_object('field', 'identification_number', 'operator', 'EQUALS', 'value', '')
    )
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 1.2: Identification Type Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_identification_type_required',
  'field_format',
  'Ensure identification type is specified (SSN, EIN, PASSPORT, DRIVER_LICENSE)',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'OR',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'identification_type', 'operator', 'IS_NULL'),
      jsonb_build_object('field', 'identification_type', 'operator', 'NOT_IN', 
        'values', jsonb_build_array('"SSN"', '"EIN"', '"PASSPORT"', '"DRIVER_LICENSE"'))
    )
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 1.3: Date of Birth Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_dob_required',
  'field_format',
  'Ensure date of birth is provided for age verification',
  ARRAY['clients'],
  jsonb_build_object(
    'field', 'date_of_birth', 'operator', 'IS_NULL'
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 1.4: Minimum Age (18 years)
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_minimum_age',
  'business_logic',
  'Ensure client is at least 18 years old',
  ARRAY['clients'],
  jsonb_build_object(
    'type', 'date_range',
    'field', 'date_of_birth',
    'operator', '<=',
    'value', 'NOW() - INTERVAL 18 YEARS'
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 1.5: Country of Citizenship Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_country_citizenship_required',
  'field_format',
  'Ensure country of citizenship is specified',
  ARRAY['clients'],
  jsonb_build_object(
    'field', 'country_of_citizenship', 'operator', 'IS_NULL'
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 2: AML SCREENING REQUIREMENTS
-- ============================================================================

-- Rule 2.1: AML Screening Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'aml_screening_required',
  'business_logic',
  'Ensure AML screening has been performed before onboarding',
  ARRAY['clients'],
  jsonb_build_object(
    'field', 'aml_status', 'operator', 'NOT_IN', 
    'values', jsonb_build_array('"approved"', '"pending"')
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 2.2: No Sanctions Matches
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'aml_no_sanctions_match',
  'business_logic',
  'Ensure client does not appear on sanctions lists (OFAC, UN, EU, etc)',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'aml_status', 'operator', 'EQUALS', 'value', 'rejected'),
      jsonb_build_object('field', 'aml_screening_result', 'path', 'sanctions_match', 'operator', 'EQUALS', 'value', 'true')
    )
  ),
  'critical',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 2.3: PEP (Politically Exposed Person) Check
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'aml_pep_check',
  'business_logic',
  'Flag if client is identified as PEP - requires escalation to compliance',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'aml_screening_result', 'path', 'pep_match', 'operator', 'EQUALS', 'value', 'true')
    )
  ),
  'warning',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 2.4: High Risk Score
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'aml_high_risk_score',
  'business_logic',
  'Flag clients with AML risk score > 70 for enhanced due diligence',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'aml_screening_result', 'path', 'risk_score', 'operator', '>', 'value', 70)
    )
  ),
  'warning',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 3: RISK PROFILE & WEALTH ANALYSIS
-- ============================================================================

-- Rule 3.1: High Net Worth Due Diligence ($5M+)
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_high_net_worth_due_diligence',
  'business_logic',
  'Clients with net worth > $5M require additional due diligence documentation',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'net_worth', 'operator', '>', 'value', 5000000)
    )
  ),
  'warning',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 3.2: Very High Net Worth ($10M+) Advisor Review Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_very_high_net_worth_advisor_review',
  'business_logic',
  'Clients with net worth > $10M must have senior advisor review before approval',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'net_worth', 'operator', '>', 'value', 10000000)
    )
  ),
  'warning',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 3.3: Source of Funds Documentation Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'aml_source_of_funds_documentation',
  'business_logic',
  'Clients with initial funding > $250K require source of funds documentation',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('path', 'workflow_context.initial_funding', 'operator', '>', 'value', 250000)
    )
  ),
  'warning',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 4: BENEFICIAL OWNERSHIP & RELATED PARTY
-- ============================================================================

-- Rule 4.1: Beneficial Owner Identification (if trust or entity)
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_beneficial_owner_for_entities',
  'business_logic',
  'If identification_type is EIN, beneficial owner must be identified',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'identification_type', 'operator', 'EQUALS', 'value', 'EIN'),
      jsonb_build_object('path', 'workflow_context.beneficial_owner_name', 'operator', 'IS_NULL')
    )
  ),
  'error',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 4.2: Check for Related Party Transactions
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_check_related_parties',
  'business_logic',
  'Flag related party relationships for compliance review',
  ARRAY['clients'],
  jsonb_build_object(
    'operator', 'EXISTS',
    'query', 'SELECT 1 FROM client_contacts WHERE client_id = clients.id AND is_active = true'
  ),
  'info',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 5: DOCUMENT VERIFICATION
-- ============================================================================

-- Rule 5.1: ID Proof Document Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_id_proof_document_required',
  'referential_integrity',
  'Client must have at least one verified ID proof document',
  ARRAY['clients', 'client_documents'],
  jsonb_build_object(
    'operator', 'NOT_EXISTS',
    'query', 'SELECT 1 FROM client_documents WHERE client_id = clients.id AND document_type = ''id_proof'' AND verification_status = ''verified'''
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 5.2: Proof of Address Required
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_proof_of_address_required',
  'referential_integrity',
  'Client must have at least one verified proof of address document',
  ARRAY['clients', 'client_documents'],
  jsonb_build_object(
    'operator', 'NOT_EXISTS',
    'query', 'SELECT 1 FROM client_documents WHERE client_id = clients.id AND document_type = ''proof_of_address'' AND verification_status = ''verified'''
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 5.3: Agreements Must Be Signed
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_agreements_signed',
  'referential_integrity',
  'All required agreements must be e-signed before account creation',
  ARRAY['clients', 'client_documents'],
  jsonb_build_object(
    'operator', 'NOT_EXISTS',
    'query', 'SELECT 1 FROM client_documents WHERE client_id = clients.id AND document_type IN (''client_service_agreement'', ''disclosure_form'') AND e_signature_status = ''signed'''
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 5.4: Document Expiration Check
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'kyc_document_not_expired',
  'business_logic',
  'All verification documents must not be expired',
  ARRAY['client_documents'],
  jsonb_build_object(
    'field', 'is_expired', 'operator', 'EQUALS', 'value', 'true'
  ),
  'warning',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 6: ACCOUNT CREATION RULES
-- ============================================================================

-- Rule 6.1: Account Type Appropriate for Risk Profile
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'account_type_matches_risk_profile',
  'business_logic',
  'Account features (margin, options) must match client risk profile',
  ARRAY['client_accounts'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('path', 'clients.risk_profile', 'operator', 'IN', 'values', jsonb_build_array('"low"', '"conservative"')),
      jsonb_build_object('field', 'allows_margin', 'operator', 'EQUALS', 'value', 'true')
    )
  ),
  'warning',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 6.2: Initial Funding Amount Reasonable
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'account_initial_funding_reasonable',
  'business_logic',
  'Initial account funding should not exceed client net worth by >20%',
  ARRAY['client_accounts'],
  jsonb_build_object(
    'operator', 'AND',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'initial_balance', 'operator', '>', 'value', 'clients.net_worth * 1.2')
    )
  ),
  'warning',
  true,
  false,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- WORKFLOW COMPLETION RULES
-- ============================================================================

-- Rule 7.1: All Steps Must Be Completed
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'workflow_all_steps_completed',
  'cardinality',
  'All 5 onboarding steps must be marked complete before client status = active',
  ARRAY['onboarding_workflows'],
  jsonb_build_object(
    'operator', 'OR',
    'conditions', jsonb_build_array(
      jsonb_build_object('field', 'step_1_validation_status', 'operator', '!=', 'value', 'completed'),
      jsonb_build_object('field', 'step_2_routing_status', 'operator', '!=', 'value', 'completed'),
      jsonb_build_object('field', 'step_3_agreements_status', 'operator', '!=', 'value', 'completed'),
      jsonb_build_object('field', 'step_4_accounts_status', 'operator', '!=', 'value', 'completed'),
      jsonb_build_object('field', 'step_5_notification_status', 'operator', '!=', 'value', 'completed')
    )
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Rule 7.2: No Outstanding Validation Errors
INSERT INTO validation_rules (
  tenant_id, rule_name, rule_type, description, target_entities, condition_json,
  severity, is_active, is_core, created_at, updated_at
) VALUES (
  '00000000-0000-0000-0000-000000000000',
  'workflow_no_outstanding_errors',
  'business_logic',
  'No outstanding validation errors or escalations can be active',
  ARRAY['onboarding_workflows'],
  jsonb_build_object(
    'field', 'validation_errors', 'operator', 'IS_NOT_EMPTY'
  ),
  'error',
  true,
  true,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- SUMMARY OF RULES
-- ============================================================================
-- Total Rules: 20
-- - KYC Requirements: 5
-- - AML Screening: 4
-- - Risk Profile: 3
-- - Beneficial Ownership: 2
-- - Document Verification: 4
-- - Account Creation: 2
-- - Workflow Completion: 2
--
-- These rules can be:
-- - Updated via the Validation Rules UI
-- - Configured with different tenants/datasources
-- - Toggled on/off with is_active
-- - Marked as core rules (is_core=true) that cannot be deleted
-- - Customized with tenant-specific conditions
-- ============================================================================
