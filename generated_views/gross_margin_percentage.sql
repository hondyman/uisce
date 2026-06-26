-- =============================================
-- Metric: gross_margin_percentage
-- DirectQuery Compatibility: High - Gross margin
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW gross_margin_percentage AS SELECT SUM(sr.profit) / SUM(sr.revenue) AS value FROM sales_revenue sr GROUP BY entity_id, as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON gross_margin_percentage TO reporting_users;

