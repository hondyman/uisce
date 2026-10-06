package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hondyman/uisce/backend/internal/rules"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// DecisionStatus represents the final compliance outcome
type DecisionStatus string

const (
	StatusPass             DecisionStatus = "PASS"
	StatusSoftWarning      DecisionStatus = "SOFT_WARNING"
	StatusApprovalRequired DecisionStatus = "APPROVAL_REQUIRED"
	StatusHardBlock        DecisionStatus = "HARD_BLOCK"
)

// SeverityLevel defines the enforcement severity of a rule
type SeverityLevel string

const (
	SeverityHardBlock        SeverityLevel = "HARD_BLOCK"
	SeverityApprovalRequired SeverityLevel = "APPROVAL_REQUIRED"
	SeveritySoftWarning      SeverityLevel = "SOFT_WARNING"
	SeverityDMABypassToken   SeverityLevel = "DMA_BYPASS_TOKEN"
)

// PreTradeCompiledRule represents a single compiled rule inside a bundle
type PreTradeCompiledRule struct {
	RuleID         uuid.UUID
	RuleCode       string
	RuleName       string
	Severity       SeverityLevel
	Program        *vm.CompiledProgram
	AST            *rules.RuleNode
	ThresholdValue decimal.Decimal
	Description    string
}

// RuleBundle represents an ordered collection of compiled rules executed sequentially
type RuleBundle struct {
	BundleID    uuid.UUID
	TenantID    uuid.UUID
	BundleName  string
	Version     int64
	Rules       []PreTradeCompiledRule
	SymDict     *vm.SymbolDict
	EnumDict    *vm.EnumDict
	mu          sync.RWMutex
}

// NewRuleBundle creates a new compiled rule bundle
func NewRuleBundle(tenantID uuid.UUID, bundleName string, version int64) *RuleBundle {
	return &RuleBundle{
		BundleID:   uuid.New(),
		TenantID:   tenantID,
		BundleName: bundleName,
		Version:    version,
		Rules:      make([]PreTradeCompiledRule, 0, 32),
		SymDict:    vm.NewSymbolDict(),
		EnumDict:   vm.NewEnumDict(),
	}
}

// AddCompiledRule registers a compiled rule into the bundle
func (b *RuleBundle) AddCompiledRule(rule PreTradeCompiledRule) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Rules = append(b.Rules, rule)
}

// RuleEvaluationDetail captures the evaluation of an individual rule within a bundle
type RuleEvaluationDetail struct {
	RuleID         uuid.UUID       `json:"rule_id"`
	RuleCode       string          `json:"rule_code"`
	Passed         bool            `json:"passed"`
	Severity       SeverityLevel   `json:"severity"`
	ThresholdValue decimal.Decimal `json:"threshold_value"`
	Message        string          `json:"message,omitempty"`
}

// BundleEvaluationOutcome represents the complete result of executing a rule bundle
type BundleEvaluationOutcome struct {
	LineageID     uuid.UUID              `json:"lineage_id"`
	TenantID      uuid.UUID              `json:"tenant_id"`
	AccountID     uuid.UUID              `json:"account_id"`
	Decision      DecisionStatus         `json:"decision"`
	TotalRules    int                    `json:"total_rules"`
	PassedRules   int                    `json:"passed_rules"`
	BreachedRules int                    `json:"breached_rules"`
	Details       []RuleEvaluationDetail `json:"details"`
	EvaluatedAt   time.Time              `json:"evaluated_at"`
	DurationNs    int64                  `json:"duration_ns"`
}

// BundleCompiler compiles declarative rule definitions into high-performance VM rule bundles
type BundleCompiler struct {
	vm *vm.VM
}

// NewBundleCompiler creates a bundle compiler
func NewBundleCompiler() *BundleCompiler {
	return &BundleCompiler{
		vm: vm.NewVM(),
	}
}

// CompileRule converts an AST RuleNode into a PreTradeCompiledRule
func (c *BundleCompiler) CompileRule(
	ruleID uuid.UUID,
	ruleCode, ruleName string,
	severity SeverityLevel,
	ast *rules.RuleNode,
	syms *vm.SymbolDict,
	enums *vm.EnumDict,
	threshold decimal.Decimal,
) (PreTradeCompiledRule, error) {
	res := rules.CompileVM(ast, syms, enums)
	if res.Unsupported != nil {
		return PreTradeCompiledRule{}, fmt.Errorf("failed to compile rule %s (%s): %w", ruleCode, ruleID, res.Unsupported)
	}
	if res.Program == nil {
		return PreTradeCompiledRule{}, errors.New("compilation returned nil program")
	}

	return PreTradeCompiledRule{
		RuleID:         ruleID,
		RuleCode:       ruleCode,
		RuleName:       ruleName,
		Severity:       severity,
		Program:        res.Program,
		AST:            ast,
		ThresholdValue: threshold,
	}, nil
}

// FastBundleEvaluator executes pre-trade rule bundles with zero heap allocations in the hot evaluation path
type FastBundleEvaluator struct {
	vm *vm.VM
}

// NewFastBundleEvaluator creates a new zero-allocation bundle evaluator
func NewFastBundleEvaluator() *FastBundleEvaluator {
	return &FastBundleEvaluator{
		vm: vm.NewVM(),
	}
}

// EvaluateBundle executes all rules in the bundle sequentially using pooled operand stacks
func (e *FastBundleEvaluator) EvaluateBundle(
	ctx context.Context,
	lineageID, tenantID, accountID uuid.UUID,
	bundle *RuleBundle,
	record *vm.FastRecord,
) BundleEvaluationOutcome {
	start := time.Now()
	nowUTC := start.UTC()

	numRules := len(bundle.Rules)
	outcome := BundleEvaluationOutcome{
		LineageID:   lineageID,
		TenantID:    tenantID,
		AccountID:   accountID,
		Decision:    StatusPass,
		TotalRules:  numRules,
		Details:     make([]RuleEvaluationDetail, numRules),
		EvaluatedAt: nowUTC,
	}

	// Acquire pooled operand stack for zero-allocation execution
	stack := vm.GetStack()
	defer vm.PutStack(stack)

	passedCount := 0
	breachedCount := 0
	highestSeverity := StatusPass

	for i := 0; i < numRules; i++ {
		r := &bundle.Rules[i]
		stack.Reset()

		// Run bytecode on VM
		passed := e.vm.Run(r.Program, record, stack)

		detail := RuleEvaluationDetail{
			RuleID:         r.RuleID,
			RuleCode:       r.RuleCode,
			Passed:         passed,
			Severity:       r.Severity,
			ThresholdValue: r.ThresholdValue,
		}

		if passed {
			passedCount++
		} else {
			breachedCount++
			detail.Message = fmt.Sprintf("Rule %s breached with severity %s", r.RuleCode, r.Severity)

			// Escalate decision status
			switch r.Severity {
			case SeverityHardBlock:
				highestSeverity = StatusHardBlock
			case SeverityApprovalRequired:
				if highestSeverity != StatusHardBlock {
					highestSeverity = StatusApprovalRequired
				}
			case SeveritySoftWarning:
				if highestSeverity != StatusHardBlock && highestSeverity != StatusApprovalRequired {
					highestSeverity = StatusSoftWarning
				}
			case SeverityDMABypassToken:
				if highestSeverity != StatusHardBlock && highestSeverity != StatusApprovalRequired {
					highestSeverity = StatusSoftWarning
				}
			}
		}

		outcome.Details[i] = detail
	}

	outcome.Decision = highestSeverity
	outcome.PassedRules = passedCount
	outcome.BreachedRules = breachedCount
	outcome.DurationNs = time.Since(start).Nanoseconds()

	return outcome
}

// EvaluateBundleFast executes all rules with zero allocations by accepting a pre-allocated details slice
func (e *FastBundleEvaluator) EvaluateBundleFast(
	bundle *RuleBundle,
	record *vm.FastRecord,
	outDetails []RuleEvaluationDetail,
) (DecisionStatus, int, int) {
	numRules := len(bundle.Rules)
	stack := vm.GetStack()
	defer vm.PutStack(stack)

	passedCount := 0
	breachedCount := 0
	highestSeverity := StatusPass

	for i := 0; i < numRules; i++ {
		r := &bundle.Rules[i]
		stack.Reset()

		passed := e.vm.Run(r.Program, record, stack)

		if i < len(outDetails) {
			outDetails[i].RuleID = r.RuleID
			outDetails[i].RuleCode = r.RuleCode
			outDetails[i].Passed = passed
			outDetails[i].Severity = r.Severity
			outDetails[i].ThresholdValue = r.ThresholdValue
			outDetails[i].Message = ""
		}

		if passed {
			passedCount++
		} else {
			breachedCount++
			switch r.Severity {
			case SeverityHardBlock:
				highestSeverity = StatusHardBlock
			case SeverityApprovalRequired:
				if highestSeverity != StatusHardBlock {
					highestSeverity = StatusApprovalRequired
				}
			case SeveritySoftWarning, SeverityDMABypassToken:
				if highestSeverity != StatusHardBlock && highestSeverity != StatusApprovalRequired {
					highestSeverity = StatusSoftWarning
				}
			}
		}
	}

	return highestSeverity, passedCount, breachedCount
}
