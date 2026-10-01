-- preflight2_crims.sql
-- Read-only. Enumerates crims' existing inventories for collision detection.
\set ON_ERROR_STOP on
\timing on

\echo '=== 0.5.7 crims.mdm full table list (83 tables) ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'mdm'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.8 crims.mdm tables grouped by prefix ==='
SELECT
    split_part(table_name, '_', 1) AS prefix,
    count(*)                       AS tables
FROM information_schema.tables
WHERE table_schema = 'mdm'
  AND table_type = 'BASE TABLE'
GROUP BY prefix
ORDER BY tables DESC, prefix;

\echo '=== 0.5.9 crims.cash_flow inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'cash_flow'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.10 crims.ref inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'ref'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.11 crims.vend inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'vend'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.12 crims.wlth inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'wlth'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.13 crims.orm inventory ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'orm'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.14 crims.public inventory (collision check for fabric) ==='
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
ORDER BY table_name;

\echo '=== 0.5.15 CRITICAL: collision surface ==='
-- Which crims.mdm tables share names with the incoming alpha.mdm data-plane set?
WITH incoming AS (
    SELECT unnest(ARRAY[
        'product','product_identifier','product_share_class','product_fee_schedule',
        'product_share_class_fee','product_platform','product_distribution',
        'product_registration','product_eligibility','product_target_market',
        'product_document','product_vehicle','product_security','product_benchmark',
        'product_strategy','product_lifecycle_event','product_golden_record',
        'product_golden_field','product_survivorship_log','product_match_candidate',
        'product_merge_log','product_exception','product_change_request',
        'product_feed_health','product_steward',
        'counterparty','counterparty_identifier','counterparty_agreement',
        'counterparty_agreement_term','counterparty_credit_limit',
        'counterparty_credit_profile','counterparty_due_diligence',
        'counterparty_exposure','counterparty_settlement_instruction',
        'counterparty_role_assignment','counterparty_golden_record',
        'counterparty_golden_field','counterparty_survivorship_log',
        'counterparty_match_candidate','counterparty_merge_log',
        'counterparty_exception','counterparty_change_request','counterparty_steward',
        'benchmark_master','benchmark_identifier','benchmark_hierarchy',
        'benchmark_hierarchy_closure','benchmark_golden_record',
        'benchmark_golden_field','benchmark_golden_publication',
        'benchmark_golden_distribution','benchmark_survivorship_log',
        'benchmark_match_candidate','benchmark_merge_log','benchmark_exception',
        'benchmark_change_request','benchmark_feed_health','benchmark_steward',
        'benchmark_reconciliation','benchmark_reconciliation_result',
        'calendar_master','calendar_day','calendar_session','calendar_identifier',
        'calendar_classification','calendar_rule','calendar_hierarchy',
        'calendar_hierarchy_closure','calendar_historical_closure',
        'calendar_special_event','calendar_special_event_impact',
        'holiday_definition','holiday_manual_date','holiday_definition_calendar',
        'fund_calendar','fund_calendar_exception','fund_calendar_rule',
        'central_bank_calendar','settlement_calendar','settlement_rule',
        'settlement_rule_calendar','calendar_golden_record','calendar_golden_field',
        'calendar_survivorship_log','calendar_match_candidate','calendar_merge_log',
        'calendar_exception','calendar_change_request','calendar_reconciliation',
        'calendar_steward',
        'ca_event','ca_identifier','ca_term','ca_security','ca_cash_component',
        'ca_stock_component','ca_election_option','ca_election_deadline',
        'ca_election_instruction','ca_agenda_item','ca_meeting','ca_proxy_vote',
        'ca_bond_call','ca_bond_put','ca_conversion','ca_consent_solicitation',
        'ca_default_event','ca_sinking_fund','ca_ratio_history',
        'ca_fund_distribution','ca_fund_liquidation','ca_fund_reorganization',
        'ca_golden_record','ca_golden_field','ca_survivorship_log',
        'ca_match_candidate','ca_merge_log','ca_exception','ca_change_request',
        'price','price_series','price_history','price_vendor_symbol',
        'price_source_alias','price_challenge','price_exception','price_feed_health',
        'price_golden_record','price_golden_field','price_golden_publication',
        'price_golden_distribution','price_survivorship_log','price_match_candidate',
        'price_merge_log','price_variance_event','price_change_request',
        'price_stale_event','price_steward','price_reconciliation',
        'price_reconciliation_result','curve_master','curve_point','curve_snapshot',
        'vol_surface','vol_surface_point','fx_rate_master','valuation_input',
        'valuation_sensitivity','fair_value_classification',
        'rating','rating_action','rating_bank','rating_default','rating_fund',
        'rating_insurance','rating_internal','rating_proprietary','rating_scale_map',
        'rating_scale',
        'party','party_name','party_address','party_contact','party_identifier',
        'party_individual','party_legal_entity','party_history',
        'party_relationship','party_control','party_ownership','party_hierarchy',
        'party_hierarchy_closure','party_ubo','party_role_assignment',
        'party_successor','party_tax_residency','party_consent',
        'party_golden_record','party_golden_field','party_survivorship_log',
        'party_match_candidate','party_merge_log','party_exception',
        'party_change_request','party_steward',
        'client_group','client_group_hierarchy','client_group_hierarchy_closure',
        'client_group_membership','client_group_servicing','client_group_user',
        'issuer_master'
    ]) AS table_name
)
SELECT i.table_name AS incoming_table,
       CASE WHEN c.relname IS NOT NULL THEN 'COLLIDES' ELSE 'clear' END AS status
FROM incoming i
LEFT JOIN pg_class c ON c.relname = i.table_name
                     AND c.relnamespace = 'mdm'::regnamespace
                     AND c.relkind = 'r'
WHERE c.relname IS NOT NULL
ORDER BY i.table_name;
