package scheduler_intelligence

import (
	"context"
	"math"

	"github.com/google/uuid"
)

// BlastRadiusCategory represents the severity of a change's impact
type BlastRadiusCategory string

const (
	BlastRadiusLow    BlastRadiusCategory = "LOW"
	BlastRadiusMedium BlastRadiusCategory = "MEDIUM"
	BlastRadiusHigh   BlastRadiusCategory = "HIGH"
)

// BlastRadiusResult contains the calculated score and supporting metrics
type BlastRadiusResult struct {
	Score    float64             `json:"blast_radius_score"`
	Category BlastRadiusCategory `json:"blast_radius_category"`
	Metrics  BlastRadiusMetrics  `json:"metrics"`
}

type BlastRadiusMetrics struct {
	DownstreamJobsCount int      `json:"downstream_jobs_count"`
	DownstreamDAGsCount int      `json:"downstream_dags_count"`
	SemanticImpactCount int      `json:"semantic_impact_count"`
	AffectedTenants     int      `json:"affected_tenants_count"`
	AffectedBOs         []string `json:"affected_bos,omitempty"`
}

// BlastRadiusCalculator computes the potential impact of a scheduler change
type BlastRadiusCalculator struct {
	repo           *Repository
	semanticClient SemanticClient
}

// NewBlastRadiusCalculator creates a new calculator
func NewBlastRadiusCalculator(repo *Repository, sc SemanticClient) *BlastRadiusCalculator {
	return &BlastRadiusCalculator{
		repo:           repo,
		semanticClient: sc,
	}
}

// CalculateForJob computes blast radius for a specific job change
func (c *BlastRadiusCalculator) CalculateForJob(ctx context.Context, jobID uuid.UUID) (*BlastRadiusResult, error) {
	job, err := c.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	metrics := BlastRadiusMetrics{}

	// 1. Check downstream jobs
	downstreamJobs, err := c.repo.GetDownstreamJobs(ctx, jobID)
	if err == nil {
		metrics.DownstreamJobsCount = len(downstreamJobs)
	}

	// 2. Check containing DAGs
	dags, err := c.repo.GetDAGsContainingJob(ctx, jobID)
	if err == nil {
		metrics.DownstreamDAGsCount = len(dags)
	}

	// 3. Check semantic impact
	if c.semanticClient != nil && job.SemanticBindings.HasAny() {
		impact, err := c.semanticClient.GetImpactedObjects(ctx, job.SemanticBindings.ToIDList())
		if err == nil {
			metrics.SemanticImpactCount = len(impact)
			// Deduplicate BOs
			boMap := make(map[string]bool)
			for _, obj := range impact {
				if obj.Type == "business_object" {
					boMap[obj.ID] = true
				}
			}
			for boID := range boMap {
				metrics.AffectedBOs = append(metrics.AffectedBOs, boID)
			}
		}
	}

	// For single-tenant jobs, affected tenants is always 1 unless it's a GLOBAL job
	if job.Scope == "GLOBAL" {
		// In a real system, we might query which tenants have overlays for this global job
		// For now, we'll estimate high impact for global changes
		metrics.AffectedTenants = 10 // Placeholder for high impact
	} else {
		metrics.AffectedTenants = 1
	}

	score := c.computeScore(metrics)

	return &BlastRadiusResult{
		Score:    score,
		Category: c.categorize(score),
		Metrics:  metrics,
	}, nil
}

func (c *BlastRadiusCalculator) computeScore(m BlastRadiusMetrics) float64 {
	// Simple weighted score (normalized roughly to 0-1)
	// Weights:
	// - Downstream Job: 0.1
	// - Downstream DAG: 0.2
	// - Semantic Impact (BO/API): 0.1
	// - Global Tenant impact multiplier

	rawScore := float64(m.DownstreamJobsCount)*0.1 +
		float64(m.DownstreamDAGsCount)*0.2 +
		float64(m.SemanticImpactCount)*0.1

	if m.AffectedTenants > 1 {
		rawScore *= float64(m.AffectedTenants) * 0.5
	}

	// Sigmoid-like normalization to keep it 0-1
	return 1.0 / (1.0 + math.Exp(-rawScore+2))
}

func (c *BlastRadiusCalculator) categorize(score float64) BlastRadiusCategory {
	if score >= 0.7 {
		return BlastRadiusHigh
	}
	if score >= 0.3 {
		return BlastRadiusMedium
	}
	return BlastRadiusLow
}
