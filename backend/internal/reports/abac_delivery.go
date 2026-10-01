package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// RecipientProfile represents a report recipient with their specific ABAC security scope.
type RecipientProfile struct {
	UserID          string   `json:"userId"`
	Email           string   `json:"email"`
	TenantID        string   `json:"tenantId"`
	Roles           []string `json:"roles"`
	PermittedFields []string `json:"permittedFields,omitempty"` // Attribute-level visibility mask
	MaskedFields    []string `json:"maskedFields,omitempty"`
}

// ComputeABACContextHash returns a deterministic SHA-256 hash identifying a recipient's unique security context.
// Recipients with different attribute-level masking or role scopes will yield distinct hashes.
func ComputeABACContextHash(r RecipientProfile) string {
	roles := append([]string{}, r.Roles...)
	sort.Strings(roles)

	permitted := append([]string{}, r.PermittedFields...)
	sort.Strings(permitted)

	masked := append([]string{}, r.MaskedFields...)
	sort.Strings(masked)

	payload := fmt.Sprintf("%s:%s:%s:%s",
		r.TenantID,
		strings.Join(roles, ","),
		strings.Join(permitted, ","),
		strings.Join(masked, ","),
	)

	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

// ScheduledReportDeliveryPlan groups recipients by their unique ABAC context hash so that
// distinct masked outputs are rendered per security boundary without cross-recipient data leakage.
type ScheduledReportDeliveryPlan struct {
	ScheduleID      string                        `json:"scheduleId"`
	IdempotencyKey  string                        `json:"idempotencyKey"`
	FireTimestamp   time.Time                     `json:"fireTimestamp"`
	FixedParameters map[string]interface{}        `json:"fixedParameters,omitempty"`
	RecipientGroups map[string][]RecipientProfile `json:"recipientGroups"` // Keyed by ABACContextHash
}

// BuildDeliveryPlan constructs an idempotent delivery plan partitioning recipients by distinct ABAC context.
func BuildDeliveryPlan(scheduleID string, fireTime time.Time, fixedParams map[string]interface{}, recipients []RecipientProfile) *ScheduledReportDeliveryPlan {
	idempotencyKey := fmt.Sprintf("%s:%d", scheduleID, fireTime.Unix())
	groups := make(map[string][]RecipientProfile)

	for _, r := range recipients {
		hash := ComputeABACContextHash(r)
		groups[hash] = append(groups[hash], r)
	}

	return &ScheduledReportDeliveryPlan{
		ScheduleID:      scheduleID,
		IdempotencyKey:  idempotencyKey,
		FireTimestamp:   fireTime,
		FixedParameters: fixedParams,
		RecipientGroups: groups,
	}
}

// DeliveryResult represents the outcome of an idempotent scheduled report render.
type DeliveryResult struct {
	IdempotencyKey string          `json:"idempotencyKey"`
	ExecutedAt     time.Time       `json:"executedAt"`
	RendersCount   int             `json:"rendersCount"`
	RecipientsSent int             `json:"recipientsSent"`
	Renders        map[string]bool `json:"renders"` // ABACContextHash -> rendered
}

// DeliveryOrchestrator manages per-recipient ABAC report rendering and delivery deduplication.
type DeliveryOrchestrator struct {
	mu          sync.Mutex
	executedMap map[string]*DeliveryResult
}

// NewDeliveryOrchestrator creates a new delivery orchestrator.
func NewDeliveryOrchestrator() *DeliveryOrchestrator {
	return &DeliveryOrchestrator{
		executedMap: make(map[string]*DeliveryResult),
	}
}

// ExecuteDelivery processes a delivery plan with idempotency checking and per-ABAC-hash isolated rendering.
func (o *DeliveryOrchestrator) ExecuteDelivery(
	ctx context.Context,
	plan *ScheduledReportDeliveryPlan,
	renderFn func(ctx context.Context, abacHash string, recipients []RecipientProfile, params map[string]interface{}) ([]byte, error),
) (*DeliveryResult, error) {
	o.mu.Lock()
	if existing, found := o.executedMap[plan.IdempotencyKey]; found {
		o.mu.Unlock()
		return existing, nil // Return cached result for duplicate scheduler fires
	}
	o.mu.Unlock()

	renders := make(map[string]bool)
	totalRecipients := 0

	// Execute one render per distinct ABAC context hash (guarantees masked data is not shared)
	for abacHash, recipientList := range plan.RecipientGroups {
		_, err := renderFn(ctx, abacHash, recipientList, plan.FixedParameters)
		if err != nil {
			return nil, fmt.Errorf("failed to render for ABAC context %s: %w", abacHash, err)
		}
		renders[abacHash] = true
		totalRecipients += len(recipientList)
	}

	result := &DeliveryResult{
		IdempotencyKey: plan.IdempotencyKey,
		ExecutedAt:     time.Now().UTC(),
		RendersCount:   len(plan.RecipientGroups),
		RecipientsSent: totalRecipients,
		Renders:        renders,
	}

	o.mu.Lock()
	o.executedMap[plan.IdempotencyKey] = result
	o.mu.Unlock()

	return result, nil
}
