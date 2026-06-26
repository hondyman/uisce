package mdm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
)

// ─── Source routing tables ────────────────────────────────────────────────────

// sourceByAccountType returns the preferred primary source for a semantic term,
// given the account type of the portfolio.
func sourceByAccountType(accountType, term string) string {
	switch accountType {
	case "institutional":
		if term == "Quantity" || term == "MarketValue" {
			return "FactSet"
		}
		return "Bloomberg"
	case "retail":
		return "Refinitiv"
	case "private_wealth":
		if term == "Price" {
			return "Bloomberg"
		}
		return "AccountingSystem"
	case "private_markets":
		return "Preqin"
	default:
		return "Bloomberg"
	}
}

// sourceByAssetClass returns the preferred source per asset class.
func sourceByAssetClass(assetClass, term string) string {
	switch assetClass {
	case "EQUITY":
		return "Bloomberg"
	case "FIXED_INCOME":
		return "Refinitiv"
	case "ALTERNATIVES":
		return "Bloomberg"
	case "PRIVATE_EQUITY":
		return "Preqin"
	case "REAL_ESTATE":
		return "MSCI_RE"
	default:
		return "Bloomberg"
	}
}

// sourceByRegion returns the preferred source per geographic region.
func sourceByRegion(region, term string) string {
	switch region {
	case "NAM":
		return "Bloomberg"
	case "EMEA":
		return "Refinitiv"
	case "APAC":
		return "Refinitiv"
	case "LATAM":
		return "Bloomberg"
	case "EM":
		return "S&P"
	default:
		return "Bloomberg"
	}
}

// ─── Orchestrator ─────────────────────────────────────────────────────────────

// PortfolioOrchestrator drives the full MDM data pipeline for portfolio mastering.
// It coordinates: source selection → data ingestion → gold copy build → Kafka publish.
type PortfolioOrchestrator struct {
	engine    *goldcopy.Engine
	publisher *goldcopy.Publisher
}

// NewPortfolioOrchestrator wires up the orchestrator with its engine and (optional) publisher.
// Pass nil for publisher to disable Kafka publishing (e.g. in tests).
func NewPortfolioOrchestrator(engine *goldcopy.Engine, publisher *goldcopy.Publisher) *PortfolioOrchestrator {
	return &PortfolioOrchestrator{
		engine:    engine,
		publisher: publisher,
	}
}

// IngestPortfolioData is the top-level entrypoint.
// For each (accountType, assetClass, region) combination it:
//  1. Determines the preferred source order using source routing tables.
//  2. Builds synthetic raw records from the source priority mappings.
//  3. Calls the gold copy engine to master the data.
//  4. Publishes the result to Kafka.
func (o *PortfolioOrchestrator) IngestPortfolioData(
	ctx context.Context,
	tenantID uuid.UUID,
	portfolioID string,
	accountTypes []string,
	assetClass string,
	region string,
	date time.Time,
) error {
	for _, accountType := range accountTypes {
		if err := o.ingestForAccountType(ctx, tenantID, portfolioID, accountType, assetClass, region, date); err != nil {
			// Log but continue for other account types
			log.Printf("[PortfolioOrchestrator] accountType=%s error: %v", accountType, err)
		}
	}
	return nil
}

func (o *PortfolioOrchestrator) ingestForAccountType(
	ctx context.Context,
	tenantID uuid.UUID,
	portfolioID string,
	accountType, assetClass, region string,
	date time.Time,
) error {
	// ── 1. Build priority-ordered source list using routing tables
	terms := []string{"portfolio_name", "base_currency", "portfolio_type", "inception_date",
		"risk_profile", "investment_objective", "benchmark_id", "strategy_id",
		"portfolio_manager_id", "custodian_id"}

	sourceScores := o.resolveSourceScores(accountType, assetClass, region, terms)

	// ── 2. Convert source priority into raw records
	// Each source gets a "raw record" with the fields it's authoritative for.
	rawRecords := o.buildRawRecordsFromSources(portfolioID, date, sourceScores)

	if len(rawRecords) == 0 {
		return fmt.Errorf("no sources resolved for portfolio %s / %s", portfolioID, accountType)
	}

	// ── 3. Build gold copy
	result, err := o.engine.BuildPortfolioGoldCopy(ctx, tenantID, rawRecords)
	if err != nil {
		return fmt.Errorf("gold copy engine: %w", err)
	}

	// ── 4. Publish to Kafka (non-fatal if publisher not configured)
	if o.publisher != nil && result.Success && result.GoldenRecord != nil {
		changeType := "updated"
		if err := o.publisher.PublishPortfolioMasterGoldCopy(
			ctx, result.GoldenRecord, changeType,
			fmt.Sprintf("Automated ingest for account_type=%s", accountType),
			"system", result.RunID.String(),
		); err != nil {
			log.Printf("[PortfolioOrchestrator] kafka publish warning: %v", err)
		}
		if err := o.publisher.PublishGoldCopyRunResult(ctx, result); err != nil {
			log.Printf("[PortfolioOrchestrator] kafka run-result warning: %v", err)
		}
	}

	if !result.Success {
		return fmt.Errorf("gold copy build failed: %s", result.ErrorMessage)
	}
	return nil
}

// resolveSourceScores returns a map of source → quality score, combining
// account type, asset class, and region routing signals.
// Higher score = higher priority; collisions resolved by taking the max.
func (o *PortfolioOrchestrator) resolveSourceScores(
	accountType, assetClass, region string,
	terms []string,
) map[string]int {
	scores := make(map[string]int)
	for _, term := range terms {
		atSource := sourceByAccountType(accountType, term)
		acSource := sourceByAssetClass(assetClass, term)
		rgSource := sourceByRegion(region, term)

		// Weight: accountType most important (3), then assetClass (2), then region (1)
		addScore(scores, atSource, 3)
		addScore(scores, acSource, 2)
		addScore(scores, rgSource, 1)
	}
	return scores
}

func addScore(m map[string]int, k string, v int) {
	m[k] += v
}

// buildRawRecordsFromSources creates one RawPortfolioRecord per source,
// with quality score proportional to the composite routing score.
// Field values are left empty — in production, this method would call
// the actual source adapters; here we set up the metadata shell so the
// survivorship engine can select based on source priority.
func (o *PortfolioOrchestrator) buildRawRecordsFromSources(
	portfolioID string,
	date time.Time,
	sourceScores map[string]int,
) []*goldcopy.RawPortfolioRecord {
	maxScore := 1
	for _, s := range sourceScores {
		if s > maxScore {
			maxScore = s
		}
	}

	var records []*goldcopy.RawPortfolioRecord
	for src, score := range sourceScores {
		qualityScore := 50 + int(float64(score)/float64(maxScore)*50)
		records = append(records, &goldcopy.RawPortfolioRecord{
			PortfolioID:   portfolioID,
			SourceSystem:  src,
			EffectiveDate: date,
			QualityScore:  qualityScore,
			Fields:        map[string]string{}, // populated by source adapters in production
		})
	}
	return records
}
