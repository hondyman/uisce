-- NOTE: this targets the standalone `northwinds` Postgres database (see
-- NORTHWINDS_DATABASE_URL / config.yaml's aggregates_dsn), not the app's
-- own tenant-managed database (`alpha`) that the normal migration runner
-- applies against. Apply directly, e.g.:
--   psql "$NORTHWINDS_DATABASE_URL" -f 20260902_northwind_customer_cashflows_fixture.sql
--
-- customer_cashflows: Northwind fixture for the boresolver host-runtime
-- calc engine (customer_xirr). NOT real investment cash flows -- Northwind
-- is pure sales data with no natural "return" leg. Each order line item is
-- modeled as a negative cash flow (money the customer spends), and ONE
-- synthetic positive "realized value" leg is appended per customer so the
-- series has the sign change XIRR requires to solve. This is purely an
-- engineering fixture to exercise the calc engine (parsing, tier
-- resolution, batched SQLRowSource fetch, finlib.XIRR) against real data
-- volume and real irregular dates -- it is not a real financial metric.
--
-- tenant_id is the real "northwind" tenant row from the app's own tenants
-- table (alpha.tenants, id 99e99e99-99e9-49e9-89e9-99e99e99e999) -- not a
-- placeholder. Northwind itself has no tenant_id column and lives in a
-- separate physical database from the tenants table, so this value is
-- hardcoded to match that row rather than looked up at view-eval time.
CREATE OR REPLACE VIEW public.customer_cashflows AS
SELECT
    o.customer_id,
    o.order_date AS cashflow_date,
    (-1.0 * (od.unit_price * od.quantity * (1 - od.discount)))::double precision AS cashflow_amount,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AS tenant_id
FROM public.orders o
JOIN public.order_details od
  ON od.order_id = o.order_id AND od.order_date = o.order_date

UNION ALL

-- Synthetic terminal leg per customer: 15% above total spend, realized 30
-- days after their most recent order. TEST-ONLY -- fabricated so XIRR has
-- a sign change to solve against; not a real observed value.
SELECT
    o.customer_id,
    (MAX(o.order_date) + INTERVAL '30 days')::date AS cashflow_date,
    (1.15 * SUM(od.unit_price * od.quantity * (1 - od.discount)))::double precision AS cashflow_amount,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid AS tenant_id
FROM public.orders o
JOIN public.order_details od
  ON od.order_id = o.order_id AND od.order_date = o.order_date
GROUP BY o.customer_id;
