-- =============================================
-- Metric: currency_contribution_to_return
-- DirectQuery Compatibility: High - Currency contribution
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW currency_contribution_to_return AS SELECT prb.portfolio_return_in_base_currency - prl.portfolio_return_in_local_currency AS value FROM portfolio_return_in_base_currency prb JOIN portfolio_return_in_local_currency prl ON prb.entity_id = prl.entity_id AND prb.as_of_date = prl.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON currency_contribution_to_return TO reporting_users;

