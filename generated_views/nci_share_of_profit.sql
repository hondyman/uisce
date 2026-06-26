-- =============================================
-- Metric: nci_share_of_profit
-- DirectQuery Compatibility: High - NCI share
-- Generated from DAX-to-SQL mapping
-- =============================================

CREATE VIEW nci_share_of_profit AS SELECT sp.subsidiary_profit * np.nci_pct AS value FROM subsidiary_profit sp JOIN nci_pct np ON sp.entity_id = np.entity_id AND sp.as_of_date = np.as_of_date;

-- Grant permissions (adjust as needed)
-- GRANT SELECT ON nci_share_of_profit TO reporting_users;

