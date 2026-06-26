-- Seed file for private_markets database
-- Run this against the 'private_markets' database.

CREATE SCHEMA IF NOT EXISTS private_markets;

CREATE TABLE IF NOT EXISTS private_markets.funds (
  id integer PRIMARY KEY,
  name text NOT NULL,
  strategy text,
  aum numeric,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS private_markets.cash_flows (
  id serial PRIMARY KEY,
  fund_id integer REFERENCES private_markets.funds(id),
  cf_date date NOT NULL,
  amount numeric NOT NULL
);

-- Sample data
INSERT INTO private_markets.funds (id, name, strategy, aum) VALUES
  (1, 'Alpha Fund', 'Private Equity', 100000000),
  (2, 'Beta Fund', 'Real Estate', 50000000)
ON CONFLICT (id) DO NOTHING;

INSERT INTO private_markets.cash_flows (fund_id, cf_date, amount) VALUES
  (1, '2018-01-01', -5000000),
  (1, '2019-01-01', 1000000),
  (1, '2020-01-01', 20000000),
  (2, '2019-06-01', -2000000),
  (2, '2021-06-01', 5000000)
ON CONFLICT DO NOTHING;

-- Quick verification
-- SELECT count(*) FROM private_markets.funds;
-- SELECT * FROM private_markets.cash_flows ORDER BY cf_date;
