package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/jmoiron/sqlx"
)

// VendorFeedPayload represents an incoming multi-vendor feed item for any financial domain.
type VendorFeedPayload struct {
	DomainKey       string                 `json:"domain_key"`
	MasterEntitySID string                 `json:"master_entity_sid"`
	VendorName      string                 `json:"vendor_name"`
	EffectiveDate   time.Time              `json:"effective_date"`
	Attributes      map[string]interface{} `json:"attributes"`
	ConfidenceScore float64                `json:"confidence_score"`
}

type FeedCandidate struct {
	Vendor string      `json:"vendor"`
	Val    interface{} `json:"value"`
	Conf   float64     `json:"confidence"`
}

// UniversalMDMResolver synthesizes multi-vendor payloads into authoritative golden records.
type UniversalMDMResolver struct {
	db *sqlx.DB
}

func NewUniversalMDMResolver(db *sqlx.DB) *UniversalMDMResolver {
	return &UniversalMDMResolver{
		db: db,
	}
}

// MasterIncomingFeeds synthesizes multiple incoming vendor feeds into an authoritative Golden Record
func (r *UniversalMDMResolver) MasterIncomingFeeds(
	ctx context.Context,
	tenantID uuid.UUID,
	domainKey, masterSID string,
	effectiveDate time.Time,
	feeds []VendorFeedPayload,
) (map[string]interface{}, map[string]string, error) {
	// Tenant context guard
	if tenantID == uuid.Nil {
		return nil, nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	ruleMap := make(map[string]struct {
		Strategy  string
		Vendors   []string
		Tolerance float64
	})

	// 1. Fetch Field Survivorship Rules if DB is configured (Config-Before-Code)
	if r.db != nil {
		var rulesList []struct {
			FieldName           string  `db:"field_name"`
			Strategy            string  `db:"strategy"`
			PriorityVendorsRaw  []byte  `db:"priority_vendors"`
			AnomalyTolerancePct float64 `db:"anomaly_tolerance_pct"`
		}

		query := `
			SELECT field_name, strategy, priority_vendors, anomaly_tolerance_pct 
			FROM mdm.universal_survivorship_rules 
			WHERE tenant_id = $1 AND domain_key = $2 AND is_active = TRUE;
		`
		_ = r.db.SelectContext(ctx, &rulesList, query, tenantID, domainKey)
		for _, rl := range rulesList {
			var vends []string
			_ = json.Unmarshal(rl.PriorityVendorsRaw, &vends)
			ruleMap[rl.FieldName] = struct {
				Strategy  string
				Vendors   []string
				Tolerance float64
			}{Strategy: rl.Strategy, Vendors: vends, Tolerance: rl.AnomalyTolerancePct}
		}
	}

	// 2. Extract All Unique Field Keys Across Feeds
	allFields := make(map[string]bool)
	for _, f := range feeds {
		for k := range f.Attributes {
			allFields[k] = true
		}
	}

	goldenRecord := make(map[string]interface{})
	winningSources := make(map[string]string)

	// 3. Resolve Field-by-Field Survivorship & Run Validation Checks
	for field := range allFields {
		candidates := make([]FeedCandidate, 0)
		for _, f := range feeds {
			if v, ok := f.Attributes[field]; ok && v != nil {
				candidates = append(candidates, FeedCandidate{Vendor: f.VendorName, Val: v, Conf: f.ConfidenceScore})
			}
		}
		if len(candidates) == 0 {
			continue
		}

		rl, hasRule := ruleMap[field]
		if !hasRule {
			rl.Strategy = "SOURCE_PRIORITY"
			rl.Vendors = []string{"DTCC", "BLOOMBERG", "REFINITIV", "IDC", "CRIMS", "CUSTODIAN_BNY"}
			rl.Tolerance = 10.0
		}

		// Check for pricing/numeric anomaly deviations (>10%)
		if isNumeric(candidates[0].Val) && len(candidates) > 1 {
			r.checkNumericalAnomaly(ctx, tenantID, domainKey, masterSID, field, candidates, rl.Tolerance)
		}

		// Apply Survivorship Strategy
		winnerVendor := candidates[0].Vendor
		winnerVal := candidates[0].Val

		switch rl.Strategy {
		case "SOURCE_PRIORITY":
			foundPriority := false
			for _, pVend := range rl.Vendors {
				for _, c := range candidates {
					if strings.EqualFold(c.Vendor, pVend) {
						winnerVendor = c.Vendor
						winnerVal = c.Val
						foundPriority = true
						break
					}
				}
				if foundPriority {
					break
				}
			}
		case "CONFIDENCE_SCORE":
			maxConf := -1.0
			for _, c := range candidates {
				if c.Conf > maxConf {
					maxConf = c.Conf
					winnerVendor = c.Vendor
					winnerVal = c.Val
				}
			}
		}

		// 4. Validate Value using Shared FastRecord/WASM Checksums
		if field == "isin" || field == "id_isin" || field == "security_isin" {
			if strVal, ok := winnerVal.(string); ok {
				if !boresolver.ValidateISIN(strVal) {
					r.raiseChecksumException(ctx, tenantID, domainKey, masterSID, field, strVal, "ISO 6166 Checksum Failure")
				}
			}
		} else if field == "lei_code" || field == "lei" {
			if strVal, ok := winnerVal.(string); ok {
				if !boresolver.ValidateLEI(strVal) {
					r.raiseChecksumException(ctx, tenantID, domainKey, masterSID, field, strVal, "ISO 17442 LEI Checksum Failure")
				}
			}
		}

		goldenRecord[field] = winnerVal
		winningSources[field] = winnerVendor
	}

	// 5. Persist Golden Record with Dual-Time Bitemporal Coordinates (Te vs Tk)
	if r.db != nil {
		goldenJSON, _ := json.Marshal(goldenRecord)
		winningJSON, _ := json.Marshal(winningSources)
		nowTk := time.Now().UTC()

		upsertGoldenSQL := `
			INSERT INTO mdm.golden_record_store (
				tenant_id, domain_key, master_entity_sid, golden_attributes, winning_sources,
				effective_date, knowledge_timestamp, is_active, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW())
			ON CONFLICT (tenant_id, domain_key, master_entity_sid, effective_date, knowledge_timestamp) 
			DO UPDATE SET
				golden_attributes = EXCLUDED.golden_attributes,
				winning_sources = EXCLUDED.winning_sources;
		`
		_, _ = r.db.ExecContext(ctx, upsertGoldenSQL,
			tenantID, domainKey, masterSID, goldenJSON, winningJSON, effectiveDate, nowTk)
	}

	return goldenRecord, winningSources, nil
}

func (r *UniversalMDMResolver) checkNumericalAnomaly(
	ctx context.Context,
	tenantID uuid.UUID,
	domainKey, masterSID, field string,
	candidates []FeedCandidate,
	tolerancePct float64,
) {
	if r.db == nil {
		return
	}
	nums := make([]float64, 0, len(candidates))
	for _, c := range candidates {
		if n, ok := toFloat(c.Val); ok {
			nums = append(nums, n)
		}
	}
	if len(nums) < 2 {
		return
	}

	minV, maxV := nums[0], nums[0]
	for _, v := range nums[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	if minV > 0 && ((maxV-minV)/minV)*100.0 > tolerancePct {
		competing, _ := json.Marshal(candidates)
		_, _ = r.db.ExecContext(ctx, `
			INSERT INTO mdm.universal_exception_queue (
				tenant_id, domain_key, master_entity_sid, field_name,
				competing_values, anomaly_type, status
			) VALUES ($1, $2, $3, $4, $5, 'PRICE_TOLERANCE_BREACH', 'OPEN')
		`, tenantID, domainKey, masterSID, field, competing)
	}
}

func (r *UniversalMDMResolver) raiseChecksumException(
	ctx context.Context,
	tenantID uuid.UUID,
	domainKey, masterSID, field, val, reason string,
) {
	if r.db == nil {
		return
	}
	competing, _ := json.Marshal([]map[string]string{{"value": val, "reason": reason}})
	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO mdm.universal_exception_queue (
			tenant_id, domain_key, master_entity_sid, field_name,
			competing_values, anomaly_type, status
		) VALUES ($1, $2, $3, $4, $5, 'CHECKSUM_FAILURE', 'OPEN')
	`, tenantID, domainKey, masterSID, field, competing)
}

func isNumeric(val interface{}) bool {
	_, ok := toFloat(val)
	return ok
}

func toFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0.0, false
	}
}
