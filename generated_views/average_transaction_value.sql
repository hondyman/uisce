-- =============================================
-- Metric: average_transaction_value
-- DirectQuery Compatibility: High - Average transaction value
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW average_transaction_value AS SELECT SUM(ct.amount) / SUM(ct.count) AS value FROM customer_transactions ct GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON average_transaction_value TO reporting_users;

