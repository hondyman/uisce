package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/segmentio/kafka-go"
)

type ReferenceDataFetcher interface {
	GetPortfolioReferenceState(ctx context.Context, tenantID uuid.UUID, portfolioID string, isin string) (map[string]any, error)
	GetExternalMapping(ctx context.Context, tenantID uuid.UUID, systemID string) (map[string]string, error)
	GetRuleChain(ctx context.Context, tenantID uuid.UUID, chainID string) (*rules.RuleChain, error)
}

func NewExternalComplianceHandler(
	engine *rules.RuleEngine,
	fetcher ReferenceDataFetcher,
	redisClient *redis.Client,
	db *sql.DB,
	kafkaBrokers string,
) *ExternalComplianceHandler {
	var writer *kafka.Writer
	if kafkaBrokers != "" {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(strings.Split(kafkaBrokers, ",")...),
			Topic:                  overrideAuditTopic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		}
	}
	return &ExternalComplianceHandler{
		engine:       engine,
		refFetcher:   fetcher,
		redisClient:  redisClient,
		db:           db,
		kafkaWriter:  writer,
	}
}

type ExternalComplianceHandler struct {
	engine      *rules.RuleEngine
	refFetcher  ReferenceDataFetcher
	redisClient *redis.Client
	db          *sql.DB
	kafkaWriter *kafka.Writer
}

func (h *ExternalComplianceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/compliance/external", func(r chi.Router) {
		r.Post("/evaluate-external", h.HandleEvaluateExternal)
		r.Post("/evaluate-external-batch", h.HandleEvaluateExternalBatch)
	})
}

func (h *ExternalComplianceHandler) HandleEvaluateExternal(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req ExternalEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	idempKeyHeader := r.Header.Get("X-Idempotency-Key")
	cacheKey := idempKey(tenantIDStr, idempKeyHeader, "")
	if cacheKey != "" {
		if cached, ok := h.checkIdempotency(ctx, cacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)
			return
		}
	}

	mappings, _ := h.refFetcher.GetExternalMapping(ctx, tenantID, req.SystemIdentifier)

	hybridRecord := make(map[string]any)
	for extKey, val := range req.ProposedTrade {
		internalPath, ok := mappings[extKey]
		if !ok {
			internalPath = extKey
		}
		hybridRecord[internalPath] = val
	}

	isin, _ := hybridRecord["security.isin"].(string)
	if refState, err := h.refFetcher.GetPortfolioReferenceState(ctx, tenantID, req.PortfolioID, isin); err == nil {
		for k, v := range refState {
			hybridRecord[k] = v
		}
	}

	calculateProjectedMetrics(hybridRecord)

	chain, err := h.refFetcher.GetRuleChain(ctx, tenantID, req.RuleChainID)
	if err != nil {
		http.Error(w, "failed to fetch rule chain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	batchResult, trace := h.engine.EvaluateGroup(ctx, tenantIDStr, chain, hybridRecord)

	overrideReason := r.Header.Get("X-Override-Reason")
	canOverride := false
	highestSev := SeverityInfo

	for _, result := range batchResult.Results {
		sev, ov := mapSeverityToContract(result.Severity)
		if !ov {
			highestSev = sev
			break
		}
		if ov && sev == SeveritySoftWarn {
			canOverride = true
			highestSev = sev
		}
	}

	approved := batchResult.PassedAll
	if !approved && canOverride && overrideReason != "" {
		approved = true
		var allViolations []rules.RuleViolation
		for _, res := range batchResult.Results {
			allViolations = append(allViolations, res.Violations...)
		}
		if len(allViolations) > 0 {
			go func() {
				_ = h.recordCryptographicOverride(context.Background(), tenantID, "pm_user", req.RuleChainID, overrideReason, allViolations)
			}()
		}
	}

	response := ExternalEvaluateResponse{
		Approved:          approved,
		CanOverride:      canOverride,
		HighestSeverity:  highestSev,
		EvaluatedVM:     trace.UsedVM,
		ExecutionTimeNs: time.Since(start).Nanoseconds(),
		TraceRevision:   trace.Revision,
		Violations:      extractViolations(batchResult.Results),
		Timestamp:       start,
	}

	body, _ := json.Marshal(response)
	if cacheKey != "" {
		h.saveIdempotency(ctx, cacheKey, body)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ExternalComplianceHandler) HandleEvaluateExternalBatch(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req ExternalBatchEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Trades) == 0 {
		http.Error(w, "empty trades batch", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	idempKeyHeader := r.Header.Get("X-Idempotency-Key")
	batchCacheKey := idempKey(tenantIDStr, idempKeyHeader, req.BatchID)
	if batchCacheKey != "" {
		if cached, ok := h.checkIdempotency(ctx, batchCacheKey); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)
			return
		}
	}

	mappings, _ := h.refFetcher.GetExternalMapping(ctx, tenantID, req.SystemIdentifier)

	tradeResults := make([]ExternalTradeResult, len(req.Trades))
	allApproved := true
	highestSeverity := SeverityInfo

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range req.Trades {
		wg.Add(1)
		go func(idx int, tradeItem ExternalTradeItem) {
			defer wg.Done()

			hybridRecord := make(map[string]any)
			for extKey, val := range tradeItem.TradeData {
				internalPath, ok := mappings[extKey]
				if !ok {
					internalPath = extKey
				}
				hybridRecord[internalPath] = val
			}

			isin, _ := hybridRecord["security.isin"].(string)
			if refState, err := h.refFetcher.GetPortfolioReferenceState(ctx, tenantID, tradeItem.PortfolioID, isin); err == nil {
				for k, v := range refState {
					hybridRecord[k] = v
				}
			}

			calculateProjectedMetrics(hybridRecord)

			chain, err := h.refFetcher.GetRuleChain(ctx, tenantID, req.BatchID)
			itemApproved := false
			itemCanOverride := false
			var itemViolations []rules.RuleViolation

			if err == nil {
				batchResult, _ := h.engine.EvaluateGroup(ctx, tenantIDStr, chain, hybridRecord)
				itemApproved = batchResult.PassedAll
				itemViolations = extractViolations(batchResult.Results)

				for _, res := range batchResult.Results {
					sev, ov := mapSeverityToContract(res.Severity)
					if !ov {
						itemCanOverride = false
						break
					}
					if ov {
						itemCanOverride = true
					}
					_ = sev
				}

				if !itemApproved && itemCanOverride && req.OverrideReason != "" {
					itemApproved = true
					go func() {
						_ = h.recordCryptographicOverride(context.Background(), tenantID, "pm_user", tradeItem.ExternalOrderID, req.OverrideReason, itemViolations)
					}()
				}
			}

			mu.Lock()
			if !itemApproved {
				allApproved = false
			}
			tradeResults[idx] = ExternalTradeResult{
				ExternalOrderID: tradeItem.ExternalOrderID,
				PortfolioID:    tradeItem.PortfolioID,
				Approved:      itemApproved,
				CanOverride:  itemCanOverride,
				Violations:   itemViolations,
			}
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	for _, tr := range tradeResults {
		if !tr.CanOverride && !tr.Approved {
			highestSeverity = SeverityHardBlock
			break
		}
		if tr.CanOverride && !tr.Approved && highestSeverity != SeverityHardBlock {
			highestSeverity = SeveritySoftWarn
		}
	}

	response := ExternalBatchEvaluateResponse{
		BatchID:         req.BatchID,
		AllApproved:     allApproved,
		HighestSeverity: highestSeverity,
		ExecutionTimeNs: time.Since(start).Nanoseconds(),
		Results:        tradeResults,
		Timestamp:       start,
	}

	body, _ := json.Marshal(response)
	if batchCacheKey != "" {
		h.saveIdempotency(ctx, batchCacheKey, body)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func extractViolations(results []*rules.RuleResult) []rules.RuleViolation {
	var violations []rules.RuleViolation
	for _, r := range results {
		if !r.Passed && len(r.Violations) > 0 {
			violations = append(violations, r.Violations...)
		}
	}
	return violations
}

func calculateProjectedMetrics(m map[string]any) {
	proposedQty := toFloat64FromAny(m["order.quantity"])
	proposedPx := toFloat64FromAny(m["order.price"])
	currentVal := toFloat64FromAny(m["position.current_market_value"])
	totalAUM := toFloat64FromAny(m["portfolio.total_aum"])

	proposedTradeVal := proposedQty * proposedPx
	newTotalVal := currentVal + proposedTradeVal

	m["order.trade_value"] = proposedTradeVal
	m["position.projected_market_value"] = newTotalVal

	if totalAUM > 0 {
		m["position.projected_issuer_exposure_pct"] = newTotalVal / totalAUM
	}
}

func toFloat64FromAny(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
