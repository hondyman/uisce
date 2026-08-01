package streaming

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
)

type SchemaChangeEvent struct {
	TenantID  string `json:"tenant_id"`
	BOID      string `json:"bo_id"`
	EventType string `json:"event_type"`
	Table     string `json:"table"`
}

type TenantRewarmerAdapter struct {
	engine *rules.RuleEngine
}

func NewTenantRewarmerAdapter(e *rules.RuleEngine) *TenantRewarmerAdapter {
	return &TenantRewarmerAdapter{engine: e}
}

func (a *TenantRewarmerAdapter) RewarmTenantAdapter(ctx context.Context, tenantID uuid.UUID, ruleNodes []*rules.RuleNode, version int) error {
	_, err := a.engine.RewarmTenant(tenantID.String(), ruleNodes, version)
	return err
}

type RuleRepoAdapter struct {
	engine *rules.RuleEngine
}

func NewRuleRepoAdapter(e *rules.RuleEngine) *RuleRepoAdapter {
	return &RuleRepoAdapter{engine: e}
}

func (a *RuleRepoAdapter) GetRulesForTenant(ctx context.Context, tenantID uuid.UUID) ([]*rules.RuleNode, error) {
	return nil, nil
}

type SchemaCDCConsumer struct {
	rewarmer     *TenantRewarmerAdapter
	repo         *RuleRepoAdapter
	debounceSec  int
	pending      sync.Map
	pendingMu    sync.Mutex
}

func NewSchemaCDCConsumer(rewarmer *TenantRewarmerAdapter, repo *RuleRepoAdapter, debounceSec int) *SchemaCDCConsumer {
	return &SchemaCDCConsumer{
		rewarmer:    rewarmer,
		repo:        repo,
		debounceSec: debounceSec,
	}
}

func (c *SchemaCDCConsumer) ProcessSchemaEvent(ctx context.Context, payload []byte) error {
	var evt SchemaChangeEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return err
	}

	switch evt.EventType {
	case "ADD_COLUMN", "RENAME_COLUMN", "ALTER_TABLE":
	default:
		return nil
	}

	tenantID, err := uuid.Parse(evt.TenantID)
	if err != nil {
		return err
	}

	log.Printf("[SchemaCDC] Schema change for tenant %s (Table: %s, Event: %s). Debouncing re-warm for %d seconds...", tenantID, evt.Table, evt.EventType, c.debounceSec)

	key := tenantID.String()

	c.pendingMu.Lock()
	if t, ok := c.pending.Load(key); ok {
		t.(*time.Timer).Stop()
		log.Printf("[SchemaCDC] Reset debounce timer for tenant %s", tenantID)
	}
	timer := time.AfterFunc(time.Duration(c.debounceSec)*time.Second, func() {
		c.pendingMu.Lock()
		c.pending.Delete(key)
		c.pendingMu.Unlock()
		c.doRewarm(context.Background(), tenantID)
	})
	c.pending.Store(key, timer)
	c.pendingMu.Unlock()

	return nil
}

func (c *SchemaCDCConsumer) doRewarm(ctx context.Context, tenantID uuid.UUID) {
	log.Printf("[SchemaCDC] Debounce window elapsed. Rewarming tenant %s...", tenantID)
	ruleNodes, err := c.repo.GetRulesForTenant(ctx, tenantID)
	if err != nil {
		log.Printf("[SchemaCDC] Failed to fetch tenant rules: %v", err)
	}
	if err := c.rewarmer.RewarmTenantAdapter(ctx, tenantID, ruleNodes, 1); err != nil {
		log.Printf("[SchemaCDC] Rewarm failed: %v", err)
	}
}
