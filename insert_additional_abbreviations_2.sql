-- Insert additional financial abbreviations into sml.abbreviation_lookup
-- This script adds accounting standards, valuation, and operational abbreviations

-- Accounting Standards
INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes) VALUES
('GAAP', 'Generally Accepted Accounting Principles', 'The common set of accounting principles, standards, and procedures that companies in the U.S. must follow when they compile their financial statements.'),
('IFRS', 'International Financial Reporting Standards', 'The international equivalent of GAAP, used by most other countries. It''s important to know which standard a fund or company is using.'),
('FASB', 'Financial Accounting Standards Board', 'The private, non-profit organization whose primary purpose is to establish and improve GAAP within the United States.'),
('ASC', 'Accounting Standards Codification', 'The ASC is the source of authoritative GAAP recognized by the FASB. For example, "ASC 820" is the specific rule for Fair Value Measurement.'),

-- Investment Valuation & Classification
('ABO', 'Accumulated Benefit Obligation', 'The present value of pension benefits owed to employees, based on their current salaries.'),
('AFS', 'Available-for-Sale', 'A classification for debt and equity securities that are not held for trading but may be sold before maturity. Unrealized gains/losses are reported in "other comprehensive income" (OCI).'),
('AMORT', 'Amortization', 'The practice of spreading an intangible asset''s cost over that asset''s useful life. For bonds, it''s the process of gradually writing off the initial cost of the asset.'),
('CG', 'Capital Gain', 'The profit from the sale of an asset.'),
('FV', 'Fair Value', 'The price that an asset would sell for on the open market. This is a crucial concept, especially for valuing complex or illiquid securities.'),
('FVTPL', 'Fair Value Through Profit or Loss', 'An accounting treatment under IFRS where changes in an asset''s fair value are recorded directly on the income statement.'),
('HTM', 'Held-to-Maturity', 'A classification for debt securities that a company intends to hold until they mature. They are recorded at amortized cost.'),
('OCI', 'Other Comprehensive Income', 'An entry on a financial statement that includes revenues, expenses, gains, and losses that have not yet been realized. It''s often where unrealized gains/losses from AFS securities are parked.'),
('PBO', 'Projected Benefit Obligation', 'The present value of pension benefits owed to employees, but it assumes future salary increases.'),
('UGL', 'Unrealized Gain/Loss', 'A "paper" profit or loss on an investment that has not yet been sold.'),

-- Operational & Fund Accounting
('ABOR', 'Accounting Book of Record', 'The official, audited accounting ledger for a fund or firm. It''s the "source of truth" for financial reporting, often calculated at the end of the day (T+1).'),
('GAV', 'Gross Asset Value', 'The total value of all assets in a fund, before deducting liabilities and expenses.'),
('IBOR', 'Investment Book of Record', 'A real-time or near-real-time view of a firm''s positions, used by portfolio managers for decision-making throughout the day. ABOR is the official end-of-day version.'),
('NAV', 'Net Asset Value', 'This is the fund''s per-share market value. It''s the total value of a fund''s assets minus its liabilities, divided by the number of shares outstanding. This is a cornerstone of fund accounting.'),
('SLA', 'Service-Level Agreement', 'A contract that defines the level of service expected from a service provider, like a fund administrator or custodian.'),
('SPV', 'Special Purpose Vehicle', 'A subsidiary company with an asset/liability structure and legal status that makes its obligations secure even if the parent company goes bankrupt. Often used to hold specific investments.')

ON CONFLICT (abbreviation)
DO UPDATE SET
    full_word = EXCLUDED.full_word,
    notes = EXCLUDED.notes;

-- Verify the insertions
SELECT COUNT(*) as total_abbreviations FROM sml.abbreviation_lookup;
SELECT abbreviation, full_word FROM sml.abbreviation_lookup WHERE abbreviation IN ('GAAP', 'IFRS', 'FASB', 'ASC', 'ABO', 'AFS', 'AMORT', 'CG', 'FV', 'FVTPL', 'HTM', 'OCI', 'PBO', 'UGL', 'ABOR', 'GAV', 'IBOR', 'NAV', 'SLA', 'SPV') ORDER BY abbreviation;