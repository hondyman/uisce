-- Migration 20261224_022_compliance_statutory_calendar_schedule.up.sql
-- Creates canonical statutory compliance schedule repository with legal citations and versioning

CREATE TABLE IF NOT EXISTS compliance.statutory_calendar_schedule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_code VARCHAR(100) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    jurisdiction VARCHAR(50) NOT NULL,
    regulation VARCHAR(100) NOT NULL,
    deadline_type VARCHAR(50) NOT NULL,
    recurrence_pattern VARCHAR(50) NOT NULL,
    month_offsets INT[] NOT NULL,
    day_of_month INT NOT NULL,
    cutoff_time VARCHAR(20) NOT NULL,
    legal_citation TEXT NOT NULL,
    rule_ids TEXT[] NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'HIGH',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed canonical regulatory standing schedules
INSERT INTO compliance.statutory_calendar_schedule (
    schedule_code, title, description, jurisdiction, regulation, deadline_type,
    recurrence_pattern, month_offsets, day_of_month, cutoff_time, legal_citation, rule_ids, severity
) VALUES 
(
    'SEC_13F_QUARTERLY',
    'SEC Form 13F Institutional Holdings Filing',
    'Mandatory quarterly filing for institutional investment managers exercising investment discretion over $100M+ in Section 13(f) securities. Due 45 calendar days after calendar quarter end.',
    'US',
    'SEC Form 13F',
    'STATUTORY_FILING',
    'QUARTERLY_45D',
    '{2, 5, 8, 11}',
    14,
    '17:30 EST',
    'Securities Exchange Act of 1934 Section 13(f)(1); 17 CFR § 240.13f-1',
    '{"SEC-13F-001"}',
    'HIGH'
),
(
    'SEC_FORM_PF_QUARTERLY',
    'SEC Form PF Systemic Risk Reporting',
    'Mandatory reporting of private fund regulatory assets under management, leverage, borrowing, and liquidity profiles for FSOC monitoring.',
    'US',
    'SEC Form PF',
    'STATUTORY_FILING',
    'QUARTERLY_60D',
    '{3, 5, 8, 11}',
    29,
    '17:30 EST',
    'Dodd-Frank Wall Street Reform and Consumer Protection Act Section 404/406; 17 CFR § 275.204(b)-1',
    '{"SEC-PF-001"}',
    'HIGH'
),
(
    'UK_TAKEOVER_RULE_8_3',
    'UK Takeover Panel Rule 8.3 Dealing Disclosure Review',
    'Public dealing disclosure deadline for dealings and positions in relevant securities of 1% or more of an offeree or offeror company during an offer period.',
    'UK',
    'Takeover Panel Rule 8.3',
    'DISCLOSURE_CUTOFF',
    'MONTHLY_MID',
    '{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}',
    15,
    '12:00 BST',
    'The City Code on Takeovers and Mergers, Rule 8.3',
    '{"UK-TAKEOVER-RULE83"}',
    'CRITICAL'
),
(
    'EU_UCITS_LST_QUARTERLY',
    'UCITS & AIFMD Liquidity Stress Testing & Concentration Review',
    'ESMA Guidelines on Liquidity Stress Testing in UCITS and AIFs: mandatory periodic stress simulation of asset liquidation horizons and redemption shock absorption.',
    'EU',
    'UCITS / ESMA Guidelines',
    'PORTFOLIO_REVIEW',
    'QUARTERLY_PERIOD_END',
    '{3, 6, 9, 12}',
    30,
    '18:00 CET',
    'ESMA34-39-897; Directive 2009/65/EC (UCITS) Article 51',
    '{"UCITS-5-10-40", "UCITS-GLOBAL-EXP-001"}',
    'HIGH'
),
(
    'UK_CASS_RP_MONTHLY',
    'FCA CASS 10 Resolution Pack Review & Attestation',
    'Client Assets Sourcebook verification ensuring master custody and ledger records can be retrieved within 48 hours in the event of firm resolution or insolvency.',
    'UK',
    'FCA CASS 10',
    'MANDATE_ATTESTATION',
    'MONTHLY_LAST_DAY',
    '{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}',
    31,
    '17:00 BST',
    'FCA Handbook CASS 10.1 - 10.3 (CASS Resolution Pack)',
    '{"CASS-SEGREGATION-001"}',
    'MEDIUM'
),
(
    'EU_SSR_SHORT_DISCLOSURE',
    'EU Short Selling Regulation Net Short Position Reporting',
    'Reporting of net short positions exceeding 0.1% of issued share capital of EU sovereign or corporate issuers to national competent authorities (NCAs).',
    'EU',
    'EU SSR 236/2012',
    'DISCLOSURE_CUTOFF',
    'MONTHLY_MID',
    '{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}',
    15,
    '15:30 CET',
    'Regulation (EU) No 236/2012 Article 5 & Article 6',
    '{"EU-SSR-REPORTING-001"}',
    'HIGH'
)
ON CONFLICT (schedule_code) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    jurisdiction = EXCLUDED.jurisdiction,
    regulation = EXCLUDED.regulation,
    deadline_type = EXCLUDED.deadline_type,
    recurrence_pattern = EXCLUDED.recurrence_pattern,
    month_offsets = EXCLUDED.month_offsets,
    day_of_month = EXCLUDED.day_of_month,
    cutoff_time = EXCLUDED.cutoff_time,
    legal_citation = EXCLUDED.legal_citation,
    rule_ids = EXCLUDED.rule_ids,
    severity = EXCLUDED.severity,
    updated_at = now();
