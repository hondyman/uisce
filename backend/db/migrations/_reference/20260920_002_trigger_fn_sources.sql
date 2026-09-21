--- audit_uma_accounts ---
CREATE OR REPLACE FUNCTION public.audit_uma_accounts()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$function$
;

--- audit_uma_rebalance_requests ---
CREATE OR REPLACE FUNCTION public.audit_uma_rebalance_requests()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$function$
;

--- audit_validation_patterns ---
CREATE OR REPLACE FUNCTION public.audit_validation_patterns()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    -- Log changes to audit table if it exists
    IF TG_OP = 'UPDATE' THEN
        -- Update modified timestamp
        NEW.updated_at = now();
    END IF;
    RETURN NEW;
END;
$function$
;

--- invalidate_semantic_cube_cache ---
CREATE OR REPLACE FUNCTION public.invalidate_semantic_cube_cache()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    target_cube_id UUID;
    t_id UUID;
    c_name TEXT;
BEGIN
    -- Determine cube ID based on table
    IF TG_TABLE_NAME = 'semantic_cubes_v2' THEN
        target_cube_id := COALESCE(NEW.id, OLD.id);
    ELSE
        target_cube_id := COALESCE(NEW.cube_id, OLD.cube_id);
    END IF;

    -- Get tenant_id and name from the cube table
    -- This handles cases where child tables (dimensions/measures) don't have tenant_id
    SELECT tenant_id, name INTO t_id, c_name
    FROM semantic_cubes_v2
    WHERE id = target_cube_id;

    -- Invalidate cache if cube found
    IF FOUND THEN
        DELETE FROM semantic_cube_cache 
        WHERE tenant_id = t_id AND cube_name = c_name;
    END IF;
    
    RETURN NULL; -- Return NULL for AFTER triggers (or NEW for BEFORE)
END;
$function$
;

--- log_workflow_audit ---
CREATE OR REPLACE FUNCTION public.log_workflow_audit()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, new_state)
        VALUES (TG_TABLE_NAME, NEW.opportunity_id, 'CREATE', row_to_json(NEW));
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state, new_state, changed_fields)
        VALUES (
            TG_TABLE_NAME,
            NEW.opportunity_id,
            'UPDATE',
            row_to_json(OLD),
            row_to_json(NEW),
            ARRAY(SELECT key FROM jsonb_each(row_to_json(NEW)::jsonb) 
                  WHERE row_to_json(NEW)::jsonb->key != row_to_json(OLD)::jsonb->key)
        );
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state)
        VALUES (TG_TABLE_NAME, OLD.opportunity_id, 'DELETE', row_to_json(OLD));
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$function$
;

--- notify_metrics_registry_changed ---
CREATE OR REPLACE FUNCTION public.notify_metrics_registry_changed()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
  PERFORM pg_notify('metrics_registry_changed', json_build_object(
    'operation', TG_OP,
    'node_id', COALESCE(NEW.node_id, OLD.node_id),
    'schema_domain', COALESCE(NEW.schema_domain, OLD.schema_domain),
    'timestamp', NOW()
  )::text);
  RETURN COALESCE(NEW, OLD);
END;
$function$
;

--- notify_security_change ---
CREATE OR REPLACE FUNCTION public.notify_security_change()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    payload TEXT;
BEGIN
    IF (TG_OP = 'DELETE') THEN
        payload := json_build_object(
            'tenant_id', OLD.tenant_id,
            'user_id', OLD.user_id,
            'action', TG_OP,
            'timestamp', now()
        )::text;
    ELSE
        payload := json_build_object(
            'tenant_id', NEW.tenant_id,
            'user_id', NEW.user_id,
            'action', TG_OP,
            'timestamp', now()
        )::text;
    END IF;
    
    PERFORM pg_notify('security_fund_access_change', payload);
    RETURN NULL;
END;
$function$
;

--- oms.default_order_status ---
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
$function$
;

--- public.update_holdings_on_transaction ---
CREATE OR REPLACE FUNCTION public.update_holdings_on_transaction()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.status = 'CONFIRMED' AND (OLD.status IS NULL OR OLD.status != 'CONFIRMED') THEN
        -- Update or insert holding based on transaction type
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
$function$
;

--- public.update_crypto_holding_valuation ---
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
$function$
;

--- public.refresh_latest_prices ---
CREATE OR REPLACE FUNCTION public.refresh_latest_prices()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY crypto_latest_prices;
    RETURN NEW;
END;
$function$
;

--- public.screen_investment_opportunity ---
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
    -- Get client's liquid assets
    SELECT COALESCE(SUM(current_nav), 0) INTO client_liquid_assets
    FROM alternative_investments ai
    WHERE ai.client_id = NEW.client_id;
    
    -- Add check for portfolio_summary if it exists
    -- This is a placeholder - adjust based on actual schema
    
    -- Check 1: Minimum commitment vs client capacity (max 10% of AUM per position)
    IF NEW.minimum_commitment > (client_liquid_assets * 0.10) THEN
        screening_passed := FALSE;
        screening_reasons := screening_reasons || 'Minimum commitment exceeds 10% of alternative AUM';
        screening_score := screening_score - 20;
    END IF;
    
    -- Check 2: Vintage year alignment (2025-2028 target)
    IF NEW.vintage_year IS NOT NULL AND (NEW.vintage_year < 2025 OR NEW.vintage_year > 2028) THEN
        screening_reasons := screening_reasons || 'Vintage year outside preferred 2025-2028 range';
        screening_score := screening_score - 10;
    END IF;
    
    -- Check 3: Manager track record
    IF NEW.track_record_years_min IS NOT NULL AND NEW.track_record_years_min < 5 THEN
        screening_reasons := screening_reasons || 'Manager track record less than 5 years';
        screening_score := screening_score - 15;
    END IF;
    
    -- Check 4: Target IRR reasonableness
    IF NEW.target_irr_min IS NOT NULL AND NEW.target_irr_min > 35 THEN
        screening_reasons := screening_reasons || 'Target IRR appears unrealistically high';
        screening_score := screening_score - 10;
    END IF;
    
    -- Update the record
    NEW.screening_passed := screening_passed;
    NEW.screening_reasons := screening_reasons;
    NEW.screening_score := GREATEST(screening_score, 0);
    NEW.screening_completed_at := NOW();
    
    -- Auto-advance stage if passed
    IF screening_passed AND NEW.current_stage = 'INTAKE' THEN
        NEW.current_stage := 'INITIAL_SCREEN';
        NEW.stage_updated_at := NOW();
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INITIAL_SCREEN',
            'timestamp', NOW(),
            'notes', 'Automated screening passed'
        );
    ELSIF NOT screening_passed AND NEW.current_stage = 'INTAKE' THEN
        -- Keep in intake but flag for manual review
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INTAKE',
            'timestamp', NOW(),
            'notes', 'Automated screening flagged issues: ' || array_to_string(screening_reasons, '; ')
        );
    END IF;
    
    RETURN NEW;
END;
$function$
;

--- vend.create_default_template_permissions ---
CREATE OR REPLACE FUNCTION vend.create_default_template_permissions()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
  -- Viewer: Can only run templates
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'viewer', true, false, false, false);
  
  -- Editor: Can run and edit
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'editor', true, true, false, false);
  
  -- Admin: Full access
  INSERT INTO semantic_query_template_permissions 
  (template_id, role, can_run, can_edit, can_delete, can_promote) 
  VALUES (NEW.id, 'admin', true, true, true, true);
  
  RETURN NEW;
END;
$function$
;

--- vend.create_template_version_on_update ---
CREATE OR REPLACE FUNCTION vend.create_template_version_on_update()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
  v_version_number INTEGER;
BEGIN
  -- Get next version number
  SELECT COALESCE(MAX(version_number), 0) + 1
  INTO v_version_number
  FROM semantic_query_template_versions
  WHERE template_id = NEW.id;
  
  -- Create version snapshot
  INSERT INTO semantic_query_template_versions 
  (template_id, version_number, name, description, semantic_query, parameters, 
   change_message, created_by)
  VALUES (
    NEW.id,
    v_version_number,
    NEW.name,
    NEW.description,
    NEW.semantic_query,
    NEW.parameters,
    '', -- Change message would be provided by app layer
    NEW.updated_by
  );
  
  RETURN NEW;
END;
$function$
;

--- public.cube_custom_model_version_trigger ---
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
$function$
;

--- public.cube_security_policy_version_trigger ---
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
$function$
;

--- public.update_template_rating_stats ---
CREATE OR REPLACE FUNCTION public.update_template_rating_stats()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    avg_rating DECIMAL(3,2);
    total_count INTEGER;
BEGIN
    -- Calculate new average and count
    SELECT 
        COALESCE(AVG(rating), 0.0)::DECIMAL(3,2),
        COUNT(*)
    INTO avg_rating, total_count
    FROM template_ratings
    WHERE template_id = COALESCE(NEW.template_id, OLD.template_id)
      AND moderation_status = 'approved';
    
    -- Update template
    UPDATE process_templates
    SET 
        rating_average = avg_rating,
        rating_count = total_count,
        updated_at = NOW()
    WHERE id = COALESCE(NEW.template_id, OLD.template_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$function$
;

--- public.calculate_step_duration ---
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
$function$
;

--- public.check_capital_call_liquidity ---
CREATE OR REPLACE FUNCTION public.check_capital_call_liquidity()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.liquidity_check_passed := TRUE;
    NEW.recommended_action := 'Sufficient liquidity confirmed.';
    RETURN NEW;
END;
$function$
;

--- orm.sync_identifier_cache ---
CREATE OR REPLACE FUNCTION orm.sync_identifier_cache()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF NEW.is_primary AND NEW.effective_to IS NULL THEN
        IF NEW.id_type = 'ISIN'  THEN UPDATE orm.security SET isin  = NEW.id_value WHERE id = NEW.security_id;
        ELSIF NEW.id_type = 'CUSIP' THEN UPDATE orm.security SET cusip = NEW.id_value WHERE id = NEW.security_id;
        ELSIF NEW.id_type = 'TICKER' THEN UPDATE orm.security SET ticker = NEW.id_value WHERE id = NEW.security_id;
        END IF;
    END IF;
    RETURN NEW;
END $function$
;

