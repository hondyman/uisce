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
        EXECUTE format('DROP POLICY IF EXISTS %I ON orm.%I', t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON orm.%I', t || '_tenant_write', t);
        EXECUTE format('ALTER TABLE orm.%I NO FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE orm.%I DISABLE ROW LEVEL SECURITY', t);
    END LOOP;
END
$rls$;
