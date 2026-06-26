package scheduler_intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GovernanceService manages scheduler governance
type GovernanceService struct {
	repo           *Repository
	semanticClient SemanticClient
	blastRadius    *BlastRadiusCalculator
}

// NewGovernanceService creates a new governance service
func NewGovernanceService(repo *Repository, semanticClient SemanticClient, br *BlastRadiusCalculator) *GovernanceService {
	return &GovernanceService{
		repo:           repo,
		semanticClient: semanticClient,
		blastRadius:    br,
	}
}

// CreateChangeSet creates a new change set for review
func (g *GovernanceService) CreateChangeSet(ctx context.Context, cs *SchedulerChangeSet) error {
	cs.ID = uuid.New()
	cs.CreatedAt = time.Now()
	cs.UpdatedAt = time.Now()
	cs.Status = ChangeSetStatusPending

	// 1. Analyze impact
	impact := g.analyzeImpact(ctx, cs)
	impactBytes, _ := json.Marshal(impact)
	cs.ImpactAnalysis = impactBytes

	// 2. Risk Evaluation
	cs.RiskScore = g.calculateRiskScore(impact)

	// 3. Semantic Impact Analysis
	semanticImpact := g.computeSemanticImpact(ctx, cs)
	if semanticImpact != nil {
		// Merge semantic impact into metadata for UI
		metadata := make(map[string]interface{})
		if len(cs.Metadata) > 0 {
			_ = json.Unmarshal(cs.Metadata, &metadata)
		}
		metadata["semantic_impact"] = semanticImpact
		mBytes, _ := json.Marshal(metadata)
		cs.Metadata = mBytes

		// Increase risk score based on semantic impact
		if len(semanticImpact.AffectedBOs) > 0 || len(semanticImpact.DownstreamJobs) > 0 {
			cs.RiskScore += 0.2
			if cs.RiskScore > 1.0 {
				cs.RiskScore = 1.0
			}
		}
	}

	// 4. Check Compliance (Dry Run Policy)
	policyIssues := g.CheckCompliance(ctx, cs)
	if len(policyIssues) > 0 {
		metadata := make(map[string]interface{})
		if len(cs.Metadata) > 0 {
			_ = json.Unmarshal(cs.Metadata, &metadata)
		}
		metadata["policy_issues"] = policyIssues
		metadata["requires_dry_run"] = true
		mBytes, _ := json.Marshal(metadata)
		cs.Metadata = mBytes
	}

	// 4. AI Review (Placeholder - will call AI service)
	aiView := AIReview{
		Summary:        "Proposed change for " + cs.Title,
		RiskScore:      cs.RiskScore,
		Recommendation: "Approve if business justification is valid.",
	}
	aiBytes, _ := json.Marshal(aiView)
	cs.AIReview = aiBytes

	// 4. Persistence
	return g.repo.SaveChangeSet(ctx, cs)
}

// analyzeImpact computes the blast radius of a change
func (g *GovernanceService) analyzeImpact(ctx context.Context, cs *SchedulerChangeSet) *ImpactAnalysis {
	impact := &ImpactAnalysis{
		SLOImpact:   false,
		PIIExposure: "none",
	}

	if g.blastRadius != nil && cs.TargetID != nil {
		result, err := g.blastRadius.CalculateForJob(ctx, *cs.TargetID)
		if err == nil {
			impact.BlastRadius = int(result.Score * 100)
			impact.AffectedJobs = []string{cs.TargetID.String()}
			// Map result category or other metrics as needed
		}
	} else {
		// Fallback for new jobs or missing calculator
		switch cs.Type {
		case ChangeSetTypeJobUpdate, ChangeSetTypeJobDelete:
			impact.BlastRadius = 1
			if cs.TargetID != nil {
				impact.AffectedJobs = []string{cs.TargetID.String()}
			}
		case ChangeSetTypeDAGUpdate, ChangeSetTypeDAGDelete:
			impact.BlastRadius = 5
		}
	}

	if cs.TenantID != nil {
		impact.AffectedTenants = []string{cs.TenantID.String()}
	}

	return impact
}

// computeSemanticImpact retrieves semantic dependencies from the graph
func (g *GovernanceService) computeSemanticImpact(ctx context.Context, cs *SchedulerChangeSet) *SemanticImpact {
	if g.semanticClient == nil {
		return nil
	}

	// Collect all semantic IDs from bindings
	var ids []string
	ids = append(ids, cs.SemanticBindings.BOIDs...)
	ids = append(ids, cs.SemanticBindings.APIIDs...)
	ids = append(ids, cs.SemanticBindings.PageIDs...)
	ids = append(ids, cs.SemanticBindings.WorkflowIDs...)
	ids = append(ids, cs.SemanticBindings.PreAggIDs...)

	if len(ids) == 0 {
		return nil
	}

	impacted, err := g.semanticClient.GetImpactedObjects(ctx, ids)
	if err != nil {
		return nil
	}

	res := &SemanticImpact{}
	for _, obj := range impacted {
		switch obj.Type {
		case "BO":
			res.AffectedBOs = append(res.AffectedBOs, obj.ID)
		case "API":
			res.AffectedAPIs = append(res.AffectedAPIs, obj.ID)
		case "PAGE":
			res.AffectedPages = append(res.AffectedPages, obj.ID)
		case "WORKFLOW":
			res.AffectedWorkflows = append(res.AffectedWorkflows, obj.ID)
		case "PREAGG":
			res.AffectedPreAggs = append(res.AffectedPreAggs, obj.ID)
		case "JOB":
			res.DownstreamJobs = append(res.DownstreamJobs, obj.ID)
		case "DAG":
			res.DownstreamDAGs = append(res.DownstreamDAGs, obj.ID)
		}
	}

	return res
}

// calculateRiskScore computes a risk score for the change
func (g *GovernanceService) calculateRiskScore(impact *ImpactAnalysis) float64 {
	score := 0.1

	if impact.BlastRadius > 10 {
		score += 0.3
	}
	if impact.SLOImpact {
		score += 0.4
	}
	if impact.PIIExposure != "none" {
		score += 0.2
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// CheckCompliance verifies if the changeset meets governance policies
func (g *GovernanceService) CheckCompliance(ctx context.Context, cs *SchedulerChangeSet) []string {
	var issues []string

	// Policy 1: High Risk requires Dry Run
	if cs.RiskScore >= 0.7 {
		issues = append(issues, "High risk change requires dry-run verification")
	}

	// Policy 2: SLO Critical requires Ops Approval (handled in approval flow, but noted here)
	impact := &ImpactAnalysis{}
	if len(cs.ImpactAnalysis) > 0 {
		_ = json.Unmarshal(cs.ImpactAnalysis, impact)
		if impact.SLOImpact {
			issues = append(issues, "SLO-impacting change requires Ops approval")
		}
	}

	return issues
}

// ApproveChangeSet records an approval
func (g *GovernanceService) ApproveChangeSet(ctx context.Context, changeSetID uuid.UUID, approverID, role, comment string) error {
	approval := ChangeSetApproval{
		ID:           uuid.New(),
		ChangeSetID:  changeSetID,
		ApproverID:   approverID,
		ApproverRole: role,
		Decision:     "approved",
		Comment:      comment,
		CreatedAt:    time.Now(),
	}

	if err := g.repo.SaveApproval(ctx, &approval); err != nil {
		return err
	}

	// Update changeset status if thresholds met (simple 1-approval rule for now)
	cs, err := g.repo.GetChangeSet(ctx, changeSetID)
	if err != nil {
		return err
	}

	cs.Status = ChangeSetStatusApproved
	return g.repo.SaveChangeSet(ctx, cs)
}

// RejectChangeSet records a rejection
func (g *GovernanceService) RejectChangeSet(ctx context.Context, changeSetID uuid.UUID, approverID, role, reason string) error {
	approval := ChangeSetApproval{
		ID:           uuid.New(),
		ChangeSetID:  changeSetID,
		ApproverID:   approverID,
		ApproverRole: role,
		Decision:     "rejected",
		Comment:      reason,
		CreatedAt:    time.Now(),
	}

	if err := g.repo.SaveApproval(ctx, &approval); err != nil {
		return err
	}

	cs, err := g.repo.GetChangeSet(ctx, changeSetID)
	if err != nil {
		return err
	}

	cs.Status = ChangeSetStatusRejected
	return g.repo.SaveChangeSet(ctx, cs)
}

// ApplyChangeSet executes an approved change
func (g *GovernanceService) ApplyChangeSet(ctx context.Context, changeSetID uuid.UUID) error {
	cs, err := g.repo.GetChangeSet(ctx, changeSetID)
	if err != nil {
		return err
	}

	if cs.Status != ChangeSetStatusApproved {
		return fmt.Errorf("changeset must be approved to apply (current status: %s)", cs.Status)
	}

	var diff map[string]json.RawMessage
	if err := json.Unmarshal(cs.Diff, &diff); err != nil {
		return fmt.Errorf("failed to unmarshal diff: %w", err)
	}

	switch cs.Type {
	case ChangeSetTypeJobCreate:
		var job Job
		if err := json.Unmarshal(diff["new"], &job); err != nil {
			return fmt.Errorf("failed to unmarshal job: %w", err)
		}
		job.ID = uuid.New()
		job.ChangeSetID = &cs.ID
		job.IsActive = true
		if err := g.repo.CreateJob(ctx, &job); err != nil {
			return fmt.Errorf("failed to apply job creation: %w", err)
		}

	case ChangeSetTypeJobUpdate:
		if cs.TargetID == nil {
			return fmt.Errorf("target_id is required for job update")
		}
		job, err := g.repo.GetJob(ctx, *cs.TargetID)
		if err != nil {
			return fmt.Errorf("failed to fetch target job: %w", err)
		}

		var req UpdateJobRequest
		if err := json.Unmarshal(diff["new"], &req); err != nil {
			return fmt.Errorf("failed to unmarshal update request: %w", err)
		}

		// Apply updates manually (mirroring Service.UpdateJob logic)
		if req.Name != nil {
			job.Name = *req.Name
		}
		if req.Description != nil {
			job.Description = *req.Description
		}
		if req.Category != nil {
			job.Category = *req.Category
		}
		if req.ScheduleType != nil {
			job.ScheduleType = *req.ScheduleType
		}
		if req.CronExpression != nil {
			job.CronExpression = req.CronExpression
		}
		if req.Timezone != nil {
			job.Timezone = *req.Timezone
		}
		if req.Priority != nil {
			job.Priority = *req.Priority
		}
		if req.IsActive != nil {
			job.IsActive = *req.IsActive
		}
		if req.Parameters != nil {
			p, _ := json.Marshal(req.Parameters)
			job.Parameters = p
		}

		if err := g.repo.UpdateJob(ctx, job); err != nil {
			return fmt.Errorf("failed to apply job update: %w", err)
		}

	case ChangeSetTypeJobDelete:
		if cs.TargetID == nil {
			return fmt.Errorf("target_id is required for job delete")
		}
		if err := g.repo.DeleteJob(ctx, *cs.TargetID); err != nil {
			return fmt.Errorf("failed to apply job deletion: %w", err)
		}

	case ChangeSetTypeDAGCreate:
		var dag DAG
		if err := json.Unmarshal(diff["new"], &dag); err != nil {
			return fmt.Errorf("failed to unmarshal dag: %w", err)
		}
		dag.ID = uuid.New()
		dag.ChangeSetID = &cs.ID
		dag.IsActive = true
		if err := g.repo.CreateDAG(ctx, &dag); err != nil {
			return fmt.Errorf("failed to apply dag creation: %w", err)
		}

	case ChangeSetTypeDAGUpdate:
		if cs.TargetID == nil {
			return fmt.Errorf("target_id is required for dag update")
		}
		dag, err := g.repo.GetDAG(ctx, *cs.TargetID)
		if err != nil {
			return fmt.Errorf("failed to fetch target dag: %w", err)
		}

		var req UpdateDAGRequest
		if err := json.Unmarshal(diff["new"], &req); err != nil {
			return fmt.Errorf("failed to unmarshal update request: %w", err)
		}

		if req.Name != nil {
			dag.Name = *req.Name
		}
		if req.Description != nil {
			dag.Description = *req.Description
		}
		if req.Category != nil {
			dag.Category = req.Category
		}
		if req.ScheduleType != nil {
			dag.ScheduleType = req.ScheduleType
		}
		if req.CronExpression != nil {
			dag.CronExpression = req.CronExpression
		}
		if req.Timezone != nil {
			dag.Timezone = *req.Timezone
		}
		if req.Nodes != nil {
			n, _ := json.Marshal(req.Nodes)
			dag.Nodes = n
		}
		if req.Edges != nil {
			e, _ := json.Marshal(req.Edges)
			dag.Edges = e
		}

		if err := g.repo.UpdateDAG(ctx, dag); err != nil {
			return fmt.Errorf("failed to apply dag update: %w", err)
		}

	case ChangeSetTypeDAGDelete:
		if cs.TargetID == nil {
			return fmt.Errorf("target_id is required for dag delete")
		}
		if err := g.repo.DeleteDAG(ctx, *cs.TargetID); err != nil {
			return fmt.Errorf("failed to apply dag deletion: %w", err)
		}

	default:
		return fmt.Errorf("unsupported changeset type: %s", cs.Type)
	}

	cs.Status = ChangeSetStatusApplied
	return g.repo.SaveChangeSet(ctx, cs)
}

// RollbackChangeSet reverts an applied change
func (g *GovernanceService) RollbackChangeSet(ctx context.Context, changeSetID uuid.UUID, reason string) error {
	cs, err := g.repo.GetChangeSet(ctx, changeSetID)
	if err != nil {
		return err
	}
	cs.Status = ChangeSetStatusRolledBack
	return g.repo.SaveChangeSet(ctx, cs)
}

// ListChangeSets returns change sets for a tenant
// ListChangeSets lists changesets for a tenant
func (g *GovernanceService) ListChangeSets(ctx context.Context, tenantID uuid.UUID, status *ChangeSetStatus) ([]SchedulerChangeSet, error) {
	return g.repo.ListChangeSets(ctx, tenantID, status)
}

// GetChangeSet returns a single change set
func (g *GovernanceService) GetChangeSet(ctx context.Context, id uuid.UUID) (*SchedulerChangeSet, error) {
	return g.repo.GetChangeSet(ctx, id)
}
