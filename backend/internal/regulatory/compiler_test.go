package regulatory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRegulatoryCompilerService(t *testing.T) {
	svc := NewRegulatoryCompilerService(nil)

	tenantID := uuid.New()
	portfolioID := uuid.New()

	res, err := svc.CompileSEC13FFiling(
		context.Background(),
		tenantID,
		portfolioID,
		time.Now(),
	)

	if err != nil {
		t.Fatalf("unexpected error compiling 13F filing: %v", err)
	}

	if res.TotalQualifyingHoldings <= 0 {
		t.Errorf("expected qualifying holdings > 0, got %d", res.TotalQualifyingHoldings)
	}

	if len(res.XMLPayload) == 0 {
		t.Errorf("expected XML payload to be generated")
	}

	if !res.ValidationPass {
		t.Errorf("expected validation to pass, errors: %v", res.ValidationErrors)
	}

	if len(res.MerkleRootSeal) == 0 {
		t.Errorf("expected Merkle root seal to be generated")
	}

	err = svc.AttestAndSealFiling(
		context.Background(),
		tenantID,
		res.RunID,
		"usr_cco_4491",
		"CHIEF_COMPLIANCE_OFFICER",
		"192.168.1.50",
	)

	if err != nil {
		t.Fatalf("unexpected error attesting filing: %v", err)
	}
}
