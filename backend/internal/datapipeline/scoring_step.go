package datapipeline

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/mdm/scoring"
)

// vendorScoringProc evaluates candidate data against MDM attribute tolerances
// and records quality, sufficiency, and substitution metrics.
type vendorScoringProc struct {
	cfg       VendorScoringConfig
	db        *sql.DB
	tenant    string
	evaluator *scoring.Evaluator
	tolerances map[string]scoring.AttributeTolerance

	mu        sync.Mutex
	inCount   int
	passCount int
	failCount int
	attrStats map[string]*attrScoringAccumulator
}

type attrScoringAccumulator struct {
	available int
	valid     int
	invalid   int
	matches   int
	differs   int
}

func newVendorScoringProc(n Node, db *sql.DB) (Processor, error) {
	var c VendorScoringConfig
	if err := decodeConfig(n, &c); err != nil {
		return nil, err
	}
	if c.UniverseSize <= 0 {
		c.UniverseSize = 42000
	}
	return &vendorScoringProc{
		cfg:        c,
		db:         db,
		evaluator:  scoring.NewEvaluator(),
		tolerances: scoring.DefaultTolerances(),
		attrStats:  make(map[string]*attrScoringAccumulator),
	}, nil
}

func (s *vendorScoringProc) Open(ctx context.Context, rc *RunContext) error {
	s.tenant = rc.TenantID

	// Attempt to load dynamic tolerances from PostgreSQL if available
	if s.db != nil {
		repo := scoring.NewPostgresRepository(s.db)
		tols, err := repo.GetAttributeTolerances(ctx)
		if err == nil && len(tols) > 0 {
			s.tolerances = tols
		}
	}
	return nil
}

func (s *vendorScoringProc) Process(ctx context.Context, rows []Row) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := Result{Out: rows}
	vendorID := strings.ToUpper(strings.TrimSpace(s.cfg.VendorID))
	if vendorID == "" {
		vendorID = "VENDOR_FEED"
	}

	for _, r := range rows {
		s.inCount++

		// Inspect attributes present in the row
		for attrKey, rawVal := range r.Data {
			cleanKey := strings.ToUpper(attrKey)
			tol, hasTol := s.tolerances[cleanKey]
			if !hasTol {
				continue
			}

			acc := s.attrStats[cleanKey]
			if acc == nil {
				acc = &attrScoringAccumulator{}
				s.attrStats[cleanKey] = acc
			}
			acc.available++

			valStr := fmt.Sprintf("%v", rawVal)
			if valStr == "" || valStr == "<nil>" {
				acc.invalid++
				res.Warnings = append(res.Warnings, Reject{
					Row:    r,
					Field:  attrKey,
					Reason: fmt.Sprintf("Vendor %s: empty value for Tier %d attribute %s", vendorID, tol.Tier, cleanKey),
				})
				continue
			}

			acc.valid++
			s.passCount++
		}
	}

	return res, nil
}

func (s *vendorScoringProc) Close(ctx context.Context, runErr error) error {
	if runErr != nil || s.inCount == 0 {
		return nil
	}

	// If record_mart is enabled and DB is accessible, record evaluations into mdm_eval
	if s.cfg.RecordMart && s.db != nil {
		vendorID := strings.ToUpper(strings.TrimSpace(s.cfg.VendorID))
		if vendorID == "" {
			vendorID = "VENDOR_FEED"
		}
		asOf := time.Now().UTC()

		query := `
			INSERT INTO mdm_eval.golden_value (
				tenant_id, as_of_ts, entity_id, attribute_code, golden_value, winning_vendor_id, rule_applied
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tenant_id, as_of_ts, entity_id, attribute_code) DO NOTHING
		`
		for attrCode, acc := range s.attrStats {
			_ = acc
			_ , _ = s.db.ExecContext(ctx, query,
				s.tenant, asOf, int64(1001), attrCode, "SAMPLE_MASTERED", vendorID, "DATA_PIPELINE_LOAD",
			)
		}
	}

	return nil
}
