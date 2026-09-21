-- Rollback for 20260920_002c_drop_all_logic_triggers.up.sql
-- Re-creates all 18 dropped triggers + 13 functions.
-- Authoritative bodies: backend/db/migrations/_reference/20260920_002_trigger_fn_sources.sql
--
-- criminology note: run the crims portion against the crims database separately.

BEGIN;

-- ============================================================================
-- oms: status-default + domain enforcement (polymorphic — 1 function, 4 triggers)
-- tgtype 7 = ROW BEFORE INSERT OR UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION oms.default_order_status()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    limit_order_codes TEXT[] := ARRAY['LIMIT','STOP_LIMIT','LOC','LOO','HYBRID'];
    stop_order_codes  TEXT[] := ARRAY['STOP','STOP_LIMIT'];
    ot_code TEXT;
BEGIN
    IF TG_TABLE_NAME = 'orders' THEN
        IF NEW.status_id IS NULL THEN
            SELECT id INTO NEW.status_id FROM ref.order_status WHERE code = 'NEW';
        END IF;
        IF NEW.limit_price IS NOT NULL THEN
            SELECT code INTO ot_code FROM ref.order_type WHERE id = NEW.order_type_id;
            IF ot_code IS NULL OR ot_code != ANY(limit_order_codes) THEN
                RAISE EXCEPTION 'limit_price requires limit-type order_type (got %)', ot_code USING ERRCODE = 'check_violation';
            END IF;
        END IF;
        IF NEW.stop_price IS NOT NULL THEN
            SELECT code INTO ot_code FROM ref.order_type WHERE id = NEW.order_type_id;
            IF ot_code IS NULL OR ot_code != ANY(stop_order_codes) THEN
                RAISE EXCEPTION 'stop_price requires stop-type order_type (got %)', ot_code USING ERRCODE = 'check_violation';
            END IF;
        END IF;
    ELSIF TG_TABLE_NAME = 'order_slice' THEN
        IF NEW.status_id IS NULL THEN
            SELECT id INTO NEW.status_id FROM ref.order_status WHERE code = 'WORKING';
        END IF;
    ELSIF TG_TABLE_NAME = 'allocation' THEN
        IF NEW.status_id IS NULL THEN
            SELECT id INTO NEW.status_id FROM ref.allocation_status WHERE code = 'PENDING';
        END IF;
    ELSIF TG_TABLE_NAME = 'settlement' THEN
        IF NEW.status_id IS NULL THEN
            SELECT id INTO NEW.status_id FROM ref.settlement_status WHERE code = 'PENDING';
        END IF;
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_orders_default_status
    BEFORE INSERT OR UPDATE ON oms.orders
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

CREATE TRIGGER trg_order_slice_default_status
    BEFORE INSERT OR UPDATE ON oms.order_slice
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

CREATE TRIGGER trg_allocation_default_status
    BEFORE INSERT OR UPDATE ON oms.allocation
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

CREATE TRIGGER trg_settlement_default_status
    BEFORE INSERT OR UPDATE ON oms.settlement
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

-- ============================================================================
-- crypto: holdings aggregate maintenance
-- tgtype 5  = ROW AFTER INSERT        (trigger_update_crypto_valuations)
-- tgtype 4  = STATEMENT AFTER INSERT  (trigger_refresh_prices)
-- tgtype 21 = ROW AFTER INSERT OR UPDATE (trigger_update_holdings)
-- ============================================================================

CREATE OR REPLACE FUNCTION public.update_crypto_holding_valuation()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    UPDATE crypto_holdings
    SET
        current_price_usd = NEW.price_usd,
        current_value_usd = quantity * NEW.price_usd,
        unrealized_gain_loss = (quantity * NEW.price_usd) - total_cost_basis,
        last_price_update = NEW.timestamp_utc
    WHERE asset_symbol = NEW.asset_symbol;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_update_crypto_valuations
    AFTER INSERT ON public.crypto_market_data
    FOR EACH ROW EXECUTE FUNCTION public.update_crypto_holding_valuation();

CREATE OR REPLACE FUNCTION public.refresh_latest_prices()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY crypto_latest_prices;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_refresh_prices
    AFTER INSERT ON public.crypto_prices
    FOR EACH STATEMENT EXECUTE FUNCTION public.refresh_latest_prices();

CREATE OR REPLACE FUNCTION public.update_holdings_on_transaction()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.status = 'CONFIRMED' AND (OLD.status IS NULL OR OLD.status != 'CONFIRMED') THEN
        IF NEW.txn_type IN ('BUY', 'TRANSFER_IN', 'REWARD', 'AIRDROP', 'STAKE') THEN
            INSERT INTO crypto_holdings (wallet_id, asset_symbol, contract_address, quantity, available_quantity)
            VALUES (NEW.wallet_id, NEW.asset_symbol, NEW.contract_address, NEW.quantity, NEW.quantity)
            ON CONFLICT (wallet_id, asset_symbol, COALESCE(contract_address, ''))
            DO UPDATE SET
                quantity = crypto_holdings.quantity + NEW.quantity,
                available_quantity = crypto_holdings.available_quantity + NEW.quantity,
                last_updated = NOW();
        ELSIF NEW.txn_type IN ('SELL', 'TRANSFER_OUT', 'UNSTAKE') THEN
            UPDATE crypto_holdings
            SET
                quantity = quantity - NEW.quantity,
                available_quantity = available_quantity - NEW.quantity,
                last_updated = NOW()
            WHERE wallet_id = NEW.wallet_id
              AND asset_symbol = NEW.asset_symbol
              AND COALESCE(contract_address, '') = COALESCE(NEW.contract_address, '');
        END IF;
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_update_holdings
    AFTER INSERT OR UPDATE ON public.crypto_transactions
    FOR EACH ROW EXECUTE FUNCTION public.update_holdings_on_transaction();

-- ============================================================================
-- investment_opportunities: scoring logic
-- tgtype 7 = ROW BEFORE INSERT OR UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.screen_investment_opportunity()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    screening_passed BOOLEAN := TRUE;
    screening_reasons TEXT[] := ARRAY[]::TEXT[];
    client_liquid_assets DECIMAL(15,2);
    screening_score DECIMAL(5,2) := 100;
BEGIN
    SELECT COALESCE(SUM(current_nav), 0) INTO client_liquid_assets
    FROM alternative_investments ai
    WHERE ai.client_id = NEW.client_id;
    IF NEW.minimum_commitment > (client_liquid_assets * 0.10) THEN
        screening_passed := FALSE;
        screening_reasons := screening_reasons || 'Minimum commitment exceeds 10% of alternative AUM';
        screening_score := screening_score - 20;
    END IF;
    IF NEW.vintage_year IS NOT NULL AND (NEW.vintage_year < 2025 OR NEW.vintage_year > 2028) THEN
        screening_reasons := screening_reasons || 'Vintage year outside preferred 2025-2028 range';
        screening_score := screening_score - 10;
    END IF;
    IF NEW.track_record_years_min IS NOT NULL AND NEW.track_record_years_min < 5 THEN
        screening_reasons := screening_reasons || 'Manager track record less than 5 years';
        screening_score := screening_score - 15;
    END IF;
    IF NEW.target_irr_min IS NOT NULL AND NEW.target_irr_min > 35 THEN
        screening_reasons := screening_reasons || 'Target IRR appears unrealistically high';
        screening_score := screening_score - 10;
    END IF;
    NEW.screening_passed := screening_passed;
    NEW.screening_reasons := screening_reasons;
    NEW.screening_score := GREATEST(screening_score, 0);
    NEW.screening_completed_at := NOW();
    IF screening_passed AND NEW.current_stage = 'INTAKE' THEN
        NEW.current_stage := 'INITIAL_SCREEN';
        NEW.stage_updated_at := NOW();
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INITIAL_SCREEN', 'timestamp', NOW(), 'notes', 'Automated screening passed');
    ELSIF NOT screening_passed AND NEW.current_stage = 'INTAKE' THEN
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INTAKE', 'timestamp', NOW(),
            'notes', 'Automated screening flagged issues: ' || array_to_string(screening_reasons, '; '));
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trg_screen_investment_opportunity
    BEFORE INSERT OR UPDATE ON public.investment_opportunities
    FOR EACH ROW EXECUTE FUNCTION public.screen_investment_opportunity();

-- ============================================================================
-- semantic_query_templates: RBAC seeding + version-on-update
-- tgtype 5  = ROW AFTER INSERT       (trigger_create_default_permissions)
-- tgtype 17 = ROW BEFORE UPDATE      (trigger_create_version_on_update)
-- ============================================================================

CREATE OR REPLACE FUNCTION vend.create_default_template_permissions()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
  INSERT INTO semantic_query_template_permissions
  (template_id, role, can_run, can_edit, can_delete, can_promote)
  VALUES (NEW.id, 'viewer', true, false, false, false);
  INSERT INTO semantic_query_template_permissions
  (template_id, role, can_run, can_edit, can_delete, can_promote)
  VALUES (NEW.id, 'editor', true, true, false, false);
  INSERT INTO semantic_query_template_permissions
  (template_id, role, can_run, can_edit, can_delete, can_promote)
  VALUES (NEW.id, 'admin', true, true, true, true);
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_create_default_permissions
    AFTER INSERT ON public.semantic_query_templates
    FOR EACH ROW EXECUTE FUNCTION vend.create_default_template_permissions();

CREATE OR REPLACE FUNCTION vend.create_template_version_on_update()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
  v_version_number INTEGER;
BEGIN
  SELECT COALESCE(MAX(version_number), 0) + 1 INTO v_version_number
  FROM semantic_query_template_versions
  WHERE template_id = NEW.id;
  INSERT INTO semantic_query_template_versions
  (template_id, version_number, name, description, semantic_query, parameters,
   change_message, created_by)
  VALUES (
    NEW.id, v_version_number, NEW.name, NEW.description, NEW.semantic_query,
    NEW.parameters, '', NEW.updated_by
  );
  RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_create_version_on_update
    BEFORE UPDATE ON public.semantic_query_templates
    FOR EACH ROW EXECUTE FUNCTION vend.create_template_version_on_update();

-- ============================================================================
-- cube: version-bump triggers
-- tgtype 17 = ROW BEFORE UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.cube_custom_model_version_trigger()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF OLD.custom_config IS DISTINCT FROM NEW.custom_config THEN
        INSERT INTO cube_custom_model_versions (custom_model_id, version, custom_config, changed_by)
        VALUES (NEW.id, NEW.version, NEW.custom_config, NEW.created_by);
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER cube_custom_model_version
    BEFORE UPDATE ON public.cube_custom_models
    FOR EACH ROW EXECUTE FUNCTION public.cube_custom_model_version_trigger();

CREATE OR REPLACE FUNCTION public.cube_security_policy_version_trigger()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF OLD.conditions IS DISTINCT FROM NEW.conditions OR OLD.effects IS DISTINCT FROM NEW.effects THEN
        INSERT INTO cube_security_policy_versions (policy_id, version, conditions, effects, changed_by)
        VALUES (NEW.id, NEW.version, NEW.conditions, NEW.effects, NEW.created_by);
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER cube_security_policy_version
    BEFORE UPDATE ON public.cube_security_policies
    FOR EACH ROW EXECUTE FUNCTION public.cube_security_policy_version_trigger();

-- ============================================================================
-- template_ratings: aggregate maintenance
-- tgtype 5  = ROW AFTER INSERT   (template_rating_inserted)
-- tgtype 17 = ROW BEFORE UPDATE  (template_rating_updated)
-- tgtype 9  = ROW BEFORE DELETE  (template_rating_deleted)
-- ============================================================================

CREATE OR REPLACE FUNCTION public.update_template_rating_stats()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    avg_rating DECIMAL(3,2);
    total_count INTEGER;
BEGIN
    SELECT
        COALESCE(AVG(rating), 0.0)::DECIMAL(3,2),
        COUNT(*)
    INTO avg_rating, total_count
    FROM template_ratings
    WHERE template_id = COALESCE(NEW.template_id, OLD.template_id)
      AND moderation_status = 'approved';
    UPDATE process_templates
    SET
        rating_average = avg_rating,
        rating_count = total_count,
        updated_at = NOW()
    WHERE id = COALESCE(NEW.template_id, OLD.template_id);
    RETURN COALESCE(NEW, OLD);
END;
$function$;

CREATE TRIGGER template_rating_inserted
    AFTER INSERT ON public.template_ratings
    FOR EACH ROW EXECUTE FUNCTION public.update_template_rating_stats();

CREATE TRIGGER template_rating_updated
    BEFORE UPDATE ON public.template_ratings
    FOR EACH ROW EXECUTE FUNCTION public.update_template_rating_stats();

CREATE TRIGGER template_rating_deleted
    BEFORE DELETE ON public.template_ratings
    FOR EACH ROW EXECUTE FUNCTION public.update_template_rating_stats();

-- ============================================================================
-- process_execution_metrics: duration computation
-- tgtype 19 = ROW BEFORE UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.calculate_step_duration()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.end_time IS NOT NULL AND OLD.end_time IS NULL THEN
        NEW.duration = NEW.end_time - OLD.start_time;
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER calculate_process_execution_metrics_duration
    BEFORE UPDATE ON public.process_execution_metrics
    FOR EACH ROW EXECUTE FUNCTION public.calculate_step_duration();

-- ============================================================================
-- capital_calls: stub liquidity check
-- tgtype 23 = ROW BEFORE INSERT OR UPDATE
-- ============================================================================

CREATE OR REPLACE FUNCTION public.check_capital_call_liquidity()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.liquidity_check_passed := TRUE;
    NEW.recommended_action := 'Sufficient liquidity confirmed.';
    RETURN NEW;
END;
$function$;

CREATE TRIGGER trigger_capital_call_liquidity_check
    BEFORE INSERT OR UPDATE ON public.capital_calls
    FOR EACH ROW EXECUTE FUNCTION public.check_capital_call_liquidity();

COMMIT;
-- ============================================================================
-- CRIMS SEPARATE (run against crims database, not alpha):
--
-- CREATE TRIGGER trg_sec_ident_cache
--     BEFORE INSERT OR UPDATE ON orm.security_identifier
--     FOR EACH ROW EXECUTE FUNCTION orm.sync_identifier_cache();
--
-- CREATE OR REPLACE FUNCTION orm.sync_identifier_cache()
--  RETURNS trigger LANGUAGE plpgsql AS $function$
-- BEGIN
--     IF NEW.is_primary AND NEW.effective_to IS NULL THEN
--         IF NEW.id_type = 'ISIN'
--           THEN UPDATE orm.security SET isin = NEW.id_value WHERE id = NEW.security_id;
--         ELSIF NEW.id_type = 'CUSIP'
--           THEN UPDATE orm.security SET cusip = NEW.id_value WHERE id = NEW.security_id;
--         ELSIF NEW.id_type = 'TICKER'
--           THEN UPDATE orm.security SET ticker = NEW.id_value WHERE id = NEW.security_id;
--         END IF;
--     END IF;
--     RETURN NEW;
-- END; $function$;
-- ============================================================================
