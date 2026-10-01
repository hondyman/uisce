-- 20261026_015_orm_rls_new_tables.up.sql
-- Enables RLS + FORCE + read/write policy pair on the 28 new orm tables.
--
-- Read: tenant's own rows OR the shared reference tenant's (default sentinel).
-- Write: tenant's own rows only.
--
-- PREREQUISITE: the application must set app.current_tenant per connection.
-- Without it, current_setting(..., true) returns NULL and no rows match.
-- FORCE ROW LEVEL SECURITY means the postgres role is also subject to policies.

DO $rls$
DECLARE
    t text;
    tables text[] := ARRAY[
        'order_event','order_amendment','order_reject','order_history',
        'execution_quality','order_benchmark',
        'pre_trade_check','restricted_list',
        'short_sell_locate',
        'position_lot','position_history','cash_balance',
        'quote','quote_default','market_data_snapshot',
        'trading_session','trading_halt',
        'trading_limit',
        'model_portfolio','model_portfolio_target','account_model_assignment',
        'basket','basket_item',
        'routing_rule',
        'pnl_intraday',
        'fx_exposure','fx_hedge'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE orm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE orm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON orm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON orm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$,
            t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON orm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON orm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$,
            t || '_tenant_write', t);
    END LOOP;
END
$rls$;
