{{ config(materialized='table') }}

with ranked as (
  select
    id,
    account_id,
    security_id,
    holding_type,
    market_value,
    valuation_date,
    row_number() over (
      partition by account_id, security_id, valuation_date
      order by case holding_type when 'SETTLED' then 1 when 'EOD' then 2 when 'SOD' then 3 else 4 end
    ) as rn
  from {{ ref('stg_holdings') }}
)
select
  account_id,
  security_id,
  valuation_date,
  sum(market_value) as market_value_resolved
from ranked
where rn = 1
group by account_id, security_id, valuation_date
