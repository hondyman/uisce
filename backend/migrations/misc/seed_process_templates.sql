-- ============================================================================
-- Seed Process Templates Catalog
-- Run with: psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/migrations/misc/seed_process_templates.sql
-- ============================================================================

-- First, insert template categories
INSERT INTO template_categories (id, category_key, display_name, description, icon_name, sort_order, is_active) VALUES
('a1111111-1111-1111-1111-111111111111', 'approval', 'Approval Workflows', 'Multi-level approval processes with escalation and routing', 'CheckCircle', 1, true),
('a2222222-2222-2222-2222-222222222222', 'data_collection', 'Data Collection', 'Form-based data entry and collection workflows', 'FileText', 2, true),
('a3333333-3333-3333-3333-333333333333', 'review', 'Review Processes', 'Document and code review workflows with approval chains', 'AlertCircle', 3, true),
('a4444444-4444-4444-4444-444444444444', 'onboarding', 'Onboarding', 'Employee and vendor onboarding checklists and workflows', 'Users', 4, true),
('a5555555-5555-5555-5555-555555555555', 'compliance', 'Compliance', 'Audit and compliance verification workflows', 'AlertTriangle', 5, true),
('a6666666-6666-6666-6666-666666666666', 'automation', 'Automation', 'Automated task execution and integration workflows', 'TrendingUp', 6, true),
('a7777777-7777-7777-7777-777777777777', 'notification', 'Notifications', 'Alert and notification distribution workflows', 'Clock', 7, true),
('a8888888-8888-8888-8888-888888888888', 'other', 'Other', 'Miscellaneous workflow templates', 'Package', 8, true)
ON CONFLICT (category_key) DO UPDATE SET 
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  icon_name = EXCLUDED.icon_name,
  sort_order = EXCLUDED.sort_order,
  is_active = EXCLUDED.is_active;

-- ============================================================================
-- APPROVAL WORKFLOW TEMPLATES (3)
-- ============================================================================

-- 1. Simple Manager Approval
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name, 
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'b1111111-1111-1111-1111-111111111111',
  'simple-manager-approval',
  'Simple Manager Approval',
  'Basic two-step approval workflow: submit request, manager approves, process completes. Perfect for simple approval scenarios.',
  'approval',
  ARRAY['approval', 'manager', 'simple', 'beginner', '2-step'],
  'CheckCircle',
  'beginner',
  15,
  true,
  true,
  '{"processName":"Manager Approval","entity":"request","description":"Submit request for manager approval","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Submit Request","stepDescription":"Requester submits request with details","durationHours":0.5,"assigneeRole":"requester","validationRules":[{"field":"amount","operator":"required","value":"","errorMessage":"Amount is required"},{"field":"description","operator":"required","value":"","errorMessage":"Description is required"}],"formFields":[{"fieldName":"amount","fieldType":"number","isRequired":true},{"fieldName":"description","fieldType":"textarea","isRequired":true},{"fieldName":"justification","fieldType":"textarea","isRequired":false}]},{"id":"step-2","stepOrder":2,"stepType":"approve","stepName":"Manager Approval","stepDescription":"Manager reviews and approves or rejects request","durationHours":24,"assigneeRole":"manager","notificationTemplate":"manager_approval_required","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"manager","order":1}],"onReject":"terminate","onApprove":"next_step"}},{"id":"step-3","stepOrder":3,"stepType":"notify","stepName":"Notify Requester","stepDescription":"Send notification of approval decision","durationHours":0.1,"notificationTemplate":"approval_decision","notificationConfig":{"recipients":["requester"],"template":"approval_complete","includeAttachments":false}}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Update Roles**: Change "requester" and "manager" roles to match your organization\n2. **Adjust Duration**: Default approval time is 24 hours - adjust as needed\n3. **Add Validation**: Add custom validation rules for your specific fields\n4. **Configure Notifications**: Set up email/SMS notification templates\n5. **Test Workflow**: Run a test approval with sample data\n\n## Tips\n\n- Start with the default 2-step flow and add complexity later\n- Consider adding an escalation path if approval takes too long\n- Add conditional logic for different approval thresholds',
  ARRAY[
    'Purchase requests under $1000',
    'Time-off requests',
    'Access requests for systems',
    'Document approvals',
    'Equipment checkout requests'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.7,
  156,
  2847,
  1203,
  427,
  'approval manager simple two-step basic request purchase timeoff',
  NOW()
);

-- 2. Multi-Level Approval
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'b2222222-2222-2222-2222-222222222222',
  'multi-level-approval',
  'Multi-Level Approval Chain',
  'Three-tier approval workflow with escalation: team lead → department manager → executive. Includes parallel approval and escalation paths.',
  'approval',
  ARRAY['approval', 'multi-level', 'escalation', 'intermediate', 'hierarchy'],
  'CheckCircle',
  'intermediate',
  30,
  true,
  true,
  '{"processName":"Multi-Level Approval","entity":"request","description":"Multi-tier approval with escalation","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Submit Request","stepDescription":"Employee submits request for approval","durationHours":1,"assigneeRole":"employee","validationRules":[{"field":"amount","operator":"required"},{"field":"category","operator":"required"}],"formFields":[{"fieldName":"amount","fieldType":"number","isRequired":true},{"fieldName":"category","fieldType":"select","options":["Capital","Operating","Travel","Training"],"isRequired":true},{"fieldName":"business_justification","fieldType":"textarea","isRequired":true}]},{"id":"step-2","stepOrder":2,"stepType":"approve","stepName":"Team Lead Approval","stepDescription":"First level approval by team lead","durationHours":24,"assigneeRole":"team_lead","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"team_lead","order":1}],"onReject":"terminate","onApprove":"next_step","escalationHours":48,"escalateTo":"department_manager"}},{"id":"step-3","stepOrder":3,"stepType":"approve","stepName":"Department Manager Approval","stepDescription":"Second level approval by department manager","durationHours":48,"assigneeRole":"department_manager","conditionalBranch":{"field":"amount","operator":">=","value":"5000","trueNextStep":"step-4","falseNextStep":"step-5"},"approvalChain":{"requiredApprovals":1,"approvers":[{"role":"department_manager","order":1}],"onReject":"terminate","onApprove":"next_step"}},{"id":"step-4","stepOrder":4,"stepType":"approve","stepName":"Executive Approval","stepDescription":"Third level approval for high-value requests","durationHours":72,"assigneeRole":"executive","conditionalBranch":{"field":"amount","operator":">=","value":"5000","trueNextStep":"step-5","falseNextStep":"skip"},"approvalChain":{"requiredApprovals":1,"approvers":[{"role":"executive","order":1}],"onReject":"terminate","onApprove":"next_step"}},{"id":"step-5","stepOrder":5,"stepType":"notify","stepName":"Notify All Parties","stepDescription":"Send approval decision to all stakeholders","durationHours":0.1,"notificationTemplate":"approval_complete","notificationConfig":{"recipients":["employee","team_lead","department_manager"],"includeExecutive":true}}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Define Approval Levels**: Map team_lead, department_manager, executive to your org structure\n2. **Set Thresholds**: Adjust the $5000 threshold for executive approval\n3. **Configure Escalation**: Set escalation timelines (default 48 hours for team lead)\n4. **Add Parallel Approval**: Enable multiple approvers at each level if needed\n5. **Customize Notifications**: Set up escalation notification templates\n\n## Advanced Features\n\n- Amount-based routing: Requests over $5000 require executive approval\n- Auto-escalation: If team lead doesn''t respond in 48 hours, escalates to manager\n- Parallel approvals: Enable 2-of-3 approval patterns\n- Rejection handling: Configure what happens when any level rejects',
  ARRAY[
    'Capital expenditure requests',
    'Budget approval workflows',
    'Contract approvals',
    'Large purchase orders',
    'Hiring requisitions'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.8,
  243,
  1876,
  892,
  312,
  'approval multi-level hierarchy escalation three-tier budget capital',
  NOW()
);

-- 3. Conditional Approval Router
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'b3333333-3333-3333-3333-333333333333',
  'conditional-approval-router',
  'Conditional Approval Router',
  'Smart approval routing based on request attributes: amount, category, risk level. Automatically routes to appropriate approver based on business rules.',
  'approval',
  ARRAY['approval', 'conditional', 'routing', 'advanced', 'rules-based'],
  'GitBranch',
  'advanced',
  45,
  true,
  false,
  '{"processName":"Conditional Approval","entity":"request","description":"Smart routing based on request attributes","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Submit Request","stepDescription":"Employee provides request details","durationHours":0.5,"assigneeRole":"employee","formFields":[{"fieldName":"amount","fieldType":"number","isRequired":true},{"fieldName":"category","fieldType":"select","options":["Standard","High-Risk","Capital","Recurring"],"isRequired":true},{"fieldName":"priority","fieldType":"select","options":["Low","Medium","High","Critical"],"isRequired":true},{"fieldName":"department","fieldType":"select","options":["Sales","Marketing","Engineering","Operations","Finance"],"isRequired":true}]},{"id":"step-2","stepOrder":2,"stepType":"automated","stepName":"Route Request","stepDescription":"Automatically determine approval path","durationHours":0,"conditionalBranch":{"conditions":[{"field":"category","operator":"==","value":"High-Risk","nextStep":"step-3"},{"field":"amount","operator":">=","value":"10000","nextStep":"step-4"},{"field":"priority","operator":"==","value":"Critical","nextStep":"step-5"}],"defaultNextStep":"step-6"}},{"id":"step-3","stepOrder":3,"stepType":"approve","stepName":"Risk Committee Review","stepDescription":"High-risk requests reviewed by committee","durationHours":72,"assigneeRole":"risk_committee","approvalChain":{"requiredApprovals":2,"approvers":[{"role":"risk_officer","order":1},{"role":"compliance_officer","order":1}],"approvalType":"parallel"}},{"id":"step-4","stepOrder":4,"stepType":"approve","stepName":"Finance Approval","stepDescription":"High-value requests approved by finance","durationHours":48,"assigneeRole":"finance_director","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"finance_director","order":1}]}},{"id":"step-5","stepOrder":5,"stepType":"approve","stepName":"Expedited Approval","stepDescription":"Critical priority fast-tracked","durationHours":4,"assigneeRole":"executive","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"executive","order":1}]}},{"id":"step-6","stepOrder":6,"stepType":"approve","stepName":"Standard Approval","stepDescription":"Regular approval by department manager","durationHours":24,"assigneeRole":"department_manager"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Define Routing Rules**: Configure conditions for each approval path\n2. **Set Thresholds**: Adjust amount/category/priority routing logic\n3. **Map Approvers**: Assign risk_committee, finance_director, etc. to actual users\n4. **Test Scenarios**: Create test cases for each routing path\n5. **Add More Paths**: Add additional routing conditions as needed\n\n## Routing Logic\n\n- **High-Risk Category** → Risk Committee (parallel approval required)\n- **Amount ≥ $10,000** → Finance Director approval\n- **Critical Priority** → Expedited executive approval (4 hours)\n- **Default** → Standard department manager approval\n\n## Advanced Configuration\n\n- Add time-based routing (end-of-quarter, fiscal year)\n- Combine multiple conditions with AND/OR logic\n- Add fallback approvers for out-of-office scenarios',
  ARRAY[
    'Complex approval scenarios',
    'Risk-based approval routing',
    'Finance approval workflows',
    'Multi-criteria routing',
    'Compliance-heavy processes'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.6,
  89,
  567,
  298,
  156,
  'approval conditional routing smart rules-based risk finance',
  NOW()
);

-- ============================================================================
-- DATA COLLECTION TEMPLATES (3)
-- ============================================================================

-- 4. Employee Onboarding Form
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'c1111111-1111-1111-1111-111111111111',
  'employee-onboarding-form',
  'Employee Onboarding Data Collection',
  'Comprehensive new hire data collection: personal info, tax forms, benefits selection, emergency contacts. Multi-step form with document uploads.',
  'data_collection',
  ARRAY['onboarding', 'hr', 'forms', 'beginner', 'documents'],
  'FileText',
  'beginner',
  20,
  true,
  true,
  '{"processName":"Employee Onboarding","entity":"employee","description":"New hire information collection","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Personal Information","stepDescription":"Collect basic employee details","durationHours":1,"assigneeRole":"new_hire","formFields":[{"fieldName":"full_name","fieldType":"text","isRequired":true},{"fieldName":"email","fieldType":"email","isRequired":true},{"fieldName":"phone","fieldType":"tel","isRequired":true},{"fieldName":"address","fieldType":"textarea","isRequired":true},{"fieldName":"date_of_birth","fieldType":"date","isRequired":true},{"fieldName":"start_date","fieldType":"date","isRequired":true}]},{"id":"step-2","stepOrder":2,"stepType":"data_entry","stepName":"Tax & Legal Documents","stepDescription":"Upload required legal forms","durationHours":2,"assigneeRole":"new_hire","formFields":[{"fieldName":"w4_form","fieldType":"file","isRequired":true},{"fieldName":"i9_form","fieldType":"file","isRequired":true},{"fieldName":"direct_deposit","fieldType":"file","isRequired":true},{"fieldName":"signed_offer_letter","fieldType":"file","isRequired":true}]},{"id":"step-3","stepOrder":3,"stepType":"data_entry","stepName":"Benefits Selection","stepDescription":"Choose health insurance and benefits","durationHours":4,"assigneeRole":"new_hire","formFields":[{"fieldName":"health_insurance_plan","fieldType":"select","options":["PPO","HMO","High-Deductible","Decline"],"isRequired":true},{"fieldName":"dental_insurance","fieldType":"select","options":["Yes","No"],"isRequired":true},{"fieldName":"vision_insurance","fieldType":"select","options":["Yes","No"],"isRequired":true},{"fieldName":"401k_contribution_percent","fieldType":"number","isRequired":false}]},{"id":"step-4","stepOrder":4,"stepType":"data_entry","stepName":"Emergency Contacts","stepDescription":"Provide emergency contact information","durationHours":0.5,"assigneeRole":"new_hire","formFields":[{"fieldName":"emergency_contact_name","fieldType":"text","isRequired":true},{"fieldName":"emergency_contact_phone","fieldType":"tel","isRequired":true},{"fieldName":"emergency_contact_relationship","fieldType":"text","isRequired":true}]},{"id":"step-5","stepOrder":5,"stepType":"review","stepName":"HR Review","stepDescription":"HR verifies all submitted information","durationHours":24,"assigneeRole":"hr_coordinator","notificationTemplate":"onboarding_complete"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Adjust Form Fields**: Add/remove fields based on your requirements\n2. **Configure Document Types**: Specify which documents are required\n3. **Set Up Benefits Options**: Update health insurance and benefits choices\n4. **Add Validation**: Add business rules for form validation\n5. **Configure Notifications**: Set up welcome emails and reminders\n\n## Tips\n\n- Break into smaller steps if the form is too long\n- Add progress indicators for multi-step forms\n- Enable save-and-resume functionality\n- Send reminder emails if form is incomplete',
  ARRAY[
    'New hire onboarding',
    'HR data collection',
    'Employee information forms',
    'Benefits enrollment',
    'Tax document collection'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.9,
  412,
  3247,
  1689,
  892,
  'onboarding hr employee new-hire forms data-collection benefits',
  NOW()
);

-- 5. Customer Registration
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'c2222222-2222-2222-2222-222222222222',
  'customer-registration',
  'Customer Registration & KYC',
  'Customer onboarding with identity verification: collect contact info, verify identity, perform KYC checks, approve account.',
  'data_collection',
  ARRAY['customer', 'kyc', 'verification', 'intermediate', 'compliance'],
  'Users',
  'intermediate',
  35,
  true,
  false,
  '{"processName":"Customer Registration","entity":"customer","description":"Customer onboarding with KYC","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Account Information","stepDescription":"Customer provides basic information","durationHours":0.5,"assigneeRole":"customer","formFields":[{"fieldName":"company_name","fieldType":"text","isRequired":true},{"fieldName":"contact_name","fieldType":"text","isRequired":true},{"fieldName":"email","fieldType":"email","isRequired":true},{"fieldName":"phone","fieldType":"tel","isRequired":true},{"fieldName":"country","fieldType":"select","isRequired":true},{"fieldName":"industry","fieldType":"select","isRequired":true}]},{"id":"step-2","stepOrder":2,"stepType":"data_entry","stepName":"Identity Verification","stepDescription":"Upload identity documents","durationHours":1,"assigneeRole":"customer","formFields":[{"fieldName":"business_license","fieldType":"file","isRequired":true},{"fieldName":"tax_id_number","fieldType":"text","isRequired":true},{"fieldName":"proof_of_address","fieldType":"file","isRequired":true},{"fieldName":"authorized_signatory_id","fieldType":"file","isRequired":true}]},{"id":"step-3","stepOrder":3,"stepType":"automated","stepName":"Automated KYC Check","stepDescription":"Run automated compliance checks","durationHours":0.1,"notificationTemplate":"kyc_check_initiated"},{"id":"step-4","stepOrder":4,"stepType":"review","stepName":"Compliance Review","stepDescription":"Manual review of KYC results","durationHours":48,"assigneeRole":"compliance_officer","conditionalBranch":{"field":"kyc_status","operator":"==","value":"flagged","trueNextStep":"step-5","falseNextStep":"step-6"}},{"id":"step-5","stepOrder":5,"stepType":"approve","stepName":"Enhanced Due Diligence","stepDescription":"Additional review for flagged accounts","durationHours":120,"assigneeRole":"compliance_manager","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"compliance_manager","order":1}]}},{"id":"step-6","stepOrder":6,"stepType":"automated","stepName":"Activate Account","stepDescription":"Provision customer account","durationHours":0.1,"notificationTemplate":"account_activated"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Configure KYC Rules**: Set up automated compliance check integration\n2. **Customize Document Requirements**: Adjust based on jurisdiction\n3. **Set Review Thresholds**: Define when enhanced due diligence is required\n4. **Add Risk Scoring**: Integrate risk scoring for automated flagging\n5. **Configure Notifications**: Set up account activation emails\n\n## Compliance Features\n\n- Automated AML/KYC screening\n- Document verification and storage\n- Risk-based review routing\n- Enhanced due diligence for high-risk customers\n- Audit trail for compliance reporting',
  ARRAY[
    'Customer onboarding',
    'KYC verification',
    'Identity verification',
    'Compliance workflows',
    'Account opening'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.5,
  187,
  1432,
  743,
  289,
  'customer kyc verification compliance onboarding identity',
  NOW()
);

-- 6. Survey & Feedback Collection
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'c3333333-3333-3333-3333-333333333333',
  'survey-feedback',
  'Survey & Feedback Collection',
  'Customizable survey workflow with branching questions, skip logic, and automated analysis. Collect NPS scores, satisfaction ratings, and open feedback.',
  'data_collection',
  ARRAY['survey', 'feedback', 'nps', 'beginner', 'analytics'],
  'FileText',
  'beginner',
  25,
  true,
  true,
  '{"processName":"Customer Feedback Survey","entity":"response","description":"Collect and analyze customer feedback","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Introduction","stepDescription":"Survey introduction and consent","durationHours":0.1,"assigneeRole":"respondent","formFields":[{"fieldName":"consent","fieldType":"checkbox","isRequired":true,"label":"I agree to participate in this survey"}]},{"id":"step-2","stepOrder":2,"stepType":"data_entry","stepName":"NPS Score","stepDescription":"Net Promoter Score question","durationHours":0.1,"assigneeRole":"respondent","formFields":[{"fieldName":"nps_score","fieldType":"number","min":0,"max":10,"isRequired":true,"label":"How likely are you to recommend us? (0-10)"}]},{"id":"step-3","stepOrder":3,"stepType":"data_entry","stepName":"Satisfaction Ratings","stepDescription":"Rate different aspects","durationHours":0.2,"assigneeRole":"respondent","formFields":[{"fieldName":"product_quality","fieldType":"select","options":["Very Dissatisfied","Dissatisfied","Neutral","Satisfied","Very Satisfied"],"isRequired":true},{"fieldName":"customer_service","fieldType":"select","options":["Very Dissatisfied","Dissatisfied","Neutral","Satisfied","Very Satisfied"],"isRequired":true},{"fieldName":"value_for_money","fieldType":"select","options":["Very Dissatisfied","Dissatisfied","Neutral","Satisfied","Very Satisfied"],"isRequired":true}]},{"id":"step-4","stepOrder":4,"stepType":"data_entry","stepName":"Open Feedback","stepDescription":"Additional comments","durationHours":0.2,"assigneeRole":"respondent","formFields":[{"fieldName":"what_we_do_well","fieldType":"textarea","isRequired":false,"label":"What do we do well?"},{"fieldName":"areas_for_improvement","fieldType":"textarea","isRequired":false,"label":"What could we improve?"},{"fieldName":"feature_requests","fieldType":"textarea","isRequired":false,"label":"Any feature requests?"}]},{"id":"step-5","stepOrder":5,"stepType":"automated","stepName":"Analyze Results","stepDescription":"Aggregate and categorize responses","durationHours":0,"notificationTemplate":"survey_complete"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Customize Questions**: Adjust survey questions for your needs\n2. **Add Skip Logic**: Route to different questions based on answers\n3. **Configure NPS Calculation**: Set up automated NPS scoring\n4. **Add Analytics**: Enable real-time response analytics\n5. **Set Up Follow-Up**: Trigger follow-up actions for low scores\n\n## Features\n\n- NPS score tracking\n- Multi-dimensional satisfaction ratings\n- Open-ended feedback collection\n- Anonymous or identified responses\n- Automated response aggregation\n- Follow-up workflows for detractors',
  ARRAY[
    'Customer satisfaction surveys',
    'NPS surveys',
    'Employee engagement surveys',
    'Product feedback',
    'Exit interviews'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.4,
  276,
  2134,
  967,
  534,
  'survey feedback nps satisfaction customer rating',
  NOW()
);

-- ============================================================================
-- REVIEW PROCESS TEMPLATES (2)
-- ============================================================================

-- 7. Document Review & Approval
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'd1111111-1111-1111-1111-111111111111',
  'document-review',
  'Document Review & Approval',
  'Structured document review process: upload → peer review → approval → archive. Includes version control and comment tracking.',
  'review',
  ARRAY['review', 'document', 'approval', 'intermediate', 'versioning'],
  'FileText',
  'intermediate',
  30,
  true,
  false,
  '{"processName":"Document Review","entity":"document","description":"Review and approve documents","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Upload Document","stepDescription":"Author uploads document for review","durationHours":0.5,"assigneeRole":"author","formFields":[{"fieldName":"document_title","fieldType":"text","isRequired":true},{"fieldName":"document_file","fieldType":"file","isRequired":true},{"fieldName":"document_type","fieldType":"select","options":["Policy","Procedure","Contract","Proposal","Report"],"isRequired":true},{"fieldName":"version_number","fieldType":"text","isRequired":true},{"fieldName":"summary","fieldType":"textarea","isRequired":true}]},{"id":"step-2","stepOrder":2,"stepType":"review","stepName":"Peer Review","stepDescription":"Assigned reviewers provide feedback","durationHours":48,"assigneeRole":"peer_reviewer","formFields":[{"fieldName":"review_comments","fieldType":"textarea","isRequired":true},{"fieldName":"suggested_changes","fieldType":"textarea","isRequired":false},{"fieldName":"review_status","fieldType":"select","options":["Approved","Needs Minor Revisions","Needs Major Revisions","Rejected"],"isRequired":true}]},{"id":"step-3","stepOrder":3,"stepType":"conditional","stepName":"Check Review Status","conditionalBranch":{"field":"review_status","operator":"==","value":"Needs Minor Revisions","trueNextStep":"step-4","falseNextStep":"step-5"}},{"id":"step-4","stepOrder":4,"stepType":"data_entry","stepName":"Revise Document","stepDescription":"Author makes revisions","durationHours":24,"assigneeRole":"author","formFields":[{"fieldName":"revised_document","fieldType":"file","isRequired":true},{"fieldName":"revision_notes","fieldType":"textarea","isRequired":true}]},{"id":"step-5","stepOrder":5,"stepType":"approve","stepName":"Final Approval","stepDescription":"Manager provides final approval","durationHours":48,"assigneeRole":"manager","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"manager","order":1}]}},{"id":"step-6","stepOrder":6,"stepType":"automated","stepName":"Archive Document","stepDescription":"Store approved document","durationHours":0,"notificationTemplate":"document_approved"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Define Document Types**: Add your specific document categories\n2. **Set Reviewer Roles**: Assign peer_reviewer and manager roles\n3. **Configure Revision Logic**: Set rules for when revisions are needed\n4. **Add Version Control**: Enable automatic version numbering\n5. **Set Up Archive**: Configure document storage location\n\n## Features\n\n- Multi-reviewer support\n- Revision cycles with version tracking\n- Comment and feedback collection\n- Conditional approval based on review results\n- Automated archiving of approved documents',
  ARRAY[
    'Policy review',
    'Contract review',
    'Document approval',
    'Content review',
    'Legal document review'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.6,
  198,
  1543,
  821,
  376,
  'document review approval policy contract versioning',
  NOW()
);

-- 8. Code Review Process
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'd2222222-2222-2222-2222-222222222222',
  'code-review',
  'Code Review & Merge Process',
  'Engineering code review workflow: PR submission → automated tests → peer review → approval → merge. Integrates with CI/CD pipelines.',
  'review',
  ARRAY['code-review', 'engineering', 'github', 'advanced', 'automation'],
  'Code',
  'advanced',
  40,
  true,
  false,
  '{"processName":"Code Review","entity":"pull_request","description":"Review and merge code changes","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"data_entry","stepName":"Submit Pull Request","stepDescription":"Developer submits PR for review","durationHours":0.5,"assigneeRole":"developer","formFields":[{"fieldName":"pr_title","fieldType":"text","isRequired":true},{"fieldName":"pr_description","fieldType":"textarea","isRequired":true},{"fieldName":"github_pr_url","fieldType":"url","isRequired":true},{"fieldName":"ticket_numbers","fieldType":"text","isRequired":false}]},{"id":"step-2","stepOrder":2,"stepType":"automated","stepName":"Run Automated Tests","stepDescription":"CI/CD pipeline runs tests","durationHours":0.5,"notificationTemplate":"tests_running","conditionalBranch":{"field":"tests_passed","operator":"==","value":"false","trueNextStep":"step-7","falseNextStep":"step-3"}},{"id":"step-3","stepOrder":3,"stepType":"review","stepName":"Code Review","stepDescription":"Peers review code changes","durationHours":24,"assigneeRole":"peer_developer","formFields":[{"fieldName":"code_quality_score","fieldType":"select","options":["1","2","3","4","5"],"isRequired":true},{"fieldName":"review_comments","fieldType":"textarea","isRequired":false},{"fieldName":"security_concerns","fieldType":"textarea","isRequired":false},{"fieldName":"recommendation","fieldType":"select","options":["Approve","Request Changes","Reject"],"isRequired":true}]},{"id":"step-4","stepOrder":4,"stepType":"conditional","stepName":"Check Review Result","conditionalBranch":{"field":"recommendation","operator":"==","value":"Request Changes","trueNextStep":"step-5","falseNextStep":"step-6"}},{"id":"step-5","stepOrder":5,"stepType":"data_entry","stepName":"Address Feedback","stepDescription":"Developer makes requested changes","durationHours":8,"assigneeRole":"developer"},{"id":"step-6","stepOrder":6,"stepType":"approve","stepName":"Tech Lead Approval","stepDescription":"Final approval before merge","durationHours":4,"assigneeRole":"tech_lead","approvalChain":{"requiredApprovals":1,"approvers":[{"role":"tech_lead","order":1}]}},{"id":"step-7","stepOrder":7,"stepType":"automated","stepName":"Merge to Main","stepDescription":"Merge approved code","durationHours":0,"notificationTemplate":"pr_merged"},{"id":"step-8","stepOrder":8,"stepType":"notify","stepName":"Failed Tests","stepDescription":"Notify developer of test failures","durationHours":0,"notificationTemplate":"tests_failed"}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Integrate CI/CD**: Connect to your GitHub/GitLab/Bitbucket\n2. **Define Code Standards**: Set quality criteria for reviews\n3. **Configure Auto-Tests**: Set up automated test triggers\n4. **Set Review Requirements**: Specify number of approvals needed\n5. **Add Security Checks**: Enable automated security scanning\n\n## Engineering Best Practices\n\n- Automated test execution\n- Multiple reviewer support (configurable)\n- Security and code quality checks\n- Rejection and revision cycles\n- Integration with version control systems\n- Automated merge after approval',
  ARRAY[
    'Code review',
    'Pull request workflow',
    'Engineering process',
    'CI/CD integration',
    'GitHub workflow'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.7,
  321,
  1867,
  943,
  401,
  'code review pull-request engineering github ci-cd merge',
  NOW()
);

-- ============================================================================
-- AUTOMATION TEMPLATES (2)
-- ============================================================================

-- 9. Scheduled Report Generation
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'e1111111-1111-1111-1111-111111111111',
  'scheduled-reports',
  'Scheduled Report Generation',
  'Automated report generation and distribution: query data → generate report → email to stakeholders. Runs on daily/weekly/monthly schedule.',
  'automation',
  ARRAY['automation', 'reports', 'scheduled', 'intermediate', 'analytics'],
  'TrendingUp',
  'intermediate',
  30,
  true,
  false,
  '{"processName":"Scheduled Reports","entity":"report","description":"Generate and distribute reports automatically","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"automated","stepName":"Query Data","stepDescription":"Extract data from sources","durationHours":0.5},{"id":"step-2","stepOrder":2,"stepType":"automated","stepName":"Generate Report","stepDescription":"Format data into report","durationHours":0.25},{"id":"step-3","stepOrder":3,"stepType":"automated","stepName":"Email Report","stepDescription":"Distribute to stakeholders","durationHours":0,"notificationConfig":{"recipients":["executives","department_heads"],"includeAttachments":true}}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Configure Data Sources**: Connect to your databases/APIs\n2. **Set Schedule**: Daily, weekly, monthly, or custom schedule\n3. **Design Report Template**: Create report format (PDF, Excel, HTML)\n4. **Define Recipients**: Set up distribution lists\n5. **Add Conditional Logic**: Only send if data meets criteria\n\n## Automation Features\n\n- Scheduled execution (cron-based)\n- Multi-source data aggregation\n- Template-based report generation\n- Conditional delivery rules\n- Archive historical reports',
  ARRAY[
    'Daily reports',
    'Weekly metrics',
    'Monthly dashboards',
    'Executive summaries',
    'Analytics distribution'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.5,
  167,
  1234,
  612,
  289,
  'automation scheduled reports analytics email distribution',
  NOW()
);

-- 10. Alert Escalation
INSERT INTO process_templates (
  id, template_key, name, description, category, tags, icon_name,
  difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
  template_definition, customization_guide, example_use_cases,
  author_name, author_organization, version,
  rating_average, rating_count, usage_count, clone_count, favorite_count,
  search_keywords, published_at
) VALUES (
  'e2222222-2222-2222-2222-222222222222',
  'alert-escalation',
  'Alert Escalation Workflow',
  'Progressive alert escalation with timeouts: notify on-call → escalate to manager → escalate to director. Integrates with PagerDuty/Slack.',
  'notification',
  ARRAY['alerts', 'escalation', 'incident', 'advanced', 'on-call'],
  'AlertTriangle',
  'advanced',
  45,
  true,
  true,
  '{"processName":"Alert Escalation","entity":"alert","description":"Escalate alerts through team hierarchy","isActive":true,"steps":[{"id":"step-1","stepOrder":1,"stepType":"notify","stepName":"Alert On-Call","stepDescription":"Notify primary on-call engineer","durationHours":0,"notificationConfig":{"recipients":["on_call_primary"],"channels":["sms","phone","slack"],"priority":"high"}},{"id":"step-2","stepOrder":2,"stepType":"conditional","stepName":"Wait for Acknowledgment","durationHours":0.25,"conditionalBranch":{"field":"acknowledged","operator":"==","value":"false","trueNextStep":"step-3","falseNextStep":"step-6"}},{"id":"step-3","stepOrder":3,"stepType":"notify","stepName":"Escalate to Manager","stepDescription":"Alert engineering manager","durationHours":0,"notificationConfig":{"recipients":["engineering_manager"],"channels":["sms","phone","slack"],"priority":"critical"}},{"id":"step-4","stepOrder":4,"stepType":"conditional","stepName":"Wait for Manager Response","durationHours":0.5,"conditionalBranch":{"field":"acknowledged","operator":"==","value":"false","trueNextStep":"step-5","falseNextStep":"step-6"}},{"id":"step-5","stepOrder":5,"stepType":"notify","stepName":"Escalate to Director","stepDescription":"Alert VP/Director","durationHours":0,"notificationConfig":{"recipients":["engineering_director"],"channels":["sms","phone"],"priority":"critical"}},{"id":"step-6","stepOrder":6,"stepType":"data_entry","stepName":"Record Resolution","assigneeRole":"responder","formFields":[{"fieldName":"root_cause","fieldType":"textarea"},{"fieldName":"resolution_actions","fieldType":"textarea"},{"fieldName":"time_to_resolution","fieldType":"number"}]}],"version":"1.0.0"}'::jsonb,
  E'## Customization Steps\n\n1. **Define Escalation Path**: Set up on_call_primary → manager → director\n2. **Set Timeouts**: Configure wait times before escalation (default 15/30 min)\n3. **Configure Channels**: Enable SMS, phone, Slack, PagerDuty\n4. **Add Alert Filters**: Only escalate critical severity alerts\n5. **Set Business Hours**: Different escalation for off-hours\n\n## Incident Management\n\n- Progressive escalation with timeouts\n- Multi-channel notifications (SMS, phone, Slack)\n- Acknowledgment tracking\n- Resolution documentation\n- Post-mortem workflow',
  ARRAY[
    'Incident management',
    'On-call escalation',
    'Alert routing',
    'DevOps workflows',
    'SRE processes'
  ],
  'Fabric Builder Team',
  'Semlayer',
  '1.0.0',
  4.8,
  234,
  1567,
  798,
  421,
  'alert escalation incident on-call devops pagerduty',
  NOW()
);

-- ============================================================================
-- UPDATE CATEGORY COUNTS
-- ============================================================================

-- Refresh template counts for categories
UPDATE template_categories c
SET template_count = (
  SELECT COUNT(*) 
  FROM process_templates t 
  WHERE t.category = c.category_key 
    AND t.published_at IS NOT NULL
);

-- ============================================================================
-- VERIFICATION QUERY
-- ============================================================================

-- Verify seeded data
SELECT 
  category,
  COUNT(*) as template_count,
  ROUND(AVG(rating_average), 2) as avg_rating,
  SUM(clone_count) as total_clones
FROM process_templates
GROUP BY category
ORDER BY category;

-- Show featured templates
SELECT template_key, name, rating_average, clone_count
FROM process_templates
WHERE is_featured = true
ORDER BY rating_average DESC;
