{{ config(materialized='table') }}

WITH ranked AS (
  SELECT
    id, account_id, security_id, holding_type, market_value, valuation_date, settlement_date, currency,
    ROW_NUMBER() OVER (
      PARTITION BY account_id, security_id, valuation_date
      ORDER BY CASE holding_type WHEN 'SETTLED' THEN 1 WHEN 'EOD' THEN 2 WHEN 'SOD' THEN 3 ELSE 4 END
    ) AS rn
  FROM {{ ref('raw_holdings') }}
  WHERE valuation_date = current_date
)
SELECT account_id, security_id, valuation_date, SUM(market_value) AS market_value_resolved
FROM ranked
WHERE rn = 1
GROUP BY account_id, security_id, valuation_date;
