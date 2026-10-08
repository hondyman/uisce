package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/lib/pq"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DeriveDomain derives the functional regulatory domain from rule_code prefix
func DeriveDomain(ruleCode string) string {
	rc := strings.ToUpper(ruleCode)
	switch {
	case strings.HasPrefix(rc, "UCITS_") || strings.HasPrefix(rc, "ACT40_") || strings.HasPrefix(rc, "SINGLE_POSITION") || strings.HasPrefix(rc, "GROUP_CONCENTRATION") || strings.HasPrefix(rc, "TOTAL_ISSUER_EXPOSURE") || strings.HasPrefix(rc, "BENCHMARK_TRACKING_ERROR"):
		return "Concentration & Diversification"
	case strings.HasPrefix(rc, "LIQUIDITY_") || strings.HasPrefix(rc, "ILLIQUID_") || strings.HasPrefix(rc, "SETTLEMENT_"):
		return "Liquidity & Settlement"
	case strings.HasPrefix(rc, "DERIVATIVE_") || strings.HasPrefix(rc, "LEVERAGE_") || strings.HasPrefix(rc, "COMMITMENT_") || strings.HasPrefix(rc, "MARGIN_") || strings.HasPrefix(rc, "COUNTERPARTY_"):
		return "Derivatives & Leverage"
	case strings.HasPrefix(rc, "RESTRICTED_") || strings.HasPrefix(rc, "SANCTIONS_") || strings.HasPrefix(rc, "INSIDER_") || strings.HasPrefix(rc, "VENUE_"):
		return "Restricted Lists & Insider Lists"
	case strings.HasPrefix(rc, "ORDER_RATE_") || strings.HasPrefix(rc, "LAYERING_") || strings.HasPrefix(rc, "SPOOFING_") || strings.HasPrefix(rc, "WASH_SALE_") || strings.HasPrefix(rc, "MARKET_ABUSE_") || strings.HasPrefix(rc, "REG_M_") || strings.HasPrefix(rc, "SHORT_TENDER_") || strings.HasPrefix(rc, "EXECUTION_WITHIN_SPREAD") || strings.HasPrefix(rc, "LARGE_TRADE_"):
		return "Market Abuse & Order Surveillance"
	case strings.HasPrefix(rc, "PRO_RATA_") || strings.HasPrefix(rc, "AGGREGATION_") || strings.HasPrefix(rc, "PT_"):
		return "Fair Allocation & Personal Trading"
	case strings.HasPrefix(rc, "EMIR_") || strings.HasPrefix(rc, "SFTR_") || strings.HasPrefix(rc, "TXN_REPORT_") || strings.HasPrefix(rc, "LARGE_EXPOSURE_REPORT"):
		return "Transaction Reporting"
	case strings.HasPrefix(rc, "CROSS_BORDER_") || strings.HasPrefix(rc, "COUNTRY_") || strings.HasPrefix(rc, "FOF_") || strings.HasPrefix(rc, "SEC_144A_") || strings.HasPrefix(rc, "REG_S_") || strings.HasPrefix(rc, "FX_SETTLEMENT_"):
		return "Cross-Border & Asset Eligibility"
	default:
		return "General Compliance"
	}
}

// ListCoreRules returns a filterable list of all 50 Gold-Copy Core rules
func (s *Service) ListCoreRules(ctx context.Context, filter ListRulesFilter) ([]CoreRuleSummary, error) {
	query := `
		SELECT 
			r.id,
			r.rule_code,
			r.name,
			COALESCE(r.description, ''),
			r.rule_phase,
			r.severity,
			r.priority,
			r.jurisdictions,
			COALESCE(r.citation, ''),
			r.current_version,
			v.content_hash,
			r.library_status,
			r.effective_from,
			r.effective_to,
			COALESCE(array_agg(m.ruleset_code) FILTER (WHERE m.ruleset_code IS NOT NULL), '{}') as ruleset_codes
		FROM compliance.compliance_rule r
		JOIN public.tenants t ON r.tenant_id = t.id AND t.gold_copy = true
		JOIN compliance.compliance_rule_version v ON r.id = v.rule_id AND r.current_version = v.version
		LEFT JOIN compliance.compliance_ruleset_membership m ON r.id = m.rule_id
		WHERE r.valid_to IS NULL
	`

	args := make([]interface{}, 0)
	argIdx := 1

	if filter.LibraryStatus != "" && filter.LibraryStatus != "ALL" {
		query += fmt.Sprintf(" AND r.library_status = $%d", argIdx)
		args = append(args, filter.LibraryStatus)
		argIdx++
	}
	if filter.RulePhase != "" && filter.RulePhase != "ALL" {
		query += fmt.Sprintf(" AND r.rule_phase = $%d", argIdx)
		args = append(args, filter.RulePhase)
		argIdx++
	}
	if filter.Severity != "" && filter.Severity != "ALL" {
		query += fmt.Sprintf(" AND r.severity = $%d", argIdx)
		args = append(args, filter.Severity)
		argIdx++
	}

	query += `
		GROUP BY r.id, r.rule_code, r.name, r.description, r.rule_phase, r.severity, r.priority, r.jurisdictions, r.citation, r.current_version, v.content_hash, r.library_status, r.effective_from, r.effective_to
		ORDER BY r.priority ASC, r.rule_code ASC
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query core rules: %w", err)
	}
	defer rows.Close()

	rules := make([]CoreRuleSummary, 0, 50)
	for rows.Next() {
		var r CoreRuleSummary
		var jurisdictions, rulesetCodes []string

		err := rows.Scan(
			&r.ID,
			&r.RuleCode,
			&r.Name,
			&r.Description,
			&r.RulePhase,
			&r.Severity,
			&r.Priority,
			pq.Array(&jurisdictions),
			&r.Citation,
			&r.CurrentVersion,
			&r.ContentHash,
			&r.LibraryStatus,
			&r.EffectiveFrom,
			&r.EffectiveTo,
			pq.Array(&rulesetCodes),
		)
		if err != nil {
			return nil, fmt.Errorf("scan core rule: %w", err)
		}

		r.Jurisdictions = jurisdictions
		r.RulesetCodes = rulesetCodes
		r.Domain = DeriveDomain(r.RuleCode)

		// Post-filter Domain, Ruleset, Jurisdiction, and SearchQuery
		if filter.Domain != "" && filter.Domain != "ALL" && r.Domain != filter.Domain {
			continue
		}
		if filter.RulesetCode != "" && filter.RulesetCode != "ALL" {
			hasRuleset := false
			for _, rc := range r.RulesetCodes {
				if rc == filter.RulesetCode {
					hasRuleset = true
					break
				}
			}
			if !hasRuleset {
				continue
			}
		}
		if filter.Jurisdiction != "" && filter.Jurisdiction != "ALL" {
			hasJur := false
			for _, j := range r.Jurisdictions {
				if strings.EqualFold(j, filter.Jurisdiction) {
					hasJur = true
					break
				}
			}
			if !hasJur {
				continue
			}
		}
		if filter.SearchQuery != "" {
			q := strings.ToLower(filter.SearchQuery)
			if !strings.Contains(strings.ToLower(r.RuleCode), q) &&
				!strings.Contains(strings.ToLower(r.Name), q) &&
				!strings.Contains(strings.ToLower(r.Description), q) &&
				!strings.Contains(strings.ToLower(r.Citation), q) {
				continue
			}
		}

		rules = append(rules, r)
	}

	return rules, nil
}

// GetRuleDetails returns complete metadata, AST, parameters, version timeline, and test vectors for a rule
func (s *Service) GetRuleDetails(ctx context.Context, ruleID uuid.UUID) (*RuleDetails, error) {
	var details RuleDetails
	var astBytes, paramBytes []byte
	var jurisdictions, rulesetCodes []string

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			r.id,
			r.rule_code,
			r.name,
			COALESCE(r.description, ''),
			r.rule_phase,
			r.severity,
			r.priority,
			r.jurisdictions,
			COALESCE(r.citation, ''),
			r.current_version,
			v.content_hash,
			r.library_status,
			r.effective_from,
			r.effective_to,
			r.ast_condition,
			r.parameter_thresholds,
			COALESCE(array_agg(m.ruleset_code) FILTER (WHERE m.ruleset_code IS NOT NULL), '{}') as ruleset_codes
		FROM compliance.compliance_rule r
		JOIN compliance.compliance_rule_version v ON r.id = v.rule_id AND r.current_version = v.version
		LEFT JOIN compliance.compliance_ruleset_membership m ON r.id = m.rule_id
		WHERE r.id = $1 AND r.valid_to IS NULL
		GROUP BY r.id, r.rule_code, r.name, r.description, r.rule_phase, r.severity, r.priority, r.jurisdictions, r.citation, r.current_version, v.content_hash, r.library_status, r.effective_from, r.effective_to, r.ast_condition, r.parameter_thresholds
	`, ruleID).Scan(
		&details.ID,
		&details.RuleCode,
		&details.Name,
		&details.Description,
		&details.RulePhase,
		&details.Severity,
		&details.Priority,
		pq.Array(&jurisdictions),
		&details.Citation,
		&details.CurrentVersion,
		&details.ContentHash,
		&details.LibraryStatus,
		&details.EffectiveFrom,
		&details.EffectiveTo,
		&astBytes,
		&paramBytes,
		pq.Array(&rulesetCodes),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule %s not found", ruleID)
		}
		return nil, fmt.Errorf("fetch rule details: %w", err)
	}

	details.Jurisdictions = jurisdictions
	details.RulesetCodes = rulesetCodes
	details.Domain = DeriveDomain(details.RuleCode)

	if len(astBytes) > 0 {
		_ = json.Unmarshal(astBytes, &details.ASTCondition)
	}
	if len(paramBytes) > 0 {
		_ = json.Unmarshal(paramBytes, &details.ParameterThresholds)
	}

	// Fetch version history timeline
	vRows, err := s.db.QueryContext(ctx, `
		SELECT version, effective_from, effective_to, COALESCE(citation, ''), content_hash, COALESCE(created_by, 'system'), created_at
		FROM compliance.compliance_rule_version
		WHERE rule_id = $1
		ORDER BY version DESC
	`, ruleID)
	if err != nil {
		return nil, fmt.Errorf("fetch version history: %w", err)
	}
	defer vRows.Close()

	for vRows.Next() {
		var vs RuleVersionSummary
		if err := vRows.Scan(&vs.Version, &vs.EffectiveFrom, &vs.EffectiveTo, &vs.Citation, &vs.ContentHash, &vs.CreatedBy, &vs.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan version summary: %w", err)
		}
		details.VersionHistory = append(details.VersionHistory, vs)
	}

	// Find sample test cases from the scenario corpus
	for _, tc := range CoreScenarioCorpus {
		if tc.RuleCode == details.RuleCode || strings.HasPrefix(tc.Code, details.RuleCode+":") || strings.HasPrefix(tc.Code, details.RuleCode+"_") {
			details.SampleTestCases = append(details.SampleTestCases, tc)
		}
	}

	return &details, nil
}

// ListRulesets returns all licensable ruleset packages and their constituent rules
func (s *Service) ListRulesets(ctx context.Context) ([]RulesetSummary, error) {
	definitions := map[string]struct {
		Name        string
		Description string
		PlanTier    string
	}{
		"CORE_REGULATORY": {
			Name:        "Core Regulatory & Cross-Border Compliance Pack",
			Description: "UCITS 5/10/40, 40-Act 75-5-10, Restricted/Sanctions Lists, CSDR, MiFID II RTS 22, and Cross-Border passporting.",
			PlanTier:    "enterprise",
		},
		"MARKET_CONDUCT": {
			Name:        "Market Conduct & Surveillance Pack",
			Description: "Reg M 105, IRS 1091 wash sales, front-running, layering/spoofing, Reg T freeriding, and SEC locate rules.",
			PlanTier:    "pro",
		},
		"INSTITUTIONAL_CONTROLS": {
			Name:        "Institutional Controls & Risk Pack",
			Description: "Fat-finger notional/ADV limits, self-trade prevention, VaR leverage, liquidity bucketing, and personal trading preclearance.",
			PlanTier:    "enterprise",
		},
		"POST_TRADE_MONITORING": {
			Name:        "Post-Trade Portfolio & Exposure Monitoring Pack",
			Description: "Single/group issuer concentration, sovereign/agency limits, counterparty PFE, clearing, custodian, deposit, and collateral floors.",
			PlanTier:    "enterprise",
		},
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT ruleset_code, rule_id
		FROM compliance.compliance_ruleset_membership
		ORDER BY ruleset_code, rule_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query rulesets: %w", err)
	}
	defer rows.Close()

	rulesByCode := make(map[string][]uuid.UUID)
	for rows.Next() {
		var code string
		var ruleID uuid.UUID
		if err := rows.Scan(&code, &ruleID); err != nil {
			return nil, fmt.Errorf("scan ruleset membership: %w", err)
		}
		rulesByCode[code] = append(rulesByCode[code], ruleID)
	}

	rulesets := make([]RulesetSummary, 0, len(definitions))
	for code, def := range definitions {
		ruleIDs := rulesByCode[code]
		rulesets = append(rulesets, RulesetSummary{
			RulesetCode: code,
			Name:        def.Name,
			Description: def.Description,
			PlanTier:    def.PlanTier,
			TotalRules:  len(ruleIDs),
			RuleIDs:     ruleIDs,
		})
	}

	return rulesets, nil
}

// GetTenantActivationMatrix returns a tenant's complete compliance activation matrix
func (s *Service) GetTenantActivationMatrix(ctx context.Context, tenantID uuid.UUID) (*TenantActivationMatrix, error) {
	var matrix TenantActivationMatrix
	matrix.TenantID = tenantID

	err := s.db.QueryRowContext(ctx, `
		SELECT name, COALESCE(plan, 'enterprise'), COALESCE(gold_copy, false)
		FROM public.tenants
		WHERE id = $1
	`, tenantID).Scan(&matrix.TenantName, &matrix.Plan, &matrix.GoldCopy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant %s not found", tenantID)
		}
		return nil, fmt.Errorf("fetch tenant info: %w", err)
	}

	// 1. Load Gold Master Core Rules
	coreRules, err := s.ListCoreRules(ctx, ListRulesFilter{})
	if err != nil {
		return nil, fmt.Errorf("load core rules for activation matrix: %w", err)
	}

	// 2. Load Tenant Activations from compliance.tenant_rule_activation
	actRows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, enabled, inherit_mode, COALESCE(activated_by, ''), activated_at
		FROM compliance.tenant_rule_activation
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant activations: %w", err)
	}
	defer actRows.Close()

	type actRecord struct {
		enabled     bool
		inheritMode compliance.InheritMode
		activatedBy string
		activatedAt *time.Time
	}
	activations := make(map[uuid.UUID]actRecord)
	for actRows.Next() {
		var rid uuid.UUID
		var ar actRecord
		var mode string
		var actAt sql.NullTime
		if err := actRows.Scan(&rid, &ar.enabled, &mode, &ar.activatedBy, &actAt); err != nil {
			return nil, fmt.Errorf("scan tenant activation row: %w", err)
		}
		ar.inheritMode = compliance.InheritMode(mode)
		if actAt.Valid {
			ar.activatedAt = &actAt.Time
		}
		activations[rid] = ar
	}

	// 3. Load Tenant Custom/Extended Rules from compliance.compliance_rule
	customRows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.core_rule_id, r.inherit_mode, r.pinned_core_version, r.drift_status,
		       r.parameter_thresholds, v.content_hash
		FROM compliance.compliance_rule r
		LEFT JOIN compliance.compliance_rule_version v ON r.id = v.rule_id AND r.current_version = v.version
		WHERE r.tenant_id = $1 AND r.valid_to IS NULL
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant custom rules: %w", err)
	}
	defer customRows.Close()

	type customRecord struct {
		ruleID            uuid.UUID
		coreRuleID        *uuid.UUID
		inheritMode       compliance.InheritMode
		pinnedCoreVersion int
		driftStatus       compliance.DriftStatus
		thresholds        map[string]string
		contentHash       string
	}
	customByCoreID := make(map[uuid.UUID]customRecord)
	for customRows.Next() {
		var cr customRecord
		var coreID sql.NullString
		var mode, driftStr string
		var paramBytes []byte
		var contentHash sql.NullString

		if err := customRows.Scan(&cr.ruleID, &coreID, &mode, &cr.pinnedCoreVersion, &driftStr, &paramBytes, &contentHash); err != nil {
			return nil, fmt.Errorf("scan custom rule: %w", err)
		}
		if coreID.Valid {
			id, _ := uuid.Parse(coreID.String)
			cr.coreRuleID = &id
		}
		cr.inheritMode = compliance.InheritMode(mode)
		cr.driftStatus = compliance.DriftStatus(driftStr)
		if contentHash.Valid {
			cr.contentHash = contentHash.String
		}
		if len(paramBytes) > 0 {
			_ = json.Unmarshal(paramBytes, &cr.thresholds)
		}
		if cr.coreRuleID != nil {
			customByCoreID[*cr.coreRuleID] = cr
		}
	}

	// 4. Merge Core Rules with Tenant Activations & Overrides
	matrix.Rules = make([]TenantActivationItem, 0, len(coreRules))
	activeRuleSet := make(map[uuid.UUID]bool)

	for _, cr := range coreRules {
		// Fetch core rule thresholds
		var coreParams map[string]string
		var paramBytes []byte
		_ = s.db.QueryRowContext(ctx, "SELECT parameter_thresholds FROM compliance.compliance_rule WHERE id = $1", cr.ID).Scan(&paramBytes)
		if len(paramBytes) > 0 {
			_ = json.Unmarshal(paramBytes, &coreParams)
		}

		item := TenantActivationItem{
			RuleID:             cr.ID,
			RuleCode:           cr.RuleCode,
			RuleName:           cr.Name,
			RulePhase:          cr.RulePhase,
			Severity:           cr.Severity,
			Citation:           cr.Citation,
			Domain:             cr.Domain,
			Jurisdictions:      cr.Jurisdictions,
			RulesetCodes:       cr.RulesetCodes,
			CurrentCoreVersion: cr.CurrentVersion,
			PinnedCoreVersion:  cr.CurrentVersion,
			Enabled:            matrix.GoldCopy, // gold copy tenant is enabled by default
			InheritMode:        compliance.Inherit,
			DriftStatus:        compliance.DriftCurrent,
			CoreThresholds:     coreParams,
			TenantThresholds:   coreParams,
			CoreContentHash:    cr.ContentHash,
		}

		// Check activation
		if act, exists := activations[cr.ID]; exists {
			item.Enabled = act.enabled
			item.InheritMode = act.inheritMode
			item.ActivatedBy = act.activatedBy
			item.ActivatedAt = act.activatedAt
		}

		// Check custom/extend override
		if custom, exists := customByCoreID[cr.ID]; exists {
			item.InheritMode = custom.inheritMode
			item.PinnedCoreVersion = custom.pinnedCoreVersion
			item.DriftStatus = custom.driftStatus
			item.TenantThresholds = custom.thresholds
			item.TenantContentHash = custom.contentHash
			if custom.driftStatus == compliance.DriftCoreVersionUpdated {
				matrix.DriftCount++
			}
		}

		if item.Enabled {
			matrix.TotalActive++
			activeRuleSet[cr.ID] = true
		}
		matrix.Rules = append(matrix.Rules, item)
	}
	matrix.TotalRules = len(matrix.Rules)

	// 5. Calculate Ruleset Statuses
	rulesets, err := s.ListRulesets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rulesets for activation matrix: %w", err)
	}

	matrix.Rulesets = make([]TenantRulesetStatus, 0, len(rulesets))
	for _, rs := range rulesets {
		activeCount := 0
		for _, rid := range rs.RuleIDs {
			if activeRuleSet[rid] {
				activeCount++
			}
		}

		isLicensed := true
		if rs.PlanTier == "enterprise" && matrix.Plan != "enterprise" {
			isLicensed = false
		}

		matrix.Rulesets = append(matrix.Rulesets, TenantRulesetStatus{
			RulesetCode: rs.RulesetCode,
			Name:        rs.Name,
			Description: rs.Description,
			PlanTier:    rs.PlanTier,
			IsLicensed:  isLicensed,
			IsActive:    activeCount == rs.TotalRules && rs.TotalRules > 0,
			ActiveRules: activeCount,
			TotalRules:  rs.TotalRules,
		})
	}

	return &matrix, nil
}

// UpdateTenantRuleActivation updates activation toggle, inheritance mode, and parameter overrides
func (s *Service) UpdateTenantRuleActivation(ctx context.Context, tenantID, ruleID uuid.UUID, req UpdateRuleActivationRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	// 1. Fetch Core Rule details
	var coreCode, coreName, corePhase, coreSeverity, coreCitation string
	var coreCurVer int
	var coreASTBytes, coreParamBytes []byte
	err = tx.QueryRowContext(ctx, `
		SELECT rule_code, name, rule_phase, severity, COALESCE(citation, ''), current_version, ast_condition, parameter_thresholds
		FROM compliance.compliance_rule
		WHERE id = $1 AND valid_to IS NULL
	`, ruleID).Scan(&coreCode, &coreName, &corePhase, &coreSeverity, &coreCitation, &coreCurVer, &coreASTBytes, &coreParamBytes)
	if err != nil {
		return fmt.Errorf("fetch core rule: %w", err)
	}

	// 2. Upsert compliance.tenant_rule_activation
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.tenant_rule_activation (
			tenant_id, rule_id, enabled, inherit_mode, activated_by, activated_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, now(), now()
		)
		ON CONFLICT (tenant_id, rule_id) DO UPDATE
		SET 
			enabled = EXCLUDED.enabled,
			inherit_mode = EXCLUDED.inherit_mode,
			activated_by = EXCLUDED.activated_by,
			activated_at = CASE WHEN EXCLUDED.enabled THEN now() ELSE compliance.tenant_rule_activation.activated_at END,
			updated_at = now()
	`, tenantID, ruleID, req.Enabled, string(req.InheritMode), req.ActorID)
	if err != nil {
		return fmt.Errorf("upsert tenant activation: %w", err)
	}

	// 3. Handle Extend Mode
	if req.InheritMode == compliance.Extend {
		// Calculate custom content hash
		overrideJSON, _ := json.Marshal(req.ParameterOverrides)
		tenantContentHash, err := canonical.ComputeRuleContentHashFromRaw(coreASTBytes, overrideJSON, coreCitation)
		if err != nil {
			return fmt.Errorf("compute tenant content hash: %w", err)
		}
		bytecodeHash := canonical.ComputeBytecodeHash(nil)

		// Check if extended rule already exists for tenant
		var existingTenantRuleID uuid.UUID
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM compliance.compliance_rule
			WHERE tenant_id = $1 AND core_rule_id = $2 AND valid_to IS NULL
		`, tenantID, ruleID).Scan(&existingTenantRuleID)

		if err == sql.ErrNoRows {
			// Insert new extend rule
			newRuleID := uuid.New()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO compliance.compliance_rule (
					id, tenant_id, core_rule_id, inherit_mode, pinned_core_version, drift_status,
					rule_code, name, rule_phase, severity, priority, is_active, current_version,
					ast_condition, parameter_thresholds, citation, compiled_bytecode
				) VALUES (
					$1, $2, $3, 'extend', $4, 'CURRENT',
					$5, $6, $7, $8, 100, true, 1,
					$9::jsonb, $10::jsonb, $11, ''::bytea
				)
			`, newRuleID, tenantID, ruleID, coreCurVer, coreCode, coreName+" (Custom Override)", corePhase, coreSeverity, string(coreASTBytes), string(overrideJSON), coreCitation)
			if err != nil {
				return fmt.Errorf("insert extend rule: %w", err)
			}

			// Insert version snapshot v1
			_, err = tx.ExecContext(ctx, `
				INSERT INTO compliance.compliance_rule_version (
					rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
					citation, effective_from, content_hash, compiled_bytecode_hash, created_by
				) VALUES (
					$1, 1, $2, $3::jsonb, $4::jsonb,
					$5, now(), $6, $7, $8
				)
			`, newRuleID, tenantID, string(coreASTBytes), string(overrideJSON), coreCitation, tenantContentHash, bytecodeHash, req.ActorID)
			if err != nil {
				return fmt.Errorf("insert extend rule snapshot: %w", err)
			}
		} else if err == nil {
			var curVer int
			var curASTBytes []byte
			var curCitation string
			_ = tx.QueryRowContext(ctx, `SELECT current_version, ast_condition, COALESCE(citation, '') FROM compliance.compliance_rule WHERE id = $1`, existingTenantRuleID).Scan(&curVer, &curASTBytes, &curCitation)
			newVer := curVer + 1

			// Insert companion snapshot v(newVer)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO compliance.compliance_rule_version (
					rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
					citation, effective_from, content_hash, compiled_bytecode_hash, created_by
				) VALUES (
					$1, $2, $3, $4::jsonb, $5::jsonb,
					$6, now(), $7, $8, $9
				)
			`, existingTenantRuleID, newVer, tenantID, string(curASTBytes), string(overrideJSON), curCitation, tenantContentHash, bytecodeHash, req.ActorID)
			if err != nil {
				return fmt.Errorf("insert extend rule snapshot on threshold update: %w", err)
			}

			// Update existing extend rule
			_, err = tx.ExecContext(ctx, `
				UPDATE compliance.compliance_rule
				SET 
					current_version = $1,
					parameter_thresholds = $2::jsonb,
					drift_status = 'CURRENT',
					updated_at = now()
				WHERE id = $3
			`, newVer, string(overrideJSON), existingTenantRuleID)
			if err != nil {
				return fmt.Errorf("update extend rule: %w", err)
			}
		}
	} else if req.InheritMode == compliance.Inherit {
		// Soft-delete any custom/extend rule for this tenant
		_, err = tx.ExecContext(ctx, `
			UPDATE compliance.compliance_rule
			SET valid_to = now(), is_active = false
			WHERE tenant_id = $1 AND core_rule_id = $2 AND valid_to IS NULL
		`, tenantID, ruleID)
		if err != nil {
			return fmt.Errorf("soft-delete extend rule on revert to inherit: %w", err)
		}
	}

	// 4. Governance Audit Event
	eventType := "RULE_ACTIVATED"
	if !req.Enabled {
		eventType = "RULE_DEACTIVATED"
	}
	auditNotes := fmt.Sprintf("Updated rule %s activation for tenant %s: enabled=%v, mode=%s", coreCode, tenantID, req.Enabled, req.InheritMode)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.governance_audit_event (
			id, tenant_id, event_type, rule_id, old_pinned_version, new_pinned_version, steward_id, steward_notes, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, now()
		)
	`, tenantID, eventType, ruleID, coreCurVer, coreCurVer, req.ActorID, auditNotes)
	if err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}

	return tx.Commit()
}

// RepinTenantRule repins an extended rule to the latest or specified core version
func (s *Service) RepinTenantRule(ctx context.Context, tenantID, ruleID uuid.UUID, req RepinRuleRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	// 1. Fetch core rule's current target version
	var targetASTBytes, targetCitation string
	err = tx.QueryRowContext(ctx, `
		SELECT resolved_ast::text, citation
		FROM compliance.compliance_rule_version
		WHERE rule_id = $1 AND version = $2
	`, ruleID, req.TargetVersion).Scan(&targetASTBytes, &targetCitation)
	if err != nil {
		return fmt.Errorf("fetch target core rule version %d: %w", req.TargetVersion, err)
	}

	// 2. Fetch tenant's extended rule
	var tenantRuleID uuid.UUID
	var curVersion, oldPinnedVersion int
	var curParamsBytes []byte
	err = tx.QueryRowContext(ctx, `
		SELECT id, current_version, pinned_core_version, parameter_thresholds
		FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND core_rule_id = $2 AND inherit_mode = 'extend' AND valid_to IS NULL
	`, tenantID, ruleID).Scan(&tenantRuleID, &curVersion, &oldPinnedVersion, &curParamsBytes)
	if err != nil {
		return fmt.Errorf("fetch tenant extended rule: %w", err)
	}

	// Recompute tenant hash with new AST
	newTenantHash, err := canonical.ComputeRuleContentHashFromRaw([]byte(targetASTBytes), curParamsBytes, targetCitation)
	if err != nil {
		return fmt.Errorf("recompute tenant hash for repin: %w", err)
	}
	bytecodeHash := canonical.ComputeBytecodeHash(nil)
	newVersion := curVersion + 1

	// Insert companion snapshot
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, $2, $3, $4::jsonb, $5::jsonb,
			$6, now(), $7, $8, $9
		)
	`, tenantRuleID, newVersion, tenantID, targetASTBytes, string(curParamsBytes), targetCitation, newTenantHash, bytecodeHash, req.ActorID)
	if err != nil {
		return fmt.Errorf("insert companion snapshot for repin: %w", err)
	}

	// 3. Update pinned version, current_version, ast_condition, and clear drift status
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET 
			current_version = $1,
			pinned_core_version = $2,
			ast_condition = $3::jsonb,
			citation = $4,
			drift_status = 'CURRENT',
			updated_at = now()
		WHERE id = $5
	`, newVersion, req.TargetVersion, targetASTBytes, targetCitation, tenantRuleID)
	if err != nil {
		return fmt.Errorf("update extended rule for repin: %w", err)
	}

	// 4. Governance Audit Event
	var oldVersion int
	_ = tx.QueryRowContext(ctx, "SELECT pinned_core_version FROM compliance.compliance_rule WHERE id = $1", tenantRuleID).Scan(&oldVersion)
	if oldVersion == 0 {
		oldVersion = 1
	}

	auditNotes := fmt.Sprintf("Repinned rule %s for tenant %s from v%d to core version %d (Hash: %s). Notes: %s", ruleID, tenantID, oldVersion, req.TargetVersion, newTenantHash, req.StewardNotes)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.governance_audit_event (
			id, tenant_id, event_type, rule_id, old_pinned_version, new_pinned_version, steward_id, steward_notes, created_at
		) VALUES (
			gen_random_uuid(), $1, 'RULE_REPINNED', $2, $3, $4, $5, $6, now()
		)
	`, tenantID, tenantRuleID, oldVersion, req.TargetVersion, req.ActorID, auditNotes)
	if err != nil {
		return fmt.Errorf("record repin audit event: %w", err)
	}

	return tx.Commit()
}

// ToggleRulesetActivation toggles all rules in a ruleset for a tenant
func (s *Service) ToggleRulesetActivation(ctx context.Context, tenantID uuid.UUID, rulesetCode string, enabled bool, actorID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	// 1. Fetch ruleset rules
	rows, err := tx.QueryContext(ctx, `
		SELECT rule_id FROM compliance.compliance_ruleset_membership
		WHERE ruleset_code = $1
	`, rulesetCode)
	if err != nil {
		return fmt.Errorf("fetch ruleset members: %w", err)
	}
	defer rows.Close()

	ruleIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var rid uuid.UUID
		if err := rows.Scan(&rid); err != nil {
			return err
		}
		ruleIDs = append(ruleIDs, rid)
	}

	if len(ruleIDs) == 0 {
		return fmt.Errorf("no rules found for ruleset %s", rulesetCode)
	}

	// 2. Batch upsert activation table
	for _, rid := range ruleIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.tenant_rule_activation (
				tenant_id, rule_id, enabled, inherit_mode, activated_by, activated_at, updated_at
			) VALUES (
				$1, $2, $3, 'inherit', $4, now(), now()
			)
			ON CONFLICT (tenant_id, rule_id) DO UPDATE
			SET 
				enabled = EXCLUDED.enabled,
				activated_by = EXCLUDED.activated_by,
				activated_at = CASE WHEN EXCLUDED.enabled THEN now() ELSE compliance.tenant_rule_activation.activated_at END,
				updated_at = now()
		`, tenantID, rid, enabled, actorID)
		if err != nil {
			return fmt.Errorf("toggle activation for rule %s: %w", rid, err)
		}
	}

	// 3. Record Audit Event
	rulesetEventType := "RULE_ACTIVATED"
	if !enabled {
		rulesetEventType = "RULE_DEACTIVATED"
	}
	auditNotes := fmt.Sprintf("Toggled ruleset %s for tenant %s to enabled=%v (%d rules affected)", rulesetCode, tenantID, enabled, len(ruleIDs))
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.governance_audit_event (
			id, tenant_id, event_type, rule_id, old_pinned_version, new_pinned_version, steward_id, steward_notes, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, 1, 1, $4, $5, now()
		)
	`, tenantID, rulesetEventType, ruleIDs[0], actorID, auditNotes)
	if err != nil {
		return fmt.Errorf("record ruleset audit event: %w", err)
	}

	return tx.Commit()
}
