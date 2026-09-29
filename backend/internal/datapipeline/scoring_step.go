package datapipeline

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm/scoring"
)

type goldenValueEntry struct {
	entityID      int64
	attributeCode string
	goldenValue   string
	vendorID      string
}

// vendorScoringProc evaluates candidate data against MDM attribute tolerances
// and records quality, sufficiency, and substitution metrics.
type vendorScoringProc struct {
	cfg        VendorScoringConfig
	db         *sql.DB
	tenant     string
	evaluator  *scoring.Evaluator
	tolerances map[string]scoring.AttributeTolerance

	mu            sync.Mutex
	inCount       int
	passCount     int
	failCount     int
	attrStats     map[string]*attrScoringAccumulator
	goldenEntries []goldenValueEntry
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

func extractEntityID(data map[string]any, fallback int64) int64 {
	candidates := []string{"entity_id", "id", "master_id", "security_id", "@source_key", "source_key", "sec_id"}
	for _, c := range candidates {
		val, ok := data[c]
		if !ok || val == nil {
			continue
		}
		switch v := val.(type) {
		case int:
			return int64(v)
		case int64:
			return v
		case float64:
			return int64(v)
		case string:
			s := strings.TrimSpace(v)
			if s == "" {
				continue
			}
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n
			}
			h := fnv.New64a()
			h.Write([]byte(s))
			id := int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
			if id == 0 {
				id = 1
			}
			return id
		}
	}
	if fallback <= 0 {
		return 1
	}
	return fallback
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
		entID := extractEntityID(r.Data, int64(r.Num))

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

			if s.cfg.RecordMart && len(s.goldenEntries) < 10000 {
				s.goldenEntries = append(s.goldenEntries, goldenValueEntry{
					entityID:      entID,
					attributeCode: cleanKey,
					goldenValue:   valStr,
					vendorID:      vendorID,
				})
			}
		}
	}

	return res, nil
}

func (s *vendorScoringProc) Close(ctx context.Context, runErr error) error {
	if runErr != nil || s.inCount == 0 {
		return nil
	}

	// If record_mart is enabled and DB is accessible, record evaluations into mdm_eval
	if s.cfg.RecordMart && s.db != nil && len(s.goldenEntries) > 0 {
		tenantUUID, err := uuid.Parse(strings.TrimSpace(s.tenant))
		if err != nil {
			tenantUUID = uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
		}
		asOf := time.Now().UTC()

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin tx for golden values: %w", err)
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO mdm_eval.golden_value (
				tenant_id, as_of_ts, entity_id, attribute_code, golden_value, winning_vendor_id, rule_applied
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tenant_id, as_of_ts, entity_id, attribute_code) DO NOTHING
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare golden value statement: %w", err)
		}
		defer stmt.Close()

		for _, entry := range s.goldenEntries {
			if _, err := stmt.ExecContext(ctx,
				tenantUUID, asOf, entry.entityID, entry.attributeCode, entry.goldenValue, entry.vendorID, "DATA_PIPELINE_SURVIVORSHIP",
			); err != nil {
				return fmt.Errorf("failed to insert golden value for entity %d attr %s: %w", entry.entityID, entry.attributeCode, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit golden values: %w", err)
		}
	}

	return nil
}
