-- Create source from Redpanda topic (Debezium)
CREATE SOURCE holdings_src
FROM KAFKA BROKER 'redpanda:9092' TOPIC 'dbserver.public.holdings'
FORMAT AVRO USING SCHEMA '...';

-- Create Materialized View with canonicalization logic (Tie-Breaker: SETTLED > EOD > SOD)
CREATE MATERIALIZED VIEW holdings_canonical AS
SELECT account_id, security_id, valuation_date,
  FIRST_VALUE(market_value) OVER (
    PARTITION BY account_id, security_id, valuation_date
    ORDER BY CASE holding_type WHEN 'SETTLED' THEN 1 WHEN 'EOD' THEN 2 WHEN 'SOD' THEN 3 ELSE 4 END
  ) AS market_value_resolved,
  MAX(as_of_timestamp) AS as_of_timestamp
FROM holdings_src;
