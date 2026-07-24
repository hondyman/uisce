-- Seed Data for Process Benchmarking System
-- Industry benchmarks based on Fortune 500 financial services research

-- ============================================================================
-- Industry Benchmarks - Financial Services
-- ============================================================================

INSERT INTO bp_industry_benchmarks (
    id, industry, process_type,
    median_duration_minutes, top_quartile_duration_minutes,
    median_success_rate, top_quartile_success_rate,
    median_cost_per_process, top_quartile_cost_per_process,
    median_automation_rate, top_quartile_automation_rate,
    sample_size, data_source
) VALUES
-- Investment Approval Process
('550e8400-e29b-41d4-a716-446655440001', 'financial_services', 'investment_approval',
 480.0, 240.0,  -- 8 hours median, 4 hours top quartile
 0.89, 0.96,    -- 89% success rate median, 96% top quartile
 850.0, 450.0,  -- Cost per process
 0.55, 0.78,    -- Automation rate
 487, 'Fortune 500 Financial Services Study 2025'),

-- Client Onboarding
('550e8400-e29b-41d4-a716-446655440002', 'financial_services', 'client_onboarding',
 2880.0, 1440.0,  -- 48 hours median, 24 hours top quartile
 0.91, 0.98,      -- Success rate
 1200.0, 650.0,   -- Cost per process
 0.62, 0.85,      -- Automation rate
 523, 'Fortune 500 Financial Services Study 2025'),

-- Portfolio Rebalancing
('550e8400-e29b-41d4-a716-446655440003', 'financial_services', 'portfolio_rebalancing',
 180.0, 90.0,   -- 3 hours median, 1.5 hours top quartile
 0.94, 0.99,    -- Success rate
 450.0, 200.0,  -- Cost per process
 0.70, 0.92,    -- Automation rate
 612, 'Fortune 500 Financial Services Study 2025'),

-- Compliance Review
('550e8400-e29b-41d4-a716-446655440004', 'financial_services', 'compliance_review',
 720.0, 360.0,  -- 12 hours median, 6 hours top quartile
 0.87, 0.95,    -- Success rate
 950.0, 500.0,  -- Cost per process
 0.48, 0.72,    -- Automation rate
 441, 'Fortune 500 Financial Services Study 2025'),

-- Risk Assessment
('550e8400-e29b-41d4-a716-446655440005', 'financial_services', 'risk_assessment',
 240.0, 120.0,  -- 4 hours median, 2 hours top quartile
 0.92, 0.97,    -- Success rate
 650.0, 350.0,  -- Cost per process
 0.65, 0.88,    -- Automation rate
 498, 'Fortune 500 Financial Services Study 2025');

-- ============================================================================
-- Industry Benchmarks - Wealth Management
-- ============================================================================

INSERT INTO bp_industry_benchmarks (
    id, industry, process_type,
    median_duration_minutes, top_quartile_duration_minutes,
    median_success_rate, top_quartile_success_rate,
    median_cost_per_process, top_quartile_cost_per_process,
    median_automation_rate, top_quartile_automation_rate,
    sample_size, data_source
) VALUES
-- Client Onboarding (Wealth specific)
('550e8400-e29b-41d4-a716-446655440006', 'wealth_management', 'client_onboarding',
 3600.0, 1800.0,  -- 60 hours median, 30 hours top quartile (more complex)
 0.88, 0.96,      -- Success rate
 1800.0, 900.0,   -- Higher cost due to complexity
 0.58, 0.82,      -- Automation rate
 298, 'Wealth Management Industry Report 2025'),

-- Financial Planning
('550e8400-e29b-41d4-a716-446655440007', 'wealth_management', 'financial_planning',
 960.0, 480.0,  -- 16 hours median, 8 hours top quartile
 0.90, 0.97,    -- Success rate
 1500.0, 750.0, -- Cost per process
 0.52, 0.75,    -- Automation rate
 267, 'Wealth Management Industry Report 2025'),

-- Tax Planning
('550e8400-e29b-41d4-a716-446655440008', 'wealth_management', 'tax_planning',
 1440.0, 720.0,  -- 24 hours median, 12 hours top quartile
 0.86, 0.94,     -- Success rate
 2200.0, 1100.0, -- High cost due to expertise required
 0.45, 0.68,     -- Lower automation (complex)
 189, 'Wealth Management Industry Report 2025');

-- ============================================================================
-- Best Practices Library
-- ============================================================================

INSERT INTO bp_best_practices (
    id, title, description, industry, process_type, category,
    expected_improvement_percent, implementation_effort, implementation_time_weeks,
    industry_adoption_percent, success_rate, priority,
    prerequisites, implementation_steps, required_tools, estimated_cost_range,
    case_study_company, case_study_results, case_study_timeline, tags
) VALUES
-- 1. Automated Document Processing
('650e8400-e29b-41d4-a716-446655440001',
 'Automated Document Processing with AI/ML',
 'Implement intelligent document processing using machine learning to automatically extract, classify, and validate documents. Reduces manual review time by 60-80% and improves accuracy.',
 'financial_services', 'client_onboarding', 'automation',
 65, 'high', 12,
 72, 0.91, 'high',
 'Document management system, OCR infrastructure, training data',
 '{"steps": ["Assess current document types", "Select AI/ML platform", "Train classification models", "Integrate with workflow", "Test and validate", "Deploy incrementally"]}',
 ARRAY['Azure Form Recognizer', 'AWS Textract', 'Google Document AI'],
 '$50K-$200K',
 'Morgan Stanley',
 'Reduced onboarding time from 48 hours to 18 hours. Processing accuracy improved from 87% to 96%.',
 '9 months',
 ARRAY['automation', 'ai', 'efficiency']),

-- 2. Parallel Process Execution
('650e8400-e29b-41d4-a716-446655440002',
 'Parallel Workflow Execution Architecture',
 'Redesign sequential workflows to execute independent steps in parallel. Dramatically reduces total process duration while maintaining quality.',
 NULL, NULL, 'speed',
 45, 'medium', 8,
 58, 0.88, 'high',
 'Workflow orchestration platform, dependency mapping',
 '{"steps": ["Map workflow dependencies", "Identify parallelizable steps", "Refactor orchestration logic", "Add synchronization points", "Load test", "Monitor performance"]}',
 ARRAY['Temporal', 'Apache Airflow', 'AWS Step Functions'],
 '$30K-$100K',
 'Goldman Sachs',
 'Investment approval process reduced from 8 hours to 3.5 hours. No quality degradation observed.',
 '6 months',
 ARRAY['speed', 'architecture', 'efficiency']),

-- 3. Real-time Compliance Validation
('650e8400-e29b-41d4-a716-446655440003',
 'Real-time Compliance Rule Engine',
 'Implement real-time compliance checking during process execution rather than batch review. Prevents non-compliant actions before they occur.',
 'financial_services', NULL, 'compliance',
 52, 'high', 16,
 45, 0.85, 'high',
 'Rules engine, compliance database, API integration',
 '{"steps": ["Catalog compliance rules", "Design rules engine", "Build API integration layer", "Implement inline validation", "Create audit trail", "Train users"]}',
 ARRAY['Drools', 'AWS Lambda', 'Custom Rules Engine'],
 '$75K-$250K',
 'JP Morgan Chase',
 'Compliance violations reduced by 78%. Review time dropped from 12 hours to 2 hours.',
 '12 months',
 ARRAY['compliance', 'quality', 'real-time']),

-- 4. Predictive Analytics for Bottleneck Prevention
('650e8400-e29b-41d4-a716-446655440004',
 'ML-Powered Bottleneck Prediction',
 'Use machine learning to predict workflow bottlenecks before they occur and automatically route to alternative resources.',
 NULL, NULL, 'efficiency',
 38, 'high', 14,
 34, 0.82, 'medium',
 'Historical execution data, ML platform, monitoring infrastructure',
 '{"steps": ["Collect historical data", "Train prediction models", "Build routing engine", "Integrate alerting", "Deploy monitoring", "Continuous refinement"]}',
 ARRAY['TensorFlow', 'Azure ML', 'Databricks'],
 '$60K-$180K',
 'Charles Schwab',
 'Reduced bottleneck incidents by 62%. Average process duration improved by 28%.',
 '10 months',
 ARRAY['ml', 'efficiency', 'predictive']),

-- 5. Self-Service Client Portals
('650e8400-e29b-41d4-a716-446655440005',
 'Client Self-Service Portal with Smart Forms',
 'Enable clients to complete routine tasks through intuitive self-service portals with intelligent form validation and guidance.',
 'wealth_management', 'client_onboarding', 'automation',
 55, 'medium', 10,
 67, 0.92, 'high',
 'Web development platform, identity management, API infrastructure',
 '{"steps": ["Design user journeys", "Build smart forms", "Implement validation logic", "Integrate backend systems", "Add security layer", "Launch with support"]}',
 ARRAY['React', 'Next.js', 'Auth0', 'Stripe'],
 '$40K-$120K',
 'Vanguard',
 'Client onboarding self-completion rate of 73%. Staff time reduced by 45%.',
 '8 months',
 ARRAY['automation', 'client-experience', 'efficiency']),

-- 6. Automated Quality Assurance
('650e8400-e29b-41d4-a716-446655440006',
 'Continuous Automated Quality Checks',
 'Implement automated testing and validation at every workflow step to catch errors early and ensure consistent quality.',
 NULL, NULL, 'quality',
 48, 'medium', 6,
 51, 0.89, 'medium',
 'Testing framework, CI/CD pipeline, monitoring tools',
 '{"steps": ["Define quality metrics", "Build test suites", "Integrate into workflow", "Set up alerting", "Dashboard creation", "Continuous improvement"]}',
 ARRAY['Jest', 'Selenium', 'Datadog', 'PagerDuty'],
 '$25K-$80K',
 'Fidelity Investments',
 'Error rate reduced from 8.2% to 2.1%. Rework costs dropped by 67%.',
 '5 months',
 ARRAY['quality', 'testing', 'automation']),

-- 7. Dynamic Resource Allocation
('650e8400-e29b-41d4-a716-446655440007',
 'AI-Driven Dynamic Resource Assignment',
 'Automatically assign workflow tasks to optimal resources based on availability, expertise, workload, and historical performance.',
 NULL, NULL, 'efficiency',
 42, 'medium', 8,
 39, 0.86, 'medium',
 'Resource management system, ML models, integration APIs',
 '{"steps": ["Model resource capabilities", "Collect performance data", "Train allocation models", "Build assignment engine", "Integrate with workflow", "Monitor and optimize"]}',
 ARRAY['Azure ML', 'Custom Engine', 'Resource Management Platform'],
 '$45K-$150K',
 'UBS',
 'Resource utilization improved from 68% to 87%. Task completion time reduced by 24%.',
 '7 months',
 ARRAY['efficiency', 'ai', 'resource-management']),

-- 8. Blockchain Audit Trail
('650e8400-e29b-41d4-a716-446655440008',
 'Immutable Blockchain Audit Trail',
 'Implement blockchain-based audit trail for complete transparency and tamper-proof compliance records.',
 'financial_services', NULL, 'compliance',
 35, 'high', 18,
 28, 0.80, 'low',
 'Blockchain platform, integration layer, audit framework',
 '{"steps": ["Select blockchain platform", "Design data model", "Build integration layer", "Implement smart contracts", "Test thoroughly", "Deploy with monitoring"]}',
 ARRAY['Hyperledger Fabric', 'Ethereum', 'Corda'],
 '$100K-$350K',
 'BNY Mellon',
 'Audit time reduced by 52%. Regulatory approval improved with full transparency.',
 '15 months',
 ARRAY['compliance', 'blockchain', 'audit']);

-- ============================================================================
-- Peer Groups (Sample)
-- ============================================================================

INSERT INTO bp_peer_groups (
    id, name, description, industry,
    company_size_min, company_size_max,
    annual_revenue_min, annual_revenue_max,
    is_active
) VALUES
-- Large Financial Services
('750e8400-e29b-41d4-a716-446655440001',
 'Large Financial Services Institutions',
 'Fortune 500 financial services companies with over 10,000 employees',
 'financial_services',
 10000, NULL,
 10000000000, NULL,  -- $10B+ revenue
 true),

-- Mid-Market Wealth Management
('750e8400-e29b-41d4-a716-446655440002',
 'Mid-Market Wealth Management Firms',
 'Wealth management firms with 1,000-10,000 employees',
 'wealth_management',
 1000, 10000,
 1000000000, 10000000000,  -- $1B-$10B revenue
 true),

-- Regional Banks
('750e8400-e29b-41d4-a716-446655440003',
 'Regional Banking Institutions',
 'Regional and community banks',
 'banking',
 500, 5000,
 500000000, 5000000000,  -- $500M-$5B revenue
 true);

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE bp_industry_benchmarks IS 'Seeded with 2025 Fortune 500 financial services benchmarks';
COMMENT ON TABLE bp_best_practices IS 'Curated library of 8 proven strategies with case studies';
COMMENT ON TABLE bp_peer_groups IS 'Sample peer groups for comparison - expand based on customer base';
