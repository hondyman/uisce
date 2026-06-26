-- =============================================
-- Master Script: Create All Financial Services Views
-- Generated from DAX-to-SQL mapping
-- =============================================

-- Drop existing views (uncomment if needed)
-- DROP VIEW IF EXISTS net_interest_margin;
-- ... add other DROP statements as needed

CREATE VIEW net_interest_margin AS SELECT (SUM(ii.amount) - SUM(ie.amount)) / AVG(a.average_balance) AS value FROM interest_income ii, interest_expense ie, assets a GROUP BY entity_id, as_of_date
CREATE VIEW loan_loss_provision_ratio AS SELECT SUM(lp.provision_amount) / SUM(l.outstanding_balance) AS value FROM loan_provisions lp JOIN loans l ON lp.entity_id = l.entity_id AND lp.as_of_date = l.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW credit_risk_concentration AS SELECT SUM(CASE WHEN l.principal_amount > ct.large_borrower_threshold THEN l.principal_amount ELSE 0 END) / SUM(l.principal_amount) AS value FROM loans l CROSS JOIN (SELECT large_borrower_threshold FROM concentration_thresholds) ct GROUP BY entity_id, as_of_date
CREATE VIEW deposit_stability_ratio AS SELECT SUM(CASE WHEN d.stability_classification = 'stable' THEN d.balance ELSE 0 END) / SUM(d.balance) AS value FROM deposits d GROUP BY entity_id, as_of_date
CREATE VIEW cost_to_income_ratio AS SELECT SUM(e.amount) / SUM(i.amount) AS value FROM operating_expenses e JOIN operating_income i ON e.entity_id = i.entity_id AND e.as_of_date = i.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW return_on_assets AS SELECT SUM(i.net_income) / AVG(a.total_assets) AS value FROM income i JOIN assets a ON i.entity_id = a.entity_id AND i.as_of_date = a.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW return_on_equity AS SELECT SUM(i.net_income) / AVG(e.total_equity) AS value FROM income i JOIN equity e ON i.entity_id = e.entity_id AND i.as_of_date = e.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW loan_to_deposit_ratio AS SELECT SUM(l.outstanding_balance) / SUM(d.balance) AS value FROM loans l JOIN deposits d ON l.entity_id = d.entity_id AND l.as_of_date = d.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW non_performing_loan_ratio AS SELECT SUM(CASE WHEN l.performance_status = 'non_performing' THEN l.outstanding_balance ELSE 0 END) / SUM(l.outstanding_balance) AS value FROM loans l GROUP BY entity_id, as_of_date
CREATE VIEW capital_adequacy_ratio AS SELECT SUM(rc.amount) / SUM(rwa.weighted_balance) AS value FROM regulatory_capital rc JOIN risk_weighted_assets rwa ON rc.entity_id = rwa.entity_id AND rc.as_of_date = rwa.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW combined_ratio AS SELECT (SUM(c.amount) + SUM(ue.amount)) / SUM(ep.amount) AS value FROM claims c JOIN underwriting_expenses ue ON c.entity_id = ue.entity_id AND c.as_of_date = ue.as_of_date JOIN earned_premiums ep ON c.entity_id = ep.entity_id AND c.as_of_date = ep.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW loss_ratio AS SELECT SUM(c.amount) / SUM(ep.amount) AS value FROM claims c JOIN earned_premiums ep ON c.entity_id = ep.entity_id AND c.as_of_date = ep.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW expense_ratio AS SELECT SUM(ue.amount) / SUM(wp.amount) AS value FROM underwriting_expenses ue JOIN written_premiums wp ON ue.entity_id = wp.entity_id AND ue.as_of_date = wp.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW underwriting_profit AS SELECT SUM(ep.amount) - SUM(c.amount) - SUM(ue.amount) AS value FROM earned_premiums ep JOIN claims c ON ep.entity_id = c.entity_id AND ep.as_of_date = c.as_of_date JOIN underwriting_expenses ue ON ep.entity_id = ue.entity_id AND ep.as_of_date = ue.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW retention_ratio AS SELECT SUM(p.net_amount) / SUM(p.gross_amount) AS value FROM premiums p GROUP BY entity_id, as_of_date
CREATE VIEW claims_frequency AS SELECT COUNT(DISTINCT c.id) / SUM(p.exposure_amount) AS value FROM claims c JOIN policies p ON c.entity_id = p.entity_id AND c.as_of_date = p.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW claims_severity AS SELECT SUM(c.amount) / COUNT(DISTINCT c.id) AS value FROM claims c GROUP BY entity_id, as_of_date
CREATE VIEW reserve_adequacy_ratio AS SELECT SUM(lr.amount) / SUM(ap.projected_amount) AS value FROM loss_reserves lr JOIN actuarial_projections ap ON lr.entity_id = ap.entity_id AND lr.as_of_date = ap.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW policyholder_surplus_ratio AS SELECT SUM(ps.amount) / SUM(l.amount) AS value FROM policyholder_surplus ps JOIN liabilities l ON ps.entity_id = l.entity_id AND ps.as_of_date = l.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW value_at_risk AS SELECT STDDEV_POP(pr.daily_return) * SQRT(hp.holding_days) * cf.confidence_multiplier AS value FROM portfolio_returns pr CROSS JOIN (SELECT holding_days FROM holding_periods LIMIT 1) hp CROSS JOIN (SELECT confidence_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY entity_id, as_of_date
CREATE VIEW sharpe_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / STDDEV_POP(pr.daily_return) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates LIMIT 1) rfr GROUP BY entity_id, as_of_date
CREATE VIEW portfolio_turnover AS SELECT SUM(ABS(t.transaction_amount)) / (2 * AVG(pv.total_value)) AS value FROM trades t JOIN portfolio_values pv ON t.entity_id = pv.entity_id AND t.as_of_date = pv.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW beta_coefficient AS SELECT SUM((pmr.portfolio_return - pm.portfolio_average) * (pmr.market_return - mm.market_average)) / SUM(POWER(pmr.market_return - mm.market_average, 2)) AS value FROM portfolio_market_returns pmr CROSS JOIN (SELECT AVG(portfolio_return) as portfolio_average FROM portfolio_market_returns) pm CROSS JOIN (SELECT AVG(market_return) as market_average FROM portfolio_market_returns) mm GROUP BY entity_id, as_of_date
CREATE VIEW tracking_error AS SELECT STDDEV_POP(ar.active_return) AS value FROM active_returns ar GROUP BY entity_id, as_of_date
CREATE VIEW information_ratio AS SELECT AVG(ar.active_return) / STDDEV_POP(ar.active_return) AS value FROM active_returns ar GROUP BY entity_id, as_of_date
CREATE VIEW maximum_drawdown AS SELECT MIN(dd.drawdown) AS value FROM rolling_drawdowns dd GROUP BY entity_id, as_of_date
CREATE VIEW sortino_ratio AS SELECT (AVG(pr.daily_return) - rfr.risk_free_rate) / STDDEV_POP(CASE WHEN pr.daily_return < rfr.risk_free_rate THEN pr.daily_return ELSE NULL END) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates LIMIT 1) rfr GROUP BY entity_id, as_of_date
CREATE VIEW alpha_coefficient AS SELECT AVG(pr.daily_return) - (rfr.risk_free_rate + b.beta_coefficient * (AVG(mr.daily_return) - rfr.risk_free_rate)) AS value FROM portfolio_returns pr CROSS JOIN (SELECT risk_free_rate FROM risk_free_rates LIMIT 1) rfr CROSS JOIN (SELECT beta_coefficient FROM beta_coefficients LIMIT 1) b CROSS JOIN market_returns mr GROUP BY entity_id, as_of_date
CREATE VIEW win_rate AS SELECT COUNT(DISTINCT CASE WHEN t.profit_loss > 0 THEN t.id ELSE NULL END) / COUNT(DISTINCT t.id) AS value FROM trades t GROUP BY entity_id, as_of_date
CREATE VIEW liquidity_coverage_ratio AS SELECT SUM(hqla.amount) / SUM(nco.amount) AS value FROM high_quality_liquid_assets hqla JOIN net_cash_outflows nco ON hqla.entity_id = nco.entity_id AND hqla.as_of_date = nco.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW net_stable_funding_ratio AS SELECT SUM(asf.amount) / SUM(rsf.amount) AS value FROM available_stable_funding asf JOIN required_stable_funding rsf ON asf.entity_id = rsf.entity_id AND asf.as_of_date = rsf.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW tier1_capital_ratio AS SELECT SUM(t1c.amount) / SUM(rwa.weighted_amount) AS value FROM tier1_capital t1c JOIN risk_weighted_assets rwa ON t1c.entity_id = rwa.entity_id AND t1c.as_of_date = rwa.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW leverage_ratio AS SELECT SUM(t1c.amount) / SUM(te.amount) AS value FROM tier1_capital t1c JOIN total_exposure te ON t1c.entity_id = te.entity_id AND t1c.as_of_date = te.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW large_exposure_limit AS SELECT MAX(ce.amount) / SUM(ec.amount) AS value FROM counterparty_exposures ce JOIN eligible_capital ec ON ce.entity_id = ec.entity_id AND ce.as_of_date = ec.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW market_risk_capital AS SELECT SUM(tp.var_amount) * cf.risk_multiplier AS value FROM trading_positions tp CROSS JOIN (SELECT risk_multiplier FROM confidence_factors LIMIT 1) cf GROUP BY entity_id, as_of_date
CREATE VIEW patient_satisfaction_score AS SELECT AVG(ps.rating) AS value FROM patient_surveys ps GROUP BY entity_id, as_of_date
CREATE VIEW average_length_of_stay AS SELECT AVG(ha.days) AS value FROM hospital_admissions ha GROUP BY entity_id, as_of_date
CREATE VIEW readmission_rate AS SELECT SUM(r.count) / SUM(pd.count) AS value FROM readmissions r JOIN patient_discharges pd ON r.entity_id = pd.entity_id AND r.as_of_date = pd.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW bed_occupancy_rate AS SELECT SUM(bo.count) / SUM(bc.count) AS value FROM bed_occupancy bo JOIN bed_capacity bc ON bo.entity_id = bc.entity_id AND bo.as_of_date = bc.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW average_cost_per_patient AS SELECT SUM(tc.amount) / SUM(pc.count) AS value FROM treatment_costs tc JOIN patient_counts pc ON tc.entity_id = pc.entity_id AND tc.as_of_date = pc.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW infection_rate AS SELECT (SUM(hi.count) / SUM(pd.days)) * 1000 AS value FROM hospital_infections hi JOIN patient_days pd ON hi.entity_id = pd.entity_id AND hi.as_of_date = pd.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW physician_productivity AS SELECT SUM(pv.count) / SUM(ap.count) AS value FROM patient_visits pv JOIN active_physicians ap ON pv.entity_id = ap.entity_id AND pv.as_of_date = ap.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW medication_error_rate AS SELECT (SUM(me.count) / SUM(da.count)) * 1000 AS value FROM medication_errors me JOIN doses_administered da ON me.entity_id = da.entity_id AND me.as_of_date = da.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW emergency_wait_time AS SELECT AVG(ewt.minutes) AS value FROM emergency_wait_times ewt GROUP BY entity_id, as_of_date
CREATE VIEW revenue_per_bed AS SELECT SUM(hr.amount) / SUM(ab.count) AS value FROM hospital_revenue hr JOIN available_beds ab ON hr.entity_id = ab.entity_id AND hr.as_of_date = ab.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW sales_per_square_foot AS SELECT SUM(ss.revenue) / SUM(sd.area_sqft) AS value FROM store_sales ss JOIN store_dimensions sd ON ss.entity_id = sd.entity_id AND ss.as_of_date = sd.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW inventory_turnover_ratio AS SELECT SUM(s.cogs) / AVG(il.value) AS value FROM sales_data s JOIN inventory_levels il ON s.entity_id = il.entity_id AND s.as_of_date = il.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW customer_acquisition_cost AS SELECT SUM(me.amount) / SUM(nca.count) AS value FROM marketing_expenses me JOIN new_customer_acquisitions nca ON me.entity_id = nca.entity_id AND me.as_of_date = nca.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW gross_margin_percentage AS SELECT SUM(sr.profit) / SUM(sr.revenue) AS value FROM sales_revenue sr GROUP BY entity_id, as_of_date
CREATE VIEW average_transaction_value AS SELECT SUM(ct.amount) / SUM(ct.count) AS value FROM customer_transactions ct GROUP BY entity_id, as_of_date
CREATE VIEW customer_lifetime_value AS SELECT SUM(cs.avg_value * cs.frequency * cs.years) AS value FROM customer_segments cs GROUP BY entity_id, as_of_date
CREATE VIEW shrinkage_rate AS SELECT SUM(isv.value) / SUM(iv.amount) AS value FROM inventory_shrinkage isv JOIN inventory_value iv ON isv.entity_id = iv.entity_id AND isv.as_of_date = iv.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW conversion_rate AS SELECT SUM(sp.count) / SUM(st.count) AS value FROM store_purchases sp JOIN store_traffic st ON sp.entity_id = st.entity_id AND sp.as_of_date = st.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW employee_productivity AS SELECT SUM(ss.revenue) / SUM(eh.hours) AS value FROM store_sales ss JOIN employee_hours eh ON ss.entity_id = eh.entity_id AND ss.as_of_date = eh.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW seasonal_sales_index AS SELECT SUM(cps.revenue) / AVG(sa.avg_revenue) AS value FROM current_period_sales cps JOIN seasonal_averages sa ON cps.entity_id = sa.entity_id AND cps.as_of_date = sa.as_of_date GROUP BY entity_id, as_of_date
CREATE VIEW effective_interest_income AS SELECT oca.opening_carrying_amount * eir.effective_interest_rate AS value FROM opening_carrying_amounts oca JOIN effective_interest_rates eir ON oca.entity_id = eir.entity_id AND oca.as_of_date = eir.as_of_date
CREATE VIEW premium_discount_amortization AS SELECT (eii.value - ccr.coupon_cash_received) AS value FROM effective_interest_income eii JOIN coupon_cash_received ccr ON eii.entity_id = ccr.entity_id AND eii.as_of_date = ccr.as_of_date
CREATE VIEW carrying_amount_rollforward AS SELECT oca.opening_carrying_amount + eii.value - ccr.coupon_cash_received + pda.value - clac.ecl_closing + clac.ecl_opening + fr.fx_remeasurement AS value FROM opening_carrying_amounts oca JOIN effective_interest_income eii ON oca.entity_id = eii.entity_id AND oca.as_of_date = eii.as_of_date JOIN coupon_cash_received ccr ON oca.entity_id = ccr.entity_id AND oca.as_of_date = ccr.as_of_date JOIN premium_discount_amortization pda ON oca.entity_id = pda.entity_id AND oca.as_of_date = pda.as_of_date JOIN credit_loss_allowance_change clac ON oca.entity_id = clac.entity_id AND oca.as_of_date = clac.as_of_date JOIN fx_remeasurement fr ON oca.entity_id = fr.entity_id AND oca.as_of_date = fr.as_of_date
CREATE VIEW fair_value_change_pnl_trading AS SELECT fvc.fair_value_current - fvp.fair_value_prior AS value FROM fair_value_current fvc JOIN fair_value_prior fvp ON fvc.entity_id = fvp.entity_id AND fvc.as_of_date = fvp.as_of_date
CREATE VIEW fair_value_change_oci_afs AS SELECT fvc.fair_value_current - acc.value AS value FROM fair_value_current fvc JOIN carrying_amount_rollforward acc ON fvc.entity_id = acc.entity_id AND fvc.as_of_date = acc.as_of_date
CREATE VIEW realised_gain_loss_disposal AS SELECT cr.consideration_received - cas.carrying_amount_at_sale AS value FROM consideration_received cr JOIN carrying_amount_at_sale cas ON cr.entity_id = cas.entity_id AND cr.as_of_date = cas.as_of_date
CREATE VIEW credit_loss_allowance_change AS SELECT eclc.ecl_closing - ecli.ecl_opening AS value FROM ecl_closing eclc JOIN ecl_opening ecli ON eclc.entity_id = ecli.entity_id AND eclc.as_of_date = ecli.as_of_date
CREATE VIEW fx_remeasurement AS SELECT (fxc.fx_rate_closing - fxa.fx_rate_avg_period) * mbf.monetary_balance_foreign AS value FROM fx_rate_closing fxc JOIN fx_rate_avg_period fxa ON fxc.entity_id = fxa.entity_id AND fxc.as_of_date = fxa.as_of_date JOIN monetary_balance_foreign mbf ON fxc.entity_id = mbf.entity_id AND fxc.as_of_date = mbf.as_of_date
CREATE VIEW fx_translation_reserve_oci AS SELECT COALESCE(ctap.cta_prior, 0) + ctam.cta_movement AS value FROM cta_prior ctap JOIN cta_movement ctam ON ctap.entity_id = ctam.entity_id AND ctap.as_of_date = ctam.as_of_date
CREATE VIEW equity_method_share_of_profit AS SELECT ini.investee_net_income * op.ownership_pct AS value FROM investee_net_income ini JOIN ownership_pct op ON ini.entity_id = op.entity_id AND ini.as_of_date = op.as_of_date
CREATE VIEW equity_method_carrying_value AS SELECT ocv.opening_carrying_value + emsop.value - dr.dividends_received + oca.oci_adjustment + ftr.value AS value FROM opening_carrying_value ocv JOIN equity_method_share_of_profit emsop ON ocv.entity_id = emsop.entity_id AND ocv.as_of_date = emsop.as_of_date JOIN dividends_received dr ON ocv.entity_id = dr.entity_id AND ocv.as_of_date = dr.as_of_date JOIN oci_adjustment oca ON ocv.entity_id = oca.entity_id AND ocv.as_of_date = oca.as_of_date JOIN fx_translation_reserve_oci ftr ON ocv.entity_id = ftr.entity_id AND ocv.as_of_date = ftr.as_of_date
CREATE VIEW nci_share_of_profit AS SELECT sp.subsidiary_profit * np.nci_pct AS value FROM subsidiary_profit sp JOIN nci_pct np ON sp.entity_id = np.entity_id AND sp.as_of_date = np.as_of_date
CREATE VIEW impairment_loss_incurred AS SELECT GREATEST(0, ca.carrying_amount - ra.recoverable_amount) AS value FROM carrying_amount ca JOIN recoverable_amount ra ON ca.entity_id = ra.entity_id AND ca.as_of_date = ra.as_of_date
CREATE VIEW dividend_income_accrual AS SELECT SUM(de.dividend_amount) AS value FROM dividend_events de WHERE de.ex_date <= de.period_end GROUP BY entity_id, as_of_date
CREATE VIEW fair_value_hierarchy_exposure AS SELECT SUM(p.fair_value) AS value FROM positions p WHERE p.fv_hierarchy_level = (SELECT level_filter FROM level_filters LIMIT 1) GROUP BY entity_id, as_of_date
CREATE VIEW oci_recycling_on_disposal AS SELECT LEAST(aocb.accumulated_oci_balance, rgld.value) AS value FROM accumulated_oci_balance aocb JOIN realised_gain_loss_disposal rgld ON aocb.entity_id = rgld.entity_id AND aocb.as_of_date = rgld.as_of_date
CREATE VIEW spot_conversion AS SELECT asc.amount_source_currency * sfr.spot_fx_rate AS value FROM amount_source_currency asc JOIN spot_fx_rate sfr ON asc.entity_id = sfr.entity_id AND asc.as_of_date = sfr.as_of_date
CREATE VIEW average_period_rate AS SELECT AVG(fr.fx_rate) AS value FROM fx_rates fr WHERE fr.date >= fr.period_start AND fr.date <= fr.period_end GROUP BY entity_id, as_of_date
CREATE VIEW closing_rate_translation AS SELECT bfc.balance_foreign_currency * frc.fx_rate_closing AS value FROM balance_foreign_currency bfc JOIN fx_rate_closing frc ON bfc.entity_id = frc.entity_id AND bfc.as_of_date = frc.as_of_date
CREATE VIEW net_fx_exposure AS SELECT fa.fx_assets - fl.fx_liabilities AS value FROM fx_assets fa JOIN fx_liabilities fl ON fa.entity_id = fl.entity_id AND fa.as_of_date = fl.as_of_date
CREATE VIEW open_fx_position AS SELECT nfe.value - ha.hedged_amount AS value FROM net_fx_exposure nfe JOIN hedged_amount ha ON nfe.entity_id = ha.entity_id AND nfe.as_of_date = ha.as_of_date
CREATE VIEW fx_delta AS SELECT pv.portfolio_value * 0.01 AS value FROM portfolio_value pv
CREATE VIEW currency_contribution_to_return AS SELECT prb.portfolio_return_in_base_currency - prl.portfolio_return_in_local_currency AS value FROM portfolio_return_in_base_currency prb JOIN portfolio_return_in_local_currency prl ON prb.entity_id = prl.entity_id AND prb.as_of_date = prl.as_of_date
CREATE VIEW hedging_effectiveness AS SELECT hpl.hedge_profit_loss / epl.exposure_profit_loss AS value FROM hedge_profit_loss hpl JOIN exposure_profit_loss epl ON hpl.entity_id = epl.entity_id AND hpl.as_of_date = epl.as_of_date
CREATE VIEW cta_balance AS SELECT COALESCE(ctap.cta_prior, 0) + ctam.cta_movement AS value FROM cta_prior ctap JOIN cta_movement ctam ON ctap.entity_id = ctam.entity_id AND ctap.as_of_date = ctam.as_of_date
CREATE VIEW realised_fx_gain_loss AS SELECT sab.settlement_amount_base - cab.contracted_amount_base AS value FROM settlement_amount_base sab JOIN contracted_amount_base cab ON sab.entity_id = cab.entity_id AND sab.as_of_date = cab.as_of_date
CREATE VIEW unrealised_fx_gain_loss AS SELECT (cfr.current_fx_rate - pfr.prior_fx_rate) * opf.open_position_foreign AS value FROM current_fx_rate cfr JOIN prior_fx_rate pfr ON cfr.entity_id = pfr.entity_id AND cfr.as_of_date = pfr.as_of_date JOIN open_position_foreign opf ON cfr.entity_id = opf.entity_id AND cfr.as_of_date = opf.as_of_date

-- =============================================
-- Verification Queries
-- =============================================

-- Example: Check if views were created successfully
-- SELECT table_name FROM information_schema.views WHERE table_schema = 'public';

-- Example: Test a specific view
-- SELECT * FROM net_interest_margin LIMIT 5;

