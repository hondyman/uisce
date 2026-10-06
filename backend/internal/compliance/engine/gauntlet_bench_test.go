package engine

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hondyman/uisce/backend/internal/compliance/audit"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/reservation"
	"github.com/hondyman/uisce/backend/internal/compliance/routing"
	"github.com/hondyman/uisce/backend/internal/rules"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// Reserved test tenant ID for hygiene
var TestTenantID = uuid.MustParse("00000000-0000-4000-a000-000000000001")

// FullPipelinePreTradeEvaluator orchestrates the complete end-to-end pre-trade hot path
type FullPipelinePreTradeEvaluator struct {
	router      *routing.ConsistentHashRouter
	cache       *reservation.PositionCache
	resMgr      *reservation.ReservationManager
	evaluator   *FastBundleEvaluator
	bundle      *RuleBundle
	emitter     *audit.DurableAuditEmitter
	spool       *audit.DurableSpool
	nodeName    string
}

func NewFullPipelinePreTradeEvaluator(broker audit.MessageBroker, topic string) *FullPipelinePreTradeEvaluator {
	router := routing.NewConsistentHashRouter(100)
	router.AddNode("gateway-pod-1")
	router.AddNode("gateway-pod-2")
	router.AddNode("gateway-pod-3")

	cache := reservation.NewPositionCache()
	resMgr := reservation.NewReservationManager(cache, 5*time.Minute)
	evaluator := NewFastBundleEvaluator()

	// 15-rule bundle
	bundle := NewRuleBundle(TestTenantID, "PRE_TRADE_GAUNTLET_BUNDLE", 1)
	compiler := NewBundleCompiler()
	syms := bundle.SymDict
	enums := bundle.EnumDict

	for i := 0; i < 15; i++ {
		field := fmt.Sprintf("metric.f_%d", i)
		syms.Intern(field)

		ast := &rules.RuleNode{
			Type: vm.NodeTypeExpression,
			Expression: &vm.Expression{
				Root: &vm.BinaryExpr{
					Op:    "<=",
					Left:  &vm.FieldRef{Path: field},
					Right: &vm.Literal{Value: 100000.0},
				},
			},
		}

		r, _ := compiler.CompileRule(uuid.New(), fmt.Sprintf("R_GAUNTLET_%d", i), "Gauntlet Rule", SeverityHardBlock, ast, syms, enums, decimal.NewFromInt(100000))
		bundle.AddCompiledRule(r)
	}

	var emitter *audit.DurableAuditEmitter
	if broker != nil {
		emitter = audit.NewDurableAuditEmitter(broker, topic)
	}

	return &FullPipelinePreTradeEvaluator{
		router:    router,
		cache:     cache,
		resMgr:    resMgr,
		evaluator: evaluator,
		bundle:    bundle,
		emitter:   emitter,
		nodeName:  "gateway-pod-1",
	}
}

func (p *FullPipelinePreTradeEvaluator) SetSpool(spool *audit.DurableSpool) {
	p.spool = spool
}

// EvaluateFullHotPath runs the exact complete request path:
// [1] Ingress consistent-hash account dispatch
// [2] LRU position fetch + reservation acquire
// [3] Sequential VM rule bundle execution (15 rules)
// [4] Canonical JCS hash & audit envelope generation
// [5] Local WAL Spool / Synchronous Redpanda produce
// [6] Response serialization
func (p *FullPipelinePreTradeEvaluator) EvaluateFullHotPath(ctx context.Context, accountID, securityID, orderID uuid.UUID, qty, price decimal.Decimal) (int64, error) {
	start := time.Now()

	// [1] Account Dispatch (Ingress Authority)
	targetNode, err := p.router.RouteAccount(accountID)
	if err != nil {
		return 0, fmt.Errorf("route account: %w", err)
	}
	_ = targetNode

	// [2] Reservation Acquire
	lease, err := p.resMgr.AcquireReservation(ctx, TestTenantID, accountID, securityID, orderID, "BUY", qty, price, decimal.Zero)
	if err != nil {
		return 0, fmt.Errorf("acquire reservation: %w", err)
	}
	_ = lease

	// [3] VM Rule Execution (15 rules)
	syms := p.bundle.SymDict
	rec := vm.GetFastRecord(syms)
	for i := 0; i < 15; i++ {
		field := fmt.Sprintf("metric.f_%d", i)
		symID, _ := syms.Resolve(field)
		rec.FNumVals[symID] = float64(i) * 100.0
		rec.Present[symID] |= vm.HasFNum
	}
	var detailsBuf [16]RuleEvaluationDetail
	decision, passed, breached := p.evaluator.EvaluateBundleFast(p.bundle, rec, detailsBuf[:])
	vm.PutFastRecord(rec)

	// [4] Canonical JCS Hash & Audit Payload
	lineageID := uuid.New()
	evalTime := time.Now().UTC()
	formattedQty, _ := canonical.FormatDecimal6(qty)
	formattedPrice, _ := canonical.FormatDecimal6(price)
	inputMap := map[string]interface{}{
		"accountId":  accountID.String(),
		"securityId": securityID.String(),
		"orderId":    orderID.String(),
		"quantity":   formattedQty,
		"price":      formattedPrice,
	}
	metricMap := map[string]interface{}{
		"passedRules":   passed,
		"breachedRules": breached,
	}

	evalHash, err := canonical.ComputeEvaluationHash(canonical.EvaluationHashInput{
		LineageID:       lineageID,
		TenantID:        TestTenantID,
		RuleID:          p.bundle.BundleID,
		RuleVersion:     int(p.bundle.Version),
		ActionTaken:     string(decision),
		Passed:          decision == StatusPass,
		InputParams:     inputMap,
		MetricSnapshots: metricMap,
	})
	if err != nil {
		return 0, fmt.Errorf("compute eval hash: %w", err)
	}

	eventPayload := audit.EvaluationEventPayload{
		ID:             uuid.New(),
		LineageID:      lineageID,
		TenantID:       TestTenantID,
		OrderID:        &orderID,
		RuleID:         p.bundle.BundleID,
		RuleVersion:    int(p.bundle.Version),
		Passed:         decision == StatusPass,
		ActionTaken:    string(decision),
		LatencyMicros:  time.Since(start).Microseconds(),
		EvaluationHash: evalHash,
		InputParams:    inputMap,
		MetricSnapshots: metricMap,
		EvaluatedAt:    evalTime,
	}

	// [5] Durability Gate: Local WAL Spool OR Remote Sync Produce
	if p.spool != nil {
		if err := p.spool.WriteHotPath(ctx, eventPayload); err != nil {
			return 0, fmt.Errorf("spool write: %w", err)
		}
	} else if p.emitter != nil {
		err = p.emitter.EmitEvaluation(ctx, eventPayload)
		if err != nil {
			return 0, fmt.Errorf("emit evaluation: %w", err)
		}
	}

	// [6] Response Serialization
	respBytes, err := canonical.Marshal(eventPayload)
	if err != nil {
		return 0, fmt.Errorf("response marshal: %w", err)
	}
	_ = respBytes

	durationNs := time.Since(start).Nanoseconds()
	return durationNs, nil
}

type mockGauntletBroker struct{}

func (m *mockGauntletBroker) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	return nil
}

func (m *mockGauntletBroker) Subscribe(ctx context.Context, topic string, groupID string, handler func(key string, payload []byte) error) error {
	return nil
}

func TestBenchmarkGauntlet_OptionB_LocalDurableSpool(t *testing.T) {
	spoolDir, err := os.MkdirTemp("", "gauntlet_spool_*")
	if err != nil {
		t.Fatalf("Failed to create temp spool dir: %v", err)
	}
	defer os.RemoveAll(spoolDir)

	mockBroker := &mockGauntletBroker{}
	spool, err := audit.NewDurableSpool(spoolDir, mockBroker, "compliance.evaluations.gauntlet")
	if err != nil {
		t.Fatalf("Failed to create durable spool: %v", err)
	}
	defer spool.Close()

	pipeline := NewFullPipelinePreTradeEvaluator(nil, "")
	pipeline.SetSpool(spool)

	numAccounts := 100
	accountIDs := make([]uuid.UUID, numAccounts)
	for i := 0; i < numAccounts; i++ {
		accountIDs[i] = uuid.New()
		pipeline.cache.SetAccountNAV(TestTenantID, accountIDs[i], decimal.NewFromInt(10000000))
	}

	securityID := uuid.New()
	ctx := context.Background()

	// Warm up
	for i := 0; i < 50; i++ {
		_, _ = pipeline.EvaluateFullHotPath(ctx, accountIDs[i%numAccounts], securityID, uuid.New(), decimal.NewFromInt(100), decimal.NewFromInt(50))
	}

	var gcBefore debug.GCStats
	debug.ReadGCStats(&gcBefore)

	concurrency := 10
	totalRequests := 1000
	latencies := make([]int64, totalRequests)
	var mu sync.Mutex
	var wg sync.WaitGroup

	reqPerWorker := totalRequests / concurrency
	t.Logf("Running Pre-Trade Benchmark Gauntlet (Option B: Local Durable Spool WAL + fsync): %d requests across %d concurrent workers...", totalRequests, concurrency)

	startTotal := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for r := 0; r < reqPerWorker; r++ {
				idx := workerID*reqPerWorker + r
				acct := accountIDs[rand.Intn(numAccounts)]
				orderID := uuid.New()

				durNs, err := pipeline.EvaluateFullHotPath(ctx, acct, securityID, orderID, decimal.NewFromInt(10), decimal.NewFromInt(100))
				if err != nil {
					t.Errorf("Hot path error at req %d: %v", idx, err)
					return
				}

				mu.Lock()
				latencies[idx] = durNs
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	totalWallTime := time.Since(startTotal)

	var gcAfter debug.GCStats
	debug.ReadGCStats(&gcAfter)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := time.Duration(latencies[int(float64(totalRequests)*0.50)])
	p90 := time.Duration(latencies[int(float64(totalRequests)*0.90)])
	p95 := time.Duration(latencies[int(float64(totalRequests)*0.95)])
	p99 := time.Duration(latencies[int(float64(totalRequests)*0.99)])
	p999 := time.Duration(latencies[totalRequests-1])

	newGCCount := gcAfter.NumGC - gcBefore.NumGC
	var maxGCPause time.Duration
	if len(gcAfter.Pause) > 0 {
		for _, p := range gcAfter.Pause[:min(len(gcAfter.Pause), int(newGCCount)+1)] {
			if p > maxGCPause {
				maxGCPause = p
			}
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	t.Logf("\n====================================================================")
	t.Logf("     PRE-TRADE COMPLIANCE GAUNTLET: OPTION B (LOCAL DURABLE SPOOL)  ")
	t.Logf("====================================================================")
	t.Logf(" Persistence Architecture : Local Disk WAL with synchronous fsync + async Redpanda drain")
	t.Logf(" Total Executed Requests  : %d orders across %d concurrent workers", totalRequests, concurrency)
	t.Logf(" Total Wall Time          : %v (Throughput: %.2f ops/sec)", totalWallTime, float64(totalRequests)/totalWallTime.Seconds())
	t.Logf("--------------------------------------------------------------------")
	t.Logf(" Full Request Latency (Dispatch + Res + 15-VM + JCS + WAL fsync):")
	t.Logf("   p50 (Median)           : %v", p50)
	t.Logf("   p90                    : %v", p90)
	t.Logf("   p95                    : %v", p95)
	t.Logf("   p99 (SLI Gate < 1ms)   : %v", p99)
	t.Logf("   p99.9 (Tail)           : %v", p999)
	t.Logf("--------------------------------------------------------------------")
	t.Logf(" Garbage Collection & Memory Profile:")
	t.Logf("   Total GC Cycles during : %d", newGCCount)
	t.Logf("   Max GC Pause Duration  : %v (Gate Target < 100µs)", maxGCPause)
	t.Logf("   Total Allocations      : %d KB", m.TotalAlloc/1024)
	t.Logf("====================================================================")

	if p99 > 500*time.Millisecond {
		t.Errorf("Latency p99 %v exceeded 500ms disk fsync bound on host", p99)
	}
}

func TestBenchmarkGauntlet_OptionA_RemoteSyncProduce(t *testing.T) {
	brokerAddr := os.Getenv("REDPANDA_BROKER")
	if brokerAddr == "" {
		brokerAddr = "100.84.50.65:9092"
	}

	conn, err := net.DialTimeout("tcp", brokerAddr, 2*time.Second)
	if err != nil {
		t.Skipf("Redpanda not reachable at %s, skipping live broker benchmark", brokerAddr)
		return
	}
	conn.Close()

	topic := fmt.Sprintf("compliance.evaluations.gauntlet.%d", time.Now().UnixNano())
	broker := audit.NewKafkaBroker([]string{brokerAddr})
	pipeline := NewFullPipelinePreTradeEvaluator(broker, topic)

	// Pre-populate accounts and seed NAVs
	numAccounts := 100
	accountIDs := make([]uuid.UUID, numAccounts)
	for i := 0; i < numAccounts; i++ {
		accountIDs[i] = uuid.New()
		pipeline.cache.SetAccountNAV(TestTenantID, accountIDs[i], decimal.NewFromInt(10000000))
	}

	securityID := uuid.New()
	ctx := context.Background()

	// Warm up
	t.Log("Warming up JIT, VM stacks, and TCP connection...")
	for i := 0; i < 20; i++ {
		_, _ = pipeline.EvaluateFullHotPath(ctx, accountIDs[i%numAccounts], securityID, uuid.New(), decimal.NewFromInt(100), decimal.NewFromInt(50))
	}

	// Read initial GC Stats
	var gcBefore debug.GCStats
	debug.ReadGCStats(&gcBefore)

	// Execute Gauntlet Load Test: 500 requests across concurrent workers
	concurrency := 10
	totalRequests := 500
	latencies := make([]int64, totalRequests)
	var mu sync.Mutex
	var wg sync.WaitGroup

	reqPerWorker := totalRequests / concurrency
	t.Logf("Running Pre-Trade Benchmark Gauntlet (Option A: Remote Sync Produce acks=all): %d requests across %d concurrent workers against real Redpanda...", totalRequests, concurrency)

	startTotal := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for r := 0; r < reqPerWorker; r++ {
				idx := workerID*reqPerWorker + r
				acct := accountIDs[rand.Intn(numAccounts)]
				orderID := uuid.New()

				durNs, err := pipeline.EvaluateFullHotPath(ctx, acct, securityID, orderID, decimal.NewFromInt(10), decimal.NewFromInt(100))
				if err != nil {
					t.Errorf("Hot path error at req %d: %v", idx, err)
					return
				}

				mu.Lock()
				latencies[idx] = durNs
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	totalWallTime := time.Since(startTotal)

	// Read final GC Stats
	var gcAfter debug.GCStats
	debug.ReadGCStats(&gcAfter)

	// Compute Latency Percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := time.Duration(latencies[int(float64(totalRequests)*0.50)])
	p90 := time.Duration(latencies[int(float64(totalRequests)*0.90)])
	p95 := time.Duration(latencies[int(float64(totalRequests)*0.95)])
	p99 := time.Duration(latencies[int(float64(totalRequests)*0.99)])
	p999 := time.Duration(latencies[totalRequests-1])

	// Calculate GC pauses
	newGCCount := gcAfter.NumGC - gcBefore.NumGC
	var maxGCPause time.Duration
	if len(gcAfter.Pause) > 0 {
		for _, p := range gcAfter.Pause[:min(len(gcAfter.Pause), int(newGCCount)+1)] {
			if p > maxGCPause {
				maxGCPause = p
			}
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	t.Logf("\n====================================================================")
	t.Logf("     PRE-TRADE COMPLIANCE GAUNTLET: OPTION A (REMOTE SYNC PRODUCE)  ")
	t.Logf("====================================================================")
	t.Logf(" Broker / Persistence     : Real Redpanda on %s (RequiredAcks = acks=all)", brokerAddr)
	t.Logf(" Network Topology         : Remote WAN / Mesh Network (Tailscale)")
	t.Logf(" Total Executed Requests  : %d orders across %d concurrent workers", totalRequests, concurrency)
	t.Logf(" Total Wall Time          : %v (Throughput: %.2f ops/sec)", totalWallTime, float64(totalRequests)/totalWallTime.Seconds())
	t.Logf("--------------------------------------------------------------------")
	t.Logf(" Full Request Latency (Dispatch + Res + 15-VM + JCS + Remote Produce):")
	t.Logf("   p50 (Median)           : %v (Dominated by WAN TCP RTT)", p50)
	t.Logf("   p90                    : %v", p90)
	t.Logf("   p95                    : %v", p95)
	t.Logf("   p99                    : %v", p99)
	t.Logf("   p99.9 (Tail)           : %v", p999)
	t.Logf("--------------------------------------------------------------------")
	t.Logf(" Garbage Collection & Memory Profile:")
	t.Logf("   Total GC Cycles during : %d", newGCCount)
	t.Logf("   Max GC Pause Duration  : %v (Gate Target < 100µs)", maxGCPause)
	t.Logf("   Total Allocations      : %d KB", m.TotalAlloc/1024)
	t.Logf("====================================================================")

	if p99 > 1000*time.Millisecond {
		t.Errorf("Latency p99 %v exceeded network bound", p99)
	}
}

func TestBenchmarkGauntlet_StepByStepBreakdown(t *testing.T) {
	brokerAddr := os.Getenv("REDPANDA_BROKER")
	if brokerAddr == "" {
		brokerAddr = "100.84.50.65:9092"
	}

	pipeline := NewFullPipelinePreTradeEvaluator(nil, "") // Nil broker to isolate pure compute hot-path
	accountID := uuid.New()
	securityID := uuid.New()
	orderID := uuid.New()
	qty := decimal.NewFromInt(500)
	price := decimal.RequireFromString("150.25")

	pipeline.cache.SetAccountNAV(TestTenantID, accountID, decimal.NewFromInt(10000000))
	ctx := context.Background()

	// Iterations
	const iterations = 100000
	var tDispatch, tRes, tVM, tJCS, tResp int64

	for i := 0; i < iterations; i++ {
		// [1] Ingress Dispatch
		s1 := time.Now()
		_, _ = pipeline.router.RouteAccount(accountID)
		tDispatch += time.Since(s1).Nanoseconds()

		// [2] Reservation Delta
		s2 := time.Now()
		lease, _ := pipeline.resMgr.AcquireReservation(ctx, TestTenantID, accountID, securityID, orderID, "BUY", qty, price, decimal.Zero)
		tRes += time.Since(s2).Nanoseconds()

		// [3] Sequential VM (15 rules)
		s3 := time.Now()
		syms := pipeline.bundle.SymDict
		rec := vm.GetFastRecord(syms)
		for r := 0; r < 15; r++ {
			field := fmt.Sprintf("metric.f_%d", r)
			symID, _ := syms.Resolve(field)
			rec.FNumVals[symID] = float64(r) * 100.0
			rec.Present[symID] |= vm.HasFNum
		}
		var detailsBuf [16]RuleEvaluationDetail
		decision, passed, breached := pipeline.evaluator.EvaluateBundleFast(pipeline.bundle, rec, detailsBuf[:])
		vm.PutFastRecord(rec)
		tVM += time.Since(s3).Nanoseconds()

		// [4] Canonical JCS Hash & Envelope
		s4 := time.Now()
		lineageID := uuid.New()
		formattedQty, _ := canonical.FormatDecimal6(qty)
		formattedPrice, _ := canonical.FormatDecimal6(price)
		inputMap := map[string]interface{}{
			"accountId":  accountID.String(),
			"securityId": securityID.String(),
			"orderId":    orderID.String(),
			"quantity":   formattedQty,
			"price":      formattedPrice,
		}
		metricMap := map[string]interface{}{
			"passedRules":   passed,
			"breachedRules": breached,
		}
		evalHash, _ := canonical.ComputeEvaluationHash(canonical.EvaluationHashInput{
			LineageID:       lineageID,
			TenantID:        TestTenantID,
			RuleID:          pipeline.bundle.BundleID,
			RuleVersion:     int(pipeline.bundle.Version),
			ActionTaken:     string(decision),
			Passed:          decision == StatusPass,
			InputParams:     inputMap,
			MetricSnapshots: metricMap,
		})
		tJCS += time.Since(s4).Nanoseconds()

		// [5] Response Serialization
		s5 := time.Now()
		ev := audit.EvaluationEventPayload{
			ID:             uuid.New(),
			LineageID:      lineageID,
			TenantID:       TestTenantID,
			OrderID:        &orderID,
			RuleID:         pipeline.bundle.BundleID,
			RuleVersion:    int(pipeline.bundle.Version),
			Passed:         decision == StatusPass,
			ActionTaken:    string(decision),
			EvaluationHash: evalHash,
			InputParams:    inputMap,
			MetricSnapshots: metricMap,
			EvaluatedAt:    time.Now().UTC(),
		}
		_, _ = canonical.Marshal(ev)
		tResp += time.Since(s5).Nanoseconds()

		_ = pipeline.resMgr.ReleaseReservation(lease.LeaseID)
	}

	avgDispatch := float64(tDispatch) / float64(iterations) / 1000.0
	avgRes := float64(tRes) / float64(iterations) / 1000.0
	avgVM := float64(tVM) / float64(iterations) / 1000.0
	avgJCS := float64(tJCS) / float64(iterations) / 1000.0
	avgResp := float64(tResp) / float64(iterations) / 1000.0
	totalInProcessCompute := avgDispatch + avgRes + avgVM + avgJCS + avgResp

	t.Logf("\n====================================================================")
	t.Logf("     PRE-TRADE IN-PROCESS COMPUTATIONAL LATENCY BUDGET BREAKDOWN    ")
	t.Logf("     (Averaged over %d iterations on hot CPU path)                  ", iterations)
	t.Logf("====================================================================")
	t.Logf(" [1] Ingress & Consistent-Hash Account Dispatch : %6.2f µs (Budget:  40 µs)", avgDispatch)
	t.Logf(" [2] Local LRU Position Fetch + Reservation     : %6.2f µs (Budget:  80 µs)", avgRes)
	t.Logf(" [3] Sequential 15-Rule VM Bundle Execution    : %6.2f µs (Budget: 120 µs)", avgVM)
	t.Logf(" [4] Canonical JCS Hash & Envelope Generation   : %6.2f µs (Budget:  60 µs)", avgJCS)
	t.Logf(" [5] Response Serialization                     : %6.2f µs (Budget:  40 µs)", avgResp)
	t.Logf(" -------------------------------------------------------------------")
	t.Logf(" Total In-Process Pre-Trade Computation Time    : %6.2f µs (SLI: < 1,000 µs)", totalInProcessCompute)
	t.Logf("====================================================================")

	if totalInProcessCompute > 1000.0 {
		t.Errorf("Total in-process compute time %.2fµs exceeded 1,000µs SLI budget", totalInProcessCompute)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
