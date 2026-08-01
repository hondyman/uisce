package metadata

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildLookThroughExposureSQL_Valid(t *testing.T) {
	cfg := LookThroughQueryConfig{
		TenantID:       uuid.New(),
		PortfolioID:    "port-123",
		TargetIssuerID: "issuer-AAPL",
		WatermarkDate:  "2026-07-31",
	}
	sql, args, err := BuildLookThroughExposureSQL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 4 {
		t.Errorf("expected 4 positional args, got %d", len(args))
	}
	for _, must := range []string{
		"direct_exposure AS",
		"indirect_exposure AS",
		"portfolio_aum AS",
		"effective_exposure_pct",
		"public.ibor_positions",
		"public.fund_constituents",
		"position_weight_pct",
		"$1", "$2", "$3", "$4",
	} {
		if !strings.Contains(sql, must) {
			t.Errorf("expected SQL to contain %q", must)
		}
	}
}

func TestBuildLookThroughExposureSQL_MissingTenant(t *testing.T) {
	_, _, err := BuildLookThroughExposureSQL(LookThroughQueryConfig{
		PortfolioID:    "p",
		TargetIssuerID: "i",
		WatermarkDate:  "2026-07-31",
	})
	if err == nil {
		t.Error("expected error when tenant_id is missing")
	}
}

func TestBuildLookThroughExposureSQL_MissingPortfolio(t *testing.T) {
	_, _, err := BuildLookThroughExposureSQL(LookThroughQueryConfig{
		TenantID:       uuid.New(),
		TargetIssuerID: "i",
		WatermarkDate:  "2026-07-31",
	})
	if err == nil {
		t.Error("expected error when portfolio_id is missing")
	}
}

func TestBuildLookThroughExposureSQL_MissingTarget(t *testing.T) {
	_, _, err := BuildLookThroughExposureSQL(LookThroughQueryConfig{
		TenantID:      uuid.New(),
		PortfolioID:   "p",
		WatermarkDate: "2026-07-31",
	})
	if err == nil {
		t.Error("expected error when target_issuer_id is missing")
	}
}

func TestBuildLookThroughExposureSQL_MissingWatermark(t *testing.T) {
	_, _, err := BuildLookThroughExposureSQL(LookThroughQueryConfig{
		TenantID:       uuid.New(),
		PortfolioID:    "p",
		TargetIssuerID: "i",
	})
	if err == nil {
		t.Error("expected error when watermark_date is missing")
	}
}
