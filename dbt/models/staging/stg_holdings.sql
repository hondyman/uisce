with raw as (
  select * from {{ ref('raw_holdings') }}
)
select
  id,
  account_id,
  security_id,
  holding_type,
  cast(market_value as double) as market_value,
  cast(valuation_date as date) as valuation_date,
  settlement_date,
  currency,
  as_of_timestamp
from raw
