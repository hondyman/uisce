-- =============================================
-- Metric: customer_acquisition_cost
-- DirectQuery Compatibility: High - CAC calculation
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW customer_acquisition_cost AS SELECT SUM(me.amount) / SUM(nca.count) AS value FROM marketing_expenses me JOIN new_customer_acquisitions nca ON me.entity_id = nca.entity_id AND me.as_of_date = nca.as_of_date GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON customer_acquisition_cost TO reporting_users;

