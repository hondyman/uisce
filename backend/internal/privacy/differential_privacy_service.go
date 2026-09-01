package privacy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type PrivacyBudget struct {
	Epsilon float64 `json:"epsilon"` // e.g. 0.5 for high privacy
	Delta   float64 `json:"delta"`   // e.g. 1e-5
}

type ObfuscatedMetricResult struct {
	MetricName     string  `json:"metricName"`
	RawValue       float64 `json:"rawValue,omitempty"`
	NoisyValue     float64 `json:"noisyValue"`
	EpsilonUsed    float64 `json:"epsilonUsed"`
	Sensitivity    float64 `json:"sensitivity"`
	ZKPassportHash string  `json:"zkPassportHash"`
	CertifiedAt    time.Time `json:"certifiedAt"`
}

type DifferentialPrivacyService struct {
	rng *rand.Rand
}

func NewDifferentialPrivacyService() *DifferentialPrivacyService {
	return &DifferentialPrivacyService{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ApplyLaplaceNoise adds calibrated Laplace(0, deltaF / epsilon) noise to an aggregated benchmark metric
func (s *DifferentialPrivacyService) ApplyLaplaceNoise(
	ctx context.Context,
	tenantID uuid.UUID,
	metricName string,
	rawValue float64,
	sensitivity float64,
	budget PrivacyBudget,
) (*ObfuscatedMetricResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if budget.Epsilon <= 0 {
		budget.Epsilon = 0.5
	}
	if sensitivity <= 0 {
		sensitivity = 1.0
	}

	// Scale parameter b = sensitivity / epsilon
	scale := sensitivity / budget.Epsilon

	// Sample from standard uniform (-0.5, 0.5)
	u := s.rng.Float64() - 0.5
	var noise float64
	if u < 0 {
		noise = scale * math.Log(1.0+2.0*u)
	} else {
		noise = -scale * math.Log(1.0-2.0*u)
	}

	noisyValue := rawValue + noise

	// Generate cryptographic ZK compliance passport hash
	passportPayload := fmt.Sprintf("%s:%s:%.4f:%.2f:%d", tenantID, metricName, noisyValue, budget.Epsilon, time.Now().Unix())
	hash := sha256.Sum256([]byte(passportPayload))
	passportHash := hex.EncodeToString(hash[:])

	return &ObfuscatedMetricResult{
		MetricName:     metricName,
		NoisyValue:     noisyValue,
		EpsilonUsed:    budget.Epsilon,
		Sensitivity:    sensitivity,
		ZKPassportHash: passportHash,
		CertifiedAt:    time.Now().UTC(),
	}, nil
}
