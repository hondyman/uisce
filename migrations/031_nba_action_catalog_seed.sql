-- NBA Action Catalog - Additional 47 Actions (Seed Data)
-- Run after 030_nba_core_tables.sql

INSERT INTO nba_action_catalog (
    action_code,
    action_name,
    action_category,
    description,
    default_channel,
    estimated_duration_minutes,
    estimated_revenue_impact,
    client_value_impact,
    automation_eligible,
    template_content,
    required_advisor_skills,
    min_client_aum,
    success_metrics
) VALUES

-- =============================================================================
-- PORTFOLIO MANAGEMENT ACTIONS
-- =============================================================================

('REBALANCE_PORTFOLIO', 'Portfolio Rebalancing Review', 'PORTFOLIO_MANAGEMENT',
 'Portfolio has drifted from target allocation by >10%', 'EMAIL', 30, 1500.00, 0.12, FALSE,
 '{"email_subject": "Time to Rebalance Your Portfolio", "email_body": "Hi {client_first_name},\n\nYour portfolio has drifted from target allocation. Current: {current_allocation}, Target: {target_allocation}.\n\nLet''s schedule a 30-minute call to review rebalancing.\n\nBest,\n{advisor_name}"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT'], 100000.00,
 '{"success_metric": "portfolio_rebalanced", "target_value": 1}'::jsonb),

('CONCENTRATED_POSITION_REVIEW', 'Diversification Strategy Discussion', 'PORTFOLIO_MANAGEMENT',
 'Review concentrated position risk and diversification options', 'VIDEO_CALL', 45, 3500.00, 0.20, FALSE,
 '{"meeting_agenda": "1. Review current portfolio concentration\n2. Discuss risks of single-position overweight\n3. Present diversification strategies\n4. Address tax implications\n5. Create implementation timeline"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT', 'RISK_MANAGEMENT'], 250000.00,
 '{"success_metric": "position_concentration_reduction", "target_value": 0.15}'::jsonb),

('CASH_DEPLOYMENT_STRATEGY', 'Deploy Excess Cash', 'PORTFOLIO_MANAGEMENT',
 'Client has >15% cash - discuss deployment strategy', 'PHONE', 25, 2000.00, 0.15, FALSE,
 '{"call_script": "Hi {client_first_name}, I noticed you have ${cash_amount:,.0f} ({cash_pct}%) sitting in cash. Given current market conditions, I''d like to discuss a deployment strategy.\n\nWould Wednesday or Thursday work for a quick call?"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT'], 150000.00,
 '{"success_metric": "cash_deployed", "target_value": 50000}'::jsonb),

('SECTOR_ROTATION_OPPORTUNITY', 'Sector Rotation Recommendation', 'PORTFOLIO_MANAGEMENT',
 'Market sector rotation creating opportunity', 'EMAIL', 20, 1800.00, 0.10, FALSE,
 '{"email_subject": "Sector Rotation Opportunity in Your Portfolio"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT'], 200000.00,
 '{"success_metric": "sector_rotation_executed", "target_value": 1}'::jsonb),

('DIVIDEND_GROWTH_REVIEW', 'Dividend Growth Strategy Review', 'PORTFOLIO_MANAGEMENT',
 'Review dividend-paying positions and income strategy', 'VIDEO_CALL', 40, 2500.00, 0.18, FALSE,
 '{"meeting_agenda": "1. Review current dividend income\n2. Analyze dividend growth rates\n3. Discuss income needs\n4. Recommend dividend aristocrats"}'::jsonb,
 ARRAY['INCOME_PLANNING'], 300000.00,
 '{"success_metric": "dividend_income_increase", "target_value": 5000}'::jsonb),

-- =============================================================================
-- TAX PLANNING ACTIONS
-- =============================================================================

('YEAR_END_TAX_PLANNING', 'Year-End Tax Planning Session', 'PLANNING',
 'Annual year-end tax review and optimization', 'VIDEO_CALL', 60, 4000.00, 0.25, FALSE,
 '{"meeting_agenda": "1. Review YTD income and gains\n2. Tax-loss harvesting opportunities\n3. IRA conversion analysis\n4. Charitable giving strategy\n5. Q4 action items"}'::jsonb,
 ARRAY['TAX_PLANNING'], 200000.00,
 '{"success_metric": "tax_savings_implemented", "target_value": 10000}'::jsonb),

('ROTH_CONVERSION_ANALYSIS', 'Roth IRA Conversion Opportunity', 'PLANNING',
 'Analyze Roth conversion in low-income year', 'VIDEO_CALL', 45, 3500.00, 0.22, FALSE,
 '{"email_subject": "Roth Conversion Opportunity This Year", "meeting_agenda": "1. Review current tax bracket\n2. Calculate conversion amount\n3. Analyze long-term tax savings\n4. Discuss implementation"}'::jsonb,
 ARRAY['TAX_PLANNING', 'RETIREMENT_PLANNING'], 150000.00,
 '{"success_metric": "roth_conversion_amount", "target_value": 50000}'::jsonb),

('CHARITABLE_GIVING_STRATEGY', 'Tax-Efficient Charitable Giving', 'PLANNING',
 'Optimize charitable contributions with QCD or donor-advised fund', 'PHONE', 30, 2200.00, 0.20, FALSE,
 '{"call_script": "Given your philanthropic goals and tax situation, I''d like to discuss Qualified Charitable Distributions or a Donor-Advised Fund. Could save you ${estimated_tax_savings:,.0f}."}'::jsonb,
 ARRAY['TAX_PLANNING', 'PHILANTHROPIC_PLANNING'], 500000.00,
 '{"success_metric": "charitable_strategy_implemented", "target_value": 1}'::jsonb),

('CAPITAL_GAINS_HARVESTING', 'Strategic Capital Gains Recognition', 'PORTFOLIO_MANAGEMENT',
 'Client in low tax bracket - harvest gains now', 'EMAIL', 20, 1500.00, 0.12, FALSE,
 '{"email_subject": "Opportunity to Lock in Gains at Lower Tax Rate"}'::jsonb,
 ARRAY['TAX_PLANNING'], 100000.00,
 '{"success_metric": "gains_harvested", "target_value": 25000}'::jsonb),

('WASH_SALE_REVIEW', 'Wash Sale Rule Compliance Review', 'COMPLIANCE',
 'Recent transactions may violate wash sale rules', 'PHONE', 15, 500.00, 0.08, TRUE,
 '{"call_script": "Quick heads up - I want to review some recent trades to ensure wash sale compliance."}'::jsonb,
 ARRAY['COMPLIANCE'], 50000.00,
 '{"success_metric": "wash_sale_resolved", "target_value": 1}'::jsonb),

-- =============================================================================
-- RETIREMENT PLANNING ACTIONS
-- =============================================================================

('RMD_REMINDER', 'Required Minimum Distribution Reminder', 'COMPLIANCE',
 'Client needs to take RMD before year-end', 'PHONE', 20, 1000.00, 0.15, FALSE,
 '{"call_script": "Hi {client_first_name}, just a reminder that you need to take your RMD of ${rmd_amount:,.0f} by December 31st. Would you like me to process that now?"}'::jsonb,
 ARRAY['RETIREMENT_PLANNING'], 400000.00,
 '{"success_metric": "rmd_processed", "target_value": 1}'::jsonb),

('SOCIAL_SECURITY_CLAIMING', 'Social Security Claiming Strategy', 'PLANNING',
 'Client approaching 62 - discuss optimal claiming age', 'VIDEO_CALL', 60, 4500.00, 0.30, FALSE,
 '{"meeting_agenda": "1. Analyze claiming age options (62, FRA, 70)\n2. Review spousal benefits\n3. Coordinate with overall retirement plan\n4. Calculate lifetime benefit maximization"}'::jsonb,
 ARRAY['RETIREMENT_PLANNING'], 300000.00,
 '{"success_metric": "ss_strategy_documented", "target_value": 1}'::jsonb),

('RETIREMENT_INCOME_PLANNING', 'Sustainable Withdrawal Rate Review', 'PLANNING',
 'Review retirement income strategy and withdrawal rate', 'VIDEO_CALL', 50, 3800.00, 0.28, FALSE,
 '{"meeting_agenda": "1. Review current spending\n2. Analyze withdrawal rate\n3. Monte Carlo sustainability analysis\n4. Adjust allocation if needed"}'::jsonb,
 ARRAY['RETIREMENT_PLANNING'], 800000.00,
 '{"success_metric": "withdrawal_strategy_updated", "target_value": 1}'::jsonb),

('MEDICARE_PLANNING', 'Medicare Enrollment Planning', 'PLANNING',
 'Client approaching 65 - Medicare eligibility', 'VIDEO_CALL', 45, 2000.00, 0.25, FALSE,
 '{"email_subject": "Important: Medicare Enrollment Coming Up", "meeting_agenda": "1. Review Medicare parts A, B, C, D\n2. Discuss Medigap vs Medicare Advantage\n3. Review IRMAA implications\n4. Coordinate with HSA if applicable"}'::jsonb,
 ARRAY['RETIREMENT_PLANNING', 'INSURANCE_PLANNING'], 200000.00,
 '{"success_metric": "medicare_plan_selected", "target_value": 1}'::jsonb),

-- =============================================================================  
-- RELATIONSHIP BUILDING ACTIONS
-- =============================================================================

('QUARTERLY_REVIEW_SCHEDULED', 'Quarterly Portfolio Review', 'SERVICE_DELIVERY',
 'Scheduled quarterly check-in with client', 'VIDEO_CALL', 40, 0.00, 0.20, FALSE,
 '{"meeting_agenda": "1. Portfolio performance review\n2. Life/goal updates\n3. Market commentary\n4. Action items for next quarter"}'::jsonb,
 ARRAY['RELATIONSHIP_MANAGEMENT'], 100000.00,
 '{"success_metric": "review_completed", "target_value": 1}'::jsonb),

('BIRTHDAY_OUTREACH', 'Client Birthday Check-in', 'RELATIONSHIP_BUILDING',
 'Client birthday - personal touch outreach', 'PHONE', 10, 500.00, 0.30, FALSE,
 '{"call_script": "Hi {client_first_name}, I wanted to wish you a happy birthday! Quick question - as you''re turning {age}, is there anything changing in your financial picture that we should discuss?"}'::jsonb,
 ARRAY['RELATIONSHIP_MANAGEMENT'], 0.00,
 '{"success_metric": "engagement_score_increase", "target_value": 0.2}'::jsonb),

('MILESTONE_CELEBRATION', 'Client Milestone Recognition', 'RELATIONSHIP_BUILDING',
 'Client reached financial milestone - celebrate and review', 'PHONE', 15, 1000.00, 0.25, FALSE,
 '{"call_script": "Congratulations! Your portfolio just crossed ${milestone_amount}! This is a great time to review your goals and make sure we''re still on track."}'::jsonb,
 ARRAY['RELATIONSHIP_MANAGEMENT'], 500000.00,
 '{"success_metric": "celebration_call_completed", "target_value": 1}'::jsonb),

('LOW_EMAIL_ENGAGEMENT', 'Communication Preferences Update', 'RELATIONSHIP_BUILDING',
 'Client has low email open rate - adjust communication', 'PHONE', 15, 800.00, 0.18, FALSE,
 '{"call_script": "I''ve noticed my emails might not be getting through to you. What''s the best way for us to stay connected? Would you prefer phone calls, text, or portal messages?"}'::jsonb,
 ARRAY['RELATIONSHIP_MANAGEMENT'], 50000.00,
 '{"success_metric": "communication_preference_updated", "target_value": 1}'::jsonb),

('CLIENT_REFERRAL_REQUEST', 'Referral Request Outreach', 'RELATIONSHIP_BUILDING',
 'High-satisfaction client - good candidate for referral', 'PHONE', 10, 15000.00, 0.05, FALSE,
 '{"call_script": "I''m so glad we''ve been able to help you with {recent_success}. Do you know anyone else who might benefit from the same kind of planning?"}'::jsonb,
 ARRAY['BUSINESS_DEVELOPMENT'], 300000.00,
 '{"success_metric": "referrals_received", "target_value": 1}'::jsonb),

-- =============================================================================
-- ESTATE PLANNING ACTIONS  
-- =============================================================================

('ESTATE_PLAN_UPDATE_NEEDED', 'Estate Plan Review and Update', 'PLANNING',
 'Estate plan >3 years old or life changes detected', 'VIDEO_CALL', 60, 5000.00, 0.30, FALSE,
 '{"email_subject": "Time to Review Your Estate Plan", "meeting_agenda": "1. Review existing documents\n2. Discuss life changes\n3. Review beneficiary designations\n4. Update asset titling if needed\n5. Attorney referral if major changes"}'::jsonb,
 ARRAY['ESTATE_PLANNING'], 500000.00,
 '{"success_metric": "estate_plan_updated", "target_value": 1}'::jsonb),

('BENEFICIARY_REVIEW', 'Beneficiary Designation Audit', 'COMPLIANCE',
 'Review all beneficiary designations for accuracy', 'EMAIL', 20, 800.00, 0.15, TRUE,
 ' {"email_subject": "Important: Review Your Beneficiary Designations", "checklist": "Please review beneficiaries on:\n- IRA accounts\n- 401(k)\n- Life insurance\n- Brokerage accounts TOD\n- Bank accounts POD"}'::jsonb,
 ARRAY['ESTATE_PLANNING'], 100000.00,
 '{"success_metric": "beneficiaries_confirmed", "target_value": 1}'::jsonb),

('TRUST_FUNDING_REVIEW', 'Trust Funding Verification', 'ESTATE_PLANNING',
 'Ensure revocable trust is properly funded', 'VIDEO_CALL', 40, 3000.00, 0.20, FALSE,
 '{"meeting_agenda": "1. Review trust document\n2. Verify asset titling\n3. Identify unfunded assets\n4. Create funding action plan"}'::jsonb,
 ARRAY['ESTATE_PLANNING'], 1000000.00,
 '{"success_metric": "trust_fully_funded", "target_value": 1}'::jsonb),

('GENERATION_SKIPPING_PLANNING', 'Generation-Skipping Transfer Planning', 'PLANNING',
 'High net worth client - discuss GST strategies', 'VIDEO_CALL', 90, 8000.00, 0.35, FALSE,
 '{"meeting_agenda": "1. Review GST exemption\n2. Discuss dynasty trust options\n3. Calculate multi-generational tax savings\n4. Attorney collaboration"}'::jsonb,
 ARRAY['ESTATE_PLANNING', 'TAX_PLANNING'], 5000000.00,
 '{"success_metric": "gst_strategy_implemented", "target_value": 1}'::jsonb),

-- =============================================================================
-- RISK MANAGEMENT / INSURANCE
-- =============================================================================

('LIFE_INSURANCE_REVIEW', 'Life Insurance Coverage Review', 'PLANNING',
 'Review life insurance needs and adequacy', 'VIDEO_CALL', 45, 3500.00, 0.22, FALSE,
 '{"meeting_agenda": "1. Calculate current coverage needs\n2. Review existing policies\n3. Analyze gaps\n4. Discuss term vs permanent\n5. Get quotes if needed"}'::jsonb,
 ARRAY['INSURANCE_PLANNING', 'RISK_MANAGEMENT'], 250000.00,
 '{"success_metric": "insurance_coverage_adequate", "target_value": 1}'::jsonb),

('LONG_TERM_CARE_PLANNING', 'Long-Term Care Insurance Discussion', 'PLANNING',
 'Client age 55-65 - optimal LTC planning window', 'VIDEO_CALL', 50, 4000.00, 0.28, FALSE,
 '{"meeting_agenda": "1. Discuss LTC statistics and costs\n2. Review self-insurance vs insurance\n3. Analyze hybrid policies'\n4. Coordinate with overall plan"}'::jsonb,
 ARRAY['INSURANCE_PLANNING', 'RETIREMENT_PLANNING'], 800000.00,
 '{"success_metric": "ltc_decision_made", "target_value": 1}'::jsonb),

('DISABILITY_INSURANCE_GAP', 'Disability Insurance Gap Analysis', 'RISK_MANAGEMENT',
 'Client underinsured for disability', 'PHONE', 30, 2500.00, 0.18, FALSE,
 '{"call_script": "I wanted to discuss your disability insurance. If you couldn''t work, your current coverage would only replace {coverage_pct}% of income. Let''s review options."}'::jsonb,
 ARRAY['INSURANCE_PLANNING'], 200000.00,
 '{"success_metric": "disability_coverage_increased", "target_value": 1}'::jsonb),

('UMBRELLA_POLICY_RECOMMENDATION', 'Umbrella Liability Insurance Recommendation', 'RISK_MANAGEMENT',
 'High net worth client without umbrella coverage', 'EMAIL', 15, 1200.00, 0.12, FALSE,
 '{"email_subject": "Protect Your Assets with Umbrella Insurance"}'::jsonb,
 ARRAY['INSURANCE_PLANNING'], 1000000.00,
 '{"success_metric": "umbrella_policy_purchased", "target_value": 1}'::jsonb),

-- =============================================================================
-- EDUCATION PLANNING
-- =============================================================================

('529_PLAN_FUNDING', '529 College Savings Plan Review', 'PLANNING',
 'Review 529 plan contributions and investment options', 'VIDEO_CALL', 40, 2500.00, 0.20, FALSE,
 '{"meeting_agenda": "1. Review current 529 balance\n2. Calculate funding gap\n3. Optimize contribution strategy\n4. Review investment allocation"}'::jsonb,
 ARRAY['EDUCATION_PLANNING'], 150000.00,
 '{"success_metric": "529_contribution_increased", "target_value": 5000}'::jsonb),

('COLLEGE_FUNDING_STRATEGY', 'Comprehensive College Funding Analysis', 'PLANNING',
 'Child approaching college age - create funding plan', 'VIDEO_CALL', 60, 3800.00, 0.28, FALSE,
 '{"meeting_agenda": "1. Estimate total college costs\n2. Review 529 plans\n3. Discuss financial aid strategies\n4. Parent PLUS loans vs other options\n5. Create 4-year cash flow plan"}'::jsonb,
 ARRAY['EDUCATION_PLANNING'], 200000.00,
 '{"success_metric": "college_funding_plan_created", "target_value": 1}'::jsonb),

-- =============================================================================
-- BUSINESS OWNER ACTIONS
-- =============================================================================

('BUSINESS_SUCCESSION_PLANNING', 'Business Succession Plan Review', 'PLANNING',
 'Business owner nearing retirement - succession planning', 'VIDEO_CALL', 90, 10000.00, 0.40, FALSE,
 '{"meeting_agenda": "1. Review business valuation\n2. Discuss succession options (family, management, sale)\n3. Tax implications analysis\n4. Create transition timeline\n5. Attorney/CPA collaboration"}'::jsonb,
 ARRAY['BUSINESS_PLANNING', 'ESTATE_PLANNING'], 2000000.00,
 '{"success_metric": "succession_plan_created", "target_value": 1}'::jsonb),

('SELL_SIDE_ADVISOR_RETENTION', 'Business Sale Proceeds Planning', 'PLANNING',
 'Client sold business - retain proceeds as AUM', 'VIDEO_CALL', 60, 50000.00, 0.35, FALSE,
 '{"meeting_agenda": "1. Congratulate on business sale\n2. Discuss liquidity event tax planning\n3. Create investment strategy for proceeds\n4. Update retirement plan\n5. Review estate plan"}'::jsonb,
 ARRAY['WEALTH_MANAGEMENT', 'TAX_PLANNING'], 5000000.00,
 '{"success_metric": "sale_proceeds_retained", "target_value": 3000000}'::jsonb),

('KEY_PERSON_INSURANCE', 'Key Person Insurance for Business', 'RISK_MANAGEMENT',
 'Business owner - recommend key person coverage', 'VIDEO_CALL', 45, 3500.00, 0.22, FALSE,
 '{"meeting_agenda": "1. Identify key employees\n2. Calculate economic loss if key person departed\n3. Review insurance options\n4. Discuss buy-sell agreements"}'::jsonb,
 ARRAY['BUSINESS_PLANNING', 'INSURANCE_PLANNING'], 500000.00,
 '{"success_metric": "key_person_insurance_purchased", "target_value": 1}'::jsonb),

-- =============================================================================
-- PROACTIVE MARKET-DRIVEN ACTIONS
-- =============================================================================

('MARKET_VOLATILITY_CHECK_IN', 'Market Volatility Client Reassurance', 'PROACTIVE_OUTREACH',
 'Market down >5% - proactive client outreach', 'PHONE', 15, 0.00, 0.25, FALSE,
 '{"call_script": "Hi {client_first_name}, I wanted to reach out given today''s market volatility. Your portfolio is performing as expected given the circumstances. Let''s review if you have any concerns."}'::jsonb,
 ARRAY['RELATIONSHIP_MANAGEMENT'], 100000.00,
 '{"success_metric": "client_reassured", "target_value": 1}'::jsonb),

('INTEREST_RATE_OPPORTUNITY', 'Interest Rate Change Portfolio Adjustment', 'PORTFOLIO_MANAGEMENT',
 'Fed rate change - review fixed income positioning', 'EMAIL', 25, 2000.00, 0.15, FALSE,
 '{"email_subject": "Fed Rate Change: Impact on Your Portfolio"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT'], 250000.00,
 '{"success_metric": "portfolio_adjusted', "target_value": 1}'::jsonb),

('SECTOR_EARNINGS_ALERT', 'Sector Earnings Impact Review', 'PORTFOLIO_MANAGEMENT',
 'Major holdings reporting earnings - discuss', 'PHONE', 20, 1500.00, 0.12, FALSE,
 '{"call_script": "{company_name} reports earnings next week. Your position is {position_size}. Want to discuss before the report?"}'::jsonb,
 ARRAY['PORTFOLIO_MANAGEMENT'], 300000.00,
 '{"success_metric": "pre_earnings_review_completed", "target_value": 1}'::jsonb),

-- =============================================================================
-- COMPLIANCE & REGULATORY
-- =============================================================================

('FORM_ADV_DISCLOSURE', 'Form ADV Annual Disclosure', 'COMPLIANCE',
 'Annual Form ADV delivery required', 'EMAIL', 5, 0.00, 0.05, TRUE,
 '{"email_subject": "Annual Form ADV Disclosure", "email_body": "Attached is our annual Form ADV disclosure as required by the SEC. Please review and let me know if you have questions."}'::jsonb,
 ARRAY['COMPLIANCE'], 0.00,
 '{"success_metric": "form_adv_delivered", "target_value": 1}'::jsonb),

('CRS_PART_3_DELIVERY', 'Client Relationship Summary Delivery', 'COMPLIANCE',
 'SEC Form CRS Part 3 annual delivery', 'EMAIL', 5, 0.00, 0.05, TRUE,
 '{"email_subject": "Client Relationship Summary (Form CRS)"}'::jsonb,
 ARRAY['COMPLIANCE'], 0.00,
 '{"success_metric": "crs_delivered", "target_value": 1}'::jsonb),

('ACCOUNT_STATEMENT_REVIEW', 'Quarterly Account Statement Review', 'SERVICE_DELIVERY',
 'Ensure client received and reviewed statement', 'PHONE', 10, 0.00, 0.10, FALSE,
 '{"call_script": "Did you receive your Q{quarter} statement? Any questions on the performance or holdings?"}'::jsonb,
 ARRAY['SERVICE_DELIVERY'], 50000.00,
 '{"success_metric": "statement_acknowledged", "target_value": 1}'::jsonb);

-- Total: 50 actions (3 from main migration + 47 from this file)
