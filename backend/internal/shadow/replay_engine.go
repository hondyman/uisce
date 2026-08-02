package shadow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

type ShadowJob struct {
	JobID             uuid.UUID          `json:"job_id"`
	TenantID          uuid.UUID          `json:"tenant_id"`
	DraftRuleID       uuid.UUID          `json:"draft_rule_id"`
	RuleName          string             `json:"rule_name"`
	Status            string             `json:"status"`
	TotalEvaluated    int64              `json:"total_evaluated"`
	ProdPassedCount   int64              `json:"prod_passed_count"`
	ShadowPassedCount int64              `json:"shadow_passed_count"`
	HardBlockCount    int64              `json:"hard_block_count"`
	SoftWarnCount     int64              `json:"soft_warn_count"`
	DiscrepancyCount  int64              `json:"discrepancy_count"`
	CompiledProgram   *vm.CompiledProgram `json:"-"`
	Node              *rules.RuleNode    `json:"-"`
}

type ReplayEngine struct {
	db          *sql.DB
	syms        *vm.SymbolDict
	enums       *vm.EnumDict
	vm          *vm.VM
	activeJobs  sync.Map
}

func NewReplayEngine(db *sql.DB, syms *vm.SymbolDict, enums *vm.EnumDict) *ReplayEngine {
	return &ReplayEngine{
		db:    db,
		syms:  syms,
		enums: enums,
		vm:    vm.NewVM(),
	}
}

func (e *ReplayEngine) StartShadowJob(ctx context.Context, tenantID uuid.UUID, draftRuleID uuid.UUID, ruleName string, node *rules.RuleNode, createdBy string) (*ShadowJob, error) {
	compileRes := rules.CompileVM(node, e.syms, e.enums)
	if compileRes.Unsupported != nil {
		return nil, fmt.Errorf("failed to compile draft rule VM bytecode: %w", compileRes.Unsupported)
	}

	job := &ShadowJob{
		JobID:           uuid.New(),
		TenantID:        tenantID,
		DraftRuleID:     draftRuleID,
		RuleName:        ruleName,
		Status:          "RUNNING",
		CompiledProgram: compileRes.Program,
		Node:            node,
	}

	nodeJSON, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rule node: %w", err)
	}

	query := `
		INSERT INTO public.shadow_replay_jobs
		(job_id, tenant_id, draft_rule_id, rule_name, rule_node_payload, status, created_by)
		VALUES ($1, $2, $3, $4, $5, 'RUNNING', $6)
	`
	err = nil
	if e.db != nil {
		_, err = e.db.ExecContext(ctx, query, job.JobID, tenantID, draftRuleID, ruleName, nodeJSON, createdBy)
		if err != nil {
			return nil, fmt.Errorf("failed to insert shadow replay job: %w", err)
		}
	}

	var jobs []*ShadowJob
	if val, ok := e.activeJobs.Load(tenantID.String()); ok {
		jobs = val.([]*ShadowJob)
	}
	e.activeJobs.Store(tenantID.String(), append(jobs, job))

	return job, nil
}

func (e *ReplayEngine) ProcessShadowOrder(ctx context.Context, tenantID uuid.UUID, tradeID string, prodPassed bool, rawTrade map[string]any) {
	val, ok := e.activeJobs.Load(tenantID.String())
	if !ok {
		return
	}

	jobs := val.([]*ShadowJob)
	if len(jobs) == 0 {
		return
	}

	record := vm.Project(rawTrade, e.syms, e.enums)
	defer vm.PutFastRecord(record)

	for _, job := range jobs {
		if job.Status != "RUNNING" {
			continue
		}

		go func(j *ShadowJob) {
			atomic.AddInt64(&j.TotalEvaluated, 1)
			if prodPassed {
				atomic.AddInt64(&j.ProdPassedCount, 1)
			}

			stack := vm.GetStack()
			defer vm.PutStack(stack)

			shadowPassed := e.vm.Run(j.CompiledProgram, record, stack)
			if shadowPassed {
				atomic.AddInt64(&j.ShadowPassedCount, 1)
			}

			if prodPassed != shadowPassed {
				atomic.AddInt64(&j.DiscrepancyCount, 1)

				if !shadowPassed {
					atomic.AddInt64(&j.HardBlockCount, 1)
				}

				e.logDiff(context.Background(), j.JobID, j.TenantID, tradeID, prodPassed, shadowPassed, rawTrade)
			}
		}(job)
	}
}

func (e *ReplayEngine) logDiff(ctx context.Context, jobID uuid.UUID, tenantID uuid.UUID, tradeID string, prodRes, shadowRes bool, trade map[string]any) {
	if e.db == nil {
		return
	}

	query := `
		INSERT INTO public.shadow_replay_diffs
		(tenant_id, job_id, external_trade_id, prod_result, shadow_result, breach_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	msg := fmt.Sprintf("Shadow rule discrepancy detected for trade %s. Prod: %t, Shadow: %t", tradeID, prodRes, shadowRes)
	_, _ = e.db.ExecContext(ctx, query, tenantID, jobID, tradeID, prodRes, shadowRes, msg)
}

func (e *ReplayEngine) GetImpactReport(ctx context.Context, jobID uuid.UUID) (map[string]any, error) {
	if e.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT job_id, tenant_id, draft_rule_id, rule_name, status, total_evaluated,
		       prod_passed_count, shadow_passed_count, hard_block_count, soft_warn_count, discrepancy_count, started_at
		FROM public.shadow_replay_jobs
		WHERE job_id = $1
	`

	var j ShadowJob
	var startedAt time.Time
	err := e.db.QueryRowContext(ctx, query, jobID).Scan(
		&j.JobID, &j.TenantID, &j.DraftRuleID, &j.RuleName, &j.Status,
		&j.TotalEvaluated, &j.ProdPassedCount, &j.ShadowPassedCount,
		&j.HardBlockCount, &j.SoftWarnCount, &j.DiscrepancyCount, &startedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("shadow job not found: %w", err)
	}

	var falsePositiveRate float64
	if j.TotalEvaluated > 0 {
		falsePositiveRate = (float64(j.DiscrepancyCount) / float64(j.TotalEvaluated)) * 100.0
	}

	return map[string]any{
		"job_id":              j.JobID,
		"rule_name":           j.RuleName,
		"status":              j.Status,
		"total_evaluated":     j.TotalEvaluated,
		"discrepancy_count":   j.DiscrepancyCount,
		"false_positive_pct":  fmt.Sprintf("%.3f%%", falsePositiveRate),
		"hard_blocks_blocked": j.HardBlockCount,
		"started_at":          startedAt,
		"ready_for_promotion": j.DiscrepancyCount == 0 || falsePositiveRate < 0.05,
	}, nil
}

func (e *ReplayEngine) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	if e.db == nil {
		var target *ShadowJob
		e.activeJobs.Range(func(key, value any) bool {
			jobs := value.([]*ShadowJob)
			for _, j := range jobs {
				if j.JobID == jobID {
					target = j
					return false
				}
			}
			return true
		})
		if target != nil {
			target.Status = "CANCELLED"
		}
		return nil
	}

	query := `UPDATE public.shadow_replay_jobs SET status = 'CANCELLED', completed_at = clock_timestamp() WHERE job_id = $1`
	_, err := e.db.ExecContext(ctx, query, jobID)
	return err
}

func (e *ReplayEngine) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	if e.db == nil {
		var target *ShadowJob
		e.activeJobs.Range(func(key, value any) bool {
			jobs := value.([]*ShadowJob)
			for _, j := range jobs {
				if j.JobID == jobID {
					target = j
					return false
				}
			}
			return true
		})
		if target != nil {
			target.Status = "COMPLETED"
		}
		return nil
	}

	query := `UPDATE public.shadow_replay_jobs SET status = 'COMPLETED', completed_at = clock_timestamp() WHERE job_id = $1`
	_, err := e.db.ExecContext(ctx, query, jobID)
	return err
}
