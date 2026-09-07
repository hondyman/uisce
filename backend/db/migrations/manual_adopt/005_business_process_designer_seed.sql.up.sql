-- Seed data for Business Process Designer
-- Contains system-level step types, operators, and events

-- Step Types
INSERT INTO process_step_types (key, label, description, icon_svg, default_data, is_system)
VALUES
('initiate',   'Initiate Request',   'Start a new process request', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><circle cx="12" cy="12" r="10"/><path d="M9 12l3-3 3 3"/></svg>', '{"eventId":null}', true),
('validate',   'Validate Data',      'Check rules and constraints on input data', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M9 12l2 2 4-4m7 6A9 9 0 1 1 5 12"/></svg>', '{"eventId":null,"rules":[],"onFailure":"reject"}', true),
('aml',        'AML Screening',      'Anti-Money Laundering compliance check', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 11l3 3 5-5"/></svg>', '{"provider":"lexisnexis","timeout":30}', true),
('approve',    'Route for Approval', 'Send to approver for decision', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm5 7l-6 6-3-3"/></svg>', '{"role":"Advisor","assignee":null}', true),
('generate',   'Generate Docs',      'Create required documents and artifacts', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>', '{"template":"agreement"}', true),
('complete',   'Complete Onboarding','Mark process as successfully completed', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>', '{"notifyClient":true}', true),
('notify',     'Notify Client',      'Send notification to client', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>', '{"channel":"email","template":"default"}', true);

-- Validation Operators
INSERT INTO validation_operators (key, label, description, value_type, is_system)
VALUES
('equals',        'Equals',              'Field equals the specified value',                         'string', true),
('notEquals',     'Not Equals',          'Field does not equal the specified value',                 'string', true),
('greaterThan',   'Greater Than',        'Numeric field is greater than value',                      'number', true),
('lessThan',      'Less Than',           'Numeric field is less than value',                         'number', true),
('greaterOrEqual','Greater or Equal',    'Numeric field is >= value',                                'number', true),
('lessOrEqual',   'Less or Equal',       'Numeric field is <= value',                                'number', true),
('contains',      'Contains',            'String field contains the specified text',                 'string', true),
('notContains',   'Does Not Contain',    'String field does not contain the specified text',         'string', true),
('startsWith',    'Starts With',         'String field starts with the specified value',             'string', true),
('endsWith',      'Ends With',           'String field ends with the specified value',               'string', true),
('inList',        'In List',             'Field value is in the specified list',                     'list', true),
('notInList',     'Not In List',         'Field value is not in the specified list',                 'list', true),
('isEmpty',       'Is Empty',            'Field is empty or null',                                   'string', true),
('isNotEmpty',    'Is Not Empty',        'Field has a value',                                        'string', true),
('regex',         'Matches Regex',       'Field matches the regular expression pattern',             'string', true),
('between',       'Between',             'Numeric field is between two values',                      'number', true),
('isBefore',      'Is Before',           'Date field is before the specified date',                  'date', true),
('isAfter',       'Is After',            'Date field is after the specified date',                   'date', true),
('currencyGt',    'Currency Greater Than', 'Currency field is greater than specified amount',        'currency', true),
('currencyLt',    'Currency Less Than',    'Currency field is less than specified amount',           'currency', true);

-- Workflow Events
INSERT INTO workflow_events (key, label, description, event_type, is_system)
VALUES
('client_app_submitted',   'Client Application Submitted',   'Fired when a prospect completes the intake form',    'on_submit', true),
('client_data_updated',    'Client Data Updated',            'Any field change after onboarding start',            'on_update', true),
('kyc_docs_received',      'KYC Documents Received',         'When KYC supporting documents are uploaded',         'custom', true),
('aml_screening_complete', 'AML Screening Complete',         'After AML check returns results',                    'custom', true),
('approval_requested',     'Approval Requested',             'When the process is routed to an approver',          'custom', true),
('approval_decision',      'Approval Decision',              'When approver approves or rejects',                  'on_approval', true),
('final_approval',         'Final Approval',                 'When final sign-off is granted',                     'on_approval', true),
('onboarding_complete',    'Onboarding Complete',            'When client account is activated',                   'on_completion', true),
('process_error',          'Process Error',                  'When an error or exception occurs',                  'custom', true),
('timeout_triggered',      'Timeout Triggered',              'When a step exceeds its time limit',                 'custom', true);

-- ============================================================================
-- BUSINESS OBJECTS (commonly used in financial services)
-- ============================================================================

-- 1. Client BO
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at, created_by, last_modified_at, last_modified_by)
VALUES (
  'bo-client', 
  'default-tenant',
  'client',
  'Client',
  'Client',
  'client',
  'Core client information',
  'person',
  true,
  now(),
  'system',
  now(),
  'system'
) ON CONFLICT DO NOTHING;

-- Client Fields
INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at, last_modified_by)
SELECT 'bo-client', 'default-tenant', (SELECT id FROM business_objects WHERE key = 'client'), key, name, label, name, type, true, false, '', seq, now(), 'system', now(), 'system'
FROM (VALUES
  ('id', 'Client ID', 'string', 1),
  ('first_name', 'First Name', 'string', 2),
  ('last_name', 'Last Name', 'string', 3),
  ('email', 'Email Address', 'email', 4),
  ('phone', 'Phone Number', 'text', 5),
  ('date_of_birth', 'Date of Birth', 'date', 6),
  ('net_worth', 'Net Worth', 'currency', 7),
  ('country', 'Country of Residence', 'text', 8),
  ('accredited_investor', 'Accredited Investor Status', 'boolean', 9),
  ('kyc_status', 'KYC Status', 'text', 10),
  ('aml_status', 'AML Status', 'text', 11)
) AS t(key, label, type, seq)
ON CONFLICT DO NOTHING;

-- 2. Account BO
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at, created_by, last_modified_at, last_modified_by)
VALUES (
  'bo-account',
  'default-tenant',
  'account',
  'Account',
  'Account',
  'account',
  'Client account information',
  'credit-card',
  true,
  now(),
  'system',
  now(),
  'system'
) ON CONFLICT DO NOTHING;

-- Account Fields
INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at, last_modified_by)
SELECT 'bo-account', 'default-tenant', (SELECT id FROM business_objects WHERE key = 'account'), key, name, label, name, type, true, false, '', seq, now(), 'system', now(), 'system'
FROM (VALUES
  ('id', 'Account ID', 'text', 1),
  ('account_number', 'Account Number', 'text', 2),
  ('account_type', 'Account Type', 'text', 3),
  ('status', 'Account Status', 'text', 4),
  ('balance', 'Account Balance', 'currency', 5),
  ('created_date', 'Created Date', 'date', 6),
  ('primary_contact', 'Primary Contact', 'text', 7),
  ('approval_date', 'Approval Date', 'date', 8)
) AS t(key, label, type, seq)
ON CONFLICT DO NOTHING;

-- 3. Transaction BO
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at, created_by, last_modified_at, last_modified_by)
VALUES (
  'bo-transaction',
  'default-tenant',
  'transaction',
  'Transaction',
  'Transaction',
  'transaction',
  'Transaction details',
  'arrow-right-left',
  true,
  now(),
  'system',
  now(),
  'system'
) ON CONFLICT DO NOTHING;

-- Transaction Fields
INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at, last_modified_by)
SELECT 'bo-transaction', 'default-tenant', (SELECT id FROM business_objects WHERE key = 'transaction'), key, name, label, name, type, true, false, '', seq, now(), 'system', now(), 'system'
FROM (VALUES
  ('id', 'Transaction ID', 'text', 1),
  ('amount', 'Amount', 'currency', 2),
  ('currency', 'Currency', 'text', 3),
  ('type', 'Transaction Type', 'text', 4),
  ('status', 'Status', 'text', 5),
  ('created_date', 'Created Date', 'date', 6),
  ('description', 'Description', 'text', 7)
) AS t(key, label, type, seq)
ON CONFLICT DO NOTHING;

-- 4. Document BO
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at, created_by, last_modified_at, last_modified_by)
VALUES (
  'bo-document',
  'default-tenant',
  'document',
  'Document',
  'Document',
  'document',
  'Document submission tracking',
  'file',
  true,
  now(),
  'system',
  now(),
  'system'
) ON CONFLICT DO NOTHING;

-- Document Fields
INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at, last_modified_by)
SELECT 'bo-document', 'default-tenant', (SELECT id FROM business_objects WHERE key = 'document'), key, name, label, name, type, true, false, '', seq, now(), 'system', now(), 'system'
FROM (VALUES
  ('id', 'Document ID', 'text', 1),
  ('type', 'Document Type', 'text', 2),
  ('status', 'Upload Status', 'text', 3),
  ('file_name', 'File Name', 'text', 4),
  ('uploaded_date', 'Upload Date', 'date', 5),
  ('verified', 'Verified', 'boolean', 6),
  ('expiry_date', 'Expiry Date', 'date', 7)
) AS t(key, label, type, seq)
ON CONFLICT DO NOTHING;
