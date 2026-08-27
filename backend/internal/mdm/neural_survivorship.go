package mdm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CompetingFeedTick struct {
	VendorCode string    `json:"vendorCode"`
	RawValue   float64   `json:"rawValue"`
	FeedTime   time.Time `json:"feedTime"`
}

type NeuralMasterResolution struct {
	WinningVendor  string  `json:"winningVendor"`
	MasterValue    float64 `json:"masterValue"`
	CompositeScore float64 `json:"compositeScore"`
	MerkleLeafHash string  `json:"merkleLeafHash"`
}

type NeuralSurvivorshipEngine struct {
	db *sqlx.DB
}

func NewNeuralSurvivorshipEngine(db *sqlx.DB) *NeuralSurvivorshipEngine {
	return &NeuralSurvivorshipEngine{db: db}
}

// ResolveNeuralGoldenRecord computes time-decayed consensus trust scores and seals the decision into the Merkle ledger
func (e *NeuralSurvivorshipEngine) ResolveNeuralGoldenRecord(
	ctx context.Context,
	tenantID, entityID uuid.UUID,
	domainKey string,
	ticks []CompetingFeedTick,
) (*NeuralMasterResolution, error) {
	if len(ticks) == 0 {
		return nil, fmt.Errorf("no vendor ticks provided for entity %s", entityID)
	}

	// 1. Fetch decay profiles (Rule 1: Config-Before-Code)
	var profiles []struct {
		VendorCode         string  `db:"vendor_code"`
		BaseTrustScore     float64 `db:"base_trust_score"`
		DecayLambda        float64 `db:"decay_lambda"`
		HistoricalAccuracy float64 `db:"historical_accuracy_pct"`
	}

	query := `
		SELECT vendor_code, base_trust_score, decay_lambda, historical_accuracy_pct
		FROM catalog_mdm_neural.source_decay_profiles
		WHERE tenant_id = $1 AND domain_key = $2;
	`
	_ = e.db.SelectContext(ctx, &profiles, query, tenantID, domainKey)

	profileMap := make(map[string]struct{ Base, Lambda, Acc float64 })
	for _, p := range profiles {
		profileMap[p.VendorCode] = struct{ Base, Lambda, Acc float64 }{
			Base: p.BaseTrustScore, Lambda: p.DecayLambda, Acc: p.HistoricalAccuracy,
		}
	}

	now := time.Now().UTC()
	var bestVendor string
	var bestValue, maxScore float64

	// 2. Score feeds: Trust = Base * (Accuracy / 100) * e^(-lambda * dt)
	for _, t := range ticks {
		prof, exists := profileMap[t.VendorCode]
		if !exists {
			prof.Base = 0.85
			prof.Lambda = 0.0002
			prof.Acc = 98.0
		}

		dtSeconds := math.Max(0.0, now.Sub(t.FeedTime).Seconds())
		trustScore := prof.Base * (prof.Acc / 100.0) * math.Exp(-prof.Lambda*dtSeconds)

		if trustScore > maxScore {
			maxScore = trustScore
			bestVendor = t.VendorCode
			bestValue = t.RawValue
		}
	}

	// 3. Cryptographic Merkle Leaf Construction (SEC Rule 17a-4 / FINRA Audit)
	leafPayload := fmt.Sprintf("%s:%s:%s:%.4f:%.4f:%s",
		tenantID.String(), entityID.String(), bestVendor, bestValue, maxScore, now.Format(time.RFC3339))
	h := sha256.Sum256([]byte(leafPayload))
	leafHash := hex.EncodeToString(h[:])

	insertMerkle := `
		INSERT INTO catalog_governance.cryptographic_merkle_ledger (
			tenant_id, entity_type, entity_id, leaf_hash, merkle_root_hash, tree_depth, sealed_by, sealed_at
		) VALUES ($1, 'GOLDEN_RECORD', $2, $3, $3, 1, 'SYSTEM_NEURAL_MDM', NOW());
	`
	_, _ = e.db.ExecContext(ctx, insertMerkle, tenantID, entityID, leafHash)

	return &NeuralMasterResolution{
		WinningVendor:  bestVendor,
		MasterValue:    bestValue,
		CompositeScore: maxScore,
		MerkleLeafHash: leafHash,
	}, nil
}
