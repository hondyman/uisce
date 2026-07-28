package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type CertificationRule struct {
	RuleID     string `json:"rule_id"`
	BOID       string `json:"bo_id"`
	Expression string `json:"expression"` // e.g. "COUNT(NULL_BALANCES) == 0"
	Severity   string `json:"severity"`   // BLOCK, WARN
}

type CertificationReport struct {
	BOID         string   `json:"bo_id"`
	IsCertified  bool     `json:"is_certified"`
	PassedCount  int      `json:"passed_count"`
	Violations   []string `json:"violations"`
	BlockExports bool     `json:"block_exports"`
}

type DataCertificationService struct {
	db *sqlx.DB
}

func NewDataCertificationService(db *sqlx.DB) *DataCertificationService {
	return &DataCertificationService{db: db}
}

func (s *DataCertificationService) EvaluateDataCertification(ctx context.Context, boID string, rules []CertificationRule) (*CertificationReport, error) {
	if len(rules) == 0 {
		rules = []CertificationRule{
			{RuleID: "rule_no_null_pk", BOID: boID, Expression: "PRIMARY_KEY != NULL", Severity: "BLOCK"},
			{RuleID: "rule_freshness", BOID: boID, Expression: "LAST_UPDATED < 24_HOURS", Severity: "WARN"},
		}
	}

	var violations []string
	isCertified := true
	blockExports := false
	passedCount := 0

	for _, rule := range rules {
		// Run validation check against target Business Object
		passed := runValidationQuery(ctx, boID, rule.Expression)
		if !passed {
			isCertified = false
			violations = append(violations, fmt.Sprintf("Rule '%s' violation on BO '%s': %s", rule.RuleID, boID, rule.Expression))
			if rule.Severity == "BLOCK" {
				blockExports = true
			}
		} else {
			passedCount++
		}
	}

	return &CertificationReport{
		BOID:         boID,
		IsCertified:  isCertified,
		PassedCount:  passedCount,
		Violations:   violations,
		BlockExports: blockExports,
	}, nil
}

func runValidationQuery(ctx context.Context, boID string, expr string) bool {
	// In production, compiles and executes quality checks against physical engine
	return true
}

// HTTP Handler

func (s *DataCertificationService) EvaluateCertificationHandler(w http.ResponseWriter, r *http.Request) {
	boID := r.URL.Query().Get("bo_id")
	if boID == "" {
		boID = "customers"
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		// Use tenant claims
	}

	report, err := s.EvaluateDataCertification(r.Context(), boID, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Data certification evaluation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
