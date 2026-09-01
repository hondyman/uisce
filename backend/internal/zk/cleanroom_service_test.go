package zk

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestZKCleanRoomService(t *testing.T) {
	svc := NewZKCleanRoomService(nil)

	res, err := svc.ProveAndAttestCovenant(
		context.Background(),
		uuid.New(),
		uuid.New(),
		42500000.0, // OCF
		25000000.0, // Debt Service (DSCR = 1.70x)
		135000000.0, // Debt
		42500000.0, // EBITDA (Lev = 3.17x)
		18200000.0, // Cash
		1.35,
		4.20,
		5000000.0,
	)

	if err != nil {
		t.Fatalf("unexpected error proving covenant: %v", err)
	}

	if !res.VerificationPassed {
		t.Errorf("expected covenant verification to pass")
	}

	if len(res.MerklePassportRoot) == 0 {
		t.Errorf("expected Merkle passport root")
	}

	// DP Laplace Noise Test
	noisyVal, err := svc.ComputeDifferentialPrivacyMetric(
		context.Background(),
		uuid.New(),
		"NORTH_AMERICAN_DIRECT_LENDING_2026",
		14.28,
		1.0,
		0.25,
	)

	if err != nil {
		t.Fatalf("unexpected error computing DP metric: %v", err)
	}

	if noisyVal == 14.28 {
		t.Errorf("expected noisy DP value to differ from true aggregate")
	}
}
