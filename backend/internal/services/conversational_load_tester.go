package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/observability"
	"github.com/hondyman/semlayer/backend/internal/query"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ConversationalLoadTestConfig configures conversational load testing
type ConversationalLoadTestConfig struct {
	Duration         time.Duration
	Concurrency      int
	RequestRate      int // requests per second (0 = unlimited)
	TenantCount      int
	UserCount        int
	AssetCount       int
	WarmupDuration   time.Duration
	ProgressInterval time.Duration

	// Conversational-specific settings
	MaxTurnsPerConversation int           // 5-10 turns
	ThinkTimeBetweenTurns   time.Duration // 1-5 seconds
	AmbiguousRequestRatio   float64       // 20-30% intentionally ambiguous
	HotEntityRatio          float64       // % of requests for same metric/domain
	EnduranceMode           bool          // 8-24 hour runs
}

// ConversationalLoadTestResults contains results from conversational load testing
type ConversationalLoadTestResults struct {
	TotalRequests          int64
	SuccessfulRequests     int64
	FailedRequests         int64
	TotalConversations     int64
	TotalTurns             int64
	Duration               time.Duration
	RequestsPerSecond      float64
	ConversationsPerSecond float64
	TurnsPerSecond         float64
	AverageLatency         time.Duration
	P50Latency             time.Duration
	P95Latency             time.Duration
	P99Latency             time.Duration
	ErrorRate              float64

	// Conversational metrics
	AverageTurnsPerConversation float64
	GuardrailInterventionRate   float64
	CacheHitRate                float64
	SchemaCacheHitRate          float64
	PromptCacheHitRate          float64
	AmbiguousRequestRate        float64
	ClarificationSuccessRate    float64

	// Error taxonomy
	TimeoutErrors     int64
	PolicyFetchErrors int64
	PlannerErrors     int64
	GovernanceErrors  int64
	OtherErrors       int64
}

// ConversationalLoadTester performs load testing on the conversational layer
type ConversationalLoadTester struct {
	nlEngine      *query.NLQueryEngine
	config        ConversationalLoadTestConfig
	dtManager     *observability.DynatraceManager
	testScenarios []ConversationalTestScenario
}

// ConversationalTestScenario represents a test scenario for conversational testing
type ConversationalTestScenario struct {
	Name        string
	Description string
	Weight      float64 // Probability weight for selection
	Generator   func() *ConversationalTestCase
}

// ConversationalTestCase represents a single conversational test case
type ConversationalTestCase struct {
	TenantID       string
	UserID         string
	ConversationID string
	Turns          []ConversationalTurn
	ExpectedTurns  int
}

// ConversationalTurn represents a single turn in a conversation
type ConversationalTurn struct {
	UserMessage     string
	ExpectedIntent  string
	ShouldBeBlocked bool
	ThinkTime       time.Duration
}

// NewConversationalLoadTester creates a new conversational load tester
func NewConversationalLoadTester(nlEngine *query.NLQueryEngine, config ConversationalLoadTestConfig, dtManager *observability.DynatraceManager) *ConversationalLoadTester {
	tester := &ConversationalLoadTester{
		nlEngine:  nlEngine,
		config:    config,
		dtManager: dtManager,
	}

	tester.initializeTestScenarios()
	return tester
}

// initializeTestScenarios sets up the test scenarios
func (clt *ConversationalLoadTester) initializeTestScenarios() {
	clt.testScenarios = []ConversationalTestScenario{
		{
			Name:        "single_turn_simple",
			Description: "Single turn with simple, clear request",
			Weight:      0.4,
			Generator:   clt.generateSimpleSingleTurn,
		},
		{
			Name:        "single_turn_ambiguous",
			Description: "Single turn with intentionally ambiguous request",
			Weight:      0.2,
			Generator:   clt.generateAmbiguousSingleTurn,
		},
		{
			Name:        "multi_turn_clarification",
			Description: "Multi-turn conversation requiring clarification",
			Weight:      0.25,
			Generator:   clt.generateMultiTurnClarification,
		},
		{
			Name:        "hot_entity_campaign",
			Description: "Multiple users asking about same metric/domain",
			Weight:      0.15,
			Generator:   clt.generateHotEntityCampaign,
		},
	}
}

// RunConversationalLoadTest executes the conversational load test
func (clt *ConversationalLoadTester) RunConversationalLoadTest(ctx context.Context) (*ConversationalLoadTestResults, error) {
	logging.GetLogger().Sugar().Infof("Starting conversational load test: duration=%v, concurrency=%d, rate=%d req/s",
		clt.config.Duration, clt.config.Concurrency, clt.config.RequestRate)

	// Warmup phase
	if clt.config.WarmupDuration > 0 {
		logging.GetLogger().Sugar().Infof("Conversational warmup phase: %v", clt.config.WarmupDuration)
		clt.runConversationalWarmup(ctx)
	}

	// Test phase
	start := time.Now()
	results := clt.runConversationalTestPhase(ctx)
	results.Duration = time.Since(start)

	// Calculate final metrics
	results.RequestsPerSecond = float64(results.TotalRequests) / results.Duration.Seconds()
	results.ConversationsPerSecond = float64(results.TotalConversations) / results.Duration.Seconds()
	results.TurnsPerSecond = float64(results.TotalTurns) / results.Duration.Seconds()
	results.ErrorRate = float64(results.FailedRequests) / float64(results.TotalRequests) * 100

	if results.TotalConversations > 0 {
		results.AverageTurnsPerConversation = float64(results.TotalTurns) / float64(results.TotalConversations)
	}

	logging.GetLogger().Sugar().Infof("Conversational load test completed: %d conversations, %d turns, %.2f conv/s, %.2f turns/s",
		results.TotalConversations, results.TotalTurns, results.ConversationsPerSecond, results.TurnsPerSecond)

	return results, nil
}

// runConversationalWarmup performs the warmup phase
func (clt *ConversationalLoadTester) runConversationalWarmup(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, clt.config.WarmupDuration)
	defer cancel()

	var wg sync.WaitGroup
	conversations := make(chan *ConversationalTestCase, clt.config.Concurrency*10)

	// Start workers
	for i := 0; i < clt.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clt.conversationalWarmupWorker(ctx, conversations)
		}()
	}

	// Generate conversations
	go func() {
		defer close(conversations)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				testCase := clt.generateTestCase()
				select {
				case conversations <- testCase:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Wait()
	logging.GetLogger().Sugar().Info("Conversational warmup phase completed")
}

// runConversationalTestPhase performs the actual test phase
func (clt *ConversationalLoadTester) runConversationalTestPhase(ctx context.Context) *ConversationalLoadTestResults {
	ctx, cancel := context.WithTimeout(ctx, clt.config.Duration)
	defer cancel()

	results := &ConversationalLoadTestResults{}
	latencies := make([]time.Duration, 0, 100000)

	var wg sync.WaitGroup
	conversations := make(chan *ConversationalTestCase, clt.config.Concurrency*100)

	// Progress reporting
	progressTicker := time.NewTicker(clt.config.ProgressInterval)
	defer progressTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-progressTicker.C:
				current := atomic.LoadInt64(&results.TotalConversations)
				logging.GetLogger().Sugar().Infof("Conversational progress: %d conversations processed", current)
			}
		}
	}()

	// Rate limiter if specified
	var rateLimiter <-chan time.Time
	if clt.config.RequestRate > 0 {
		rateLimiter = time.Tick(time.Second / time.Duration(clt.config.RequestRate))
	}

	// Start workers
	for i := 0; i < clt.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			clt.conversationalTestWorker(ctx, conversations, results, latencies)
		}(i)
	}

	// Generate conversations
	go func() {
		defer close(conversations)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if rateLimiter != nil {
					select {
					case <-ctx.Done():
						return
					case <-rateLimiter:
					}
				}

				testCase := clt.generateTestCase()
				select {
				case conversations <- testCase:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Wait()

	// Calculate latency percentiles
	results.AverageLatency = clt.calculateAverageLatency(latencies)
	results.P50Latency = clt.calculatePercentileLatency(latencies, 50)
	results.P95Latency = clt.calculatePercentileLatency(latencies, 95)
	results.P99Latency = clt.calculatePercentileLatency(latencies, 99)

	return results
}

// conversationalWarmupWorker processes conversations during warmup
func (clt *ConversationalLoadTester) conversationalWarmupWorker(ctx context.Context, conversations <-chan *ConversationalTestCase) {
	for {
		select {
		case <-ctx.Done():
			return
		case testCase, ok := <-conversations:
			if !ok {
				return
			}
			// Just execute the conversation, don't track metrics
			clt.executeConversation(ctx, testCase)
		}
	}
}

// conversationalTestWorker processes conversations during test phase
func (clt *ConversationalLoadTester) conversationalTestWorker(ctx context.Context, conversations <-chan *ConversationalTestCase, results *ConversationalLoadTestResults, latencies []time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case testCase, ok := <-conversations:
			if !ok {
				return
			}

			atomic.AddInt64(&results.TotalConversations, 1)

			conversationStart := time.Now()
			turnsExecuted, conversationErr := clt.executeConversation(ctx, testCase)
			conversationDuration := time.Since(conversationStart)

			atomic.AddInt64(&results.TotalTurns, int64(turnsExecuted))
			atomic.AddInt64(&results.TotalRequests, int64(turnsExecuted)) // Each turn is a request

			if conversationErr != nil {
				atomic.AddInt64(&results.FailedRequests, 1)
				clt.categorizeError(results, conversationErr)
			} else {
				atomic.AddInt64(&results.SuccessfulRequests, int64(turnsExecuted))
			}

			// Record latency (with some sampling to avoid memory issues)
			if len(latencies) < cap(latencies) || rand.Float64() < 0.1 {
				latencies = append(latencies, conversationDuration)
			}
		}
	}
}

// executeConversation executes a full conversational test case
func (clt *ConversationalLoadTester) executeConversation(ctx context.Context, testCase *ConversationalTestCase) (int, error) {
	turnsExecuted := 0
	var conversationErr error

	var traceErr error
	if clt.dtManager != nil {
		traceErr = clt.dtManager.TraceFunc(ctx, "loadtest.conversation", func(traceCtx context.Context) error {
			span := trace.SpanFromContext(traceCtx)
			span.SetAttributes(
				attribute.String("semlayer.conversation.id", testCase.ConversationID),
				attribute.String("semlayer.user.id", testCase.UserID),
				attribute.String("semlayer.tenant.id", testCase.TenantID),
			)

			for i, turn := range testCase.Turns {
				// Simulate think time between turns
				if turn.ThinkTime > 0 {
					select {
					case <-traceCtx.Done():
						conversationErr = traceCtx.Err()
						return conversationErr
					case <-time.After(turn.ThinkTime):
					}
				}

				// Create NL query request
				req := &query.NLQueryRequest{
					Text:           turn.UserMessage,
					UserID:         testCase.UserID,
					TenantID:       testCase.TenantID,
					ConversationID: testCase.ConversationID,
					Datasource:     "test_datasource",
				}

				// Process the query with tracing
				turnErr := clt.dtManager.TraceFunc(traceCtx, fmt.Sprintf("loadtest.conversation.turn_%d", i+1), func(turnCtx context.Context) error {
					_, err := clt.nlEngine.ProcessNLQuery(turnCtx, req)
					if err != nil {
						trace.SpanFromContext(turnCtx).SetAttributes(attribute.String("error.message", err.Error()))
					}
					return err
				}, attribute.String("user.message", turn.UserMessage))

				if turnErr != nil {
					conversationErr = turnErr
					return conversationErr
				}

				turnsExecuted++
			}
			return nil
		})
	} else {
		// Execute without tracing
		for _, turn := range testCase.Turns {
			// Simulate think time between turns
			if turn.ThinkTime > 0 {
				select {
				case <-ctx.Done():
					conversationErr = ctx.Err()
					return turnsExecuted, conversationErr
				case <-time.After(turn.ThinkTime):
				}
			}

			// Create NL query request
			req := &query.NLQueryRequest{
				Text:           turn.UserMessage,
				UserID:         testCase.UserID,
				TenantID:       testCase.TenantID,
				ConversationID: testCase.ConversationID,
				Datasource:     "test_datasource",
			}

			// Process the query without tracing
			_, err := clt.nlEngine.ProcessNLQuery(ctx, req)
			if err != nil {
				conversationErr = err
				return turnsExecuted, conversationErr
			}

			turnsExecuted++
		}
	}

	if traceErr != nil {
		return turnsExecuted, traceErr
	}

	return turnsExecuted, conversationErr
}

// generateTestCase creates a random test case based on scenario weights
func (clt *ConversationalLoadTester) generateTestCase() *ConversationalTestCase {
	// Select scenario based on weights
	totalWeight := 0.0
	for _, scenario := range clt.testScenarios {
		totalWeight += scenario.Weight
	}

	r := rand.Float64() * totalWeight
	cumulativeWeight := 0.0

	for _, scenario := range clt.testScenarios {
		cumulativeWeight += scenario.Weight
		if r <= cumulativeWeight {
			return scenario.Generator()
		}
	}

	// Fallback to first scenario
	return clt.testScenarios[0].Generator()
}

// generateSimpleSingleTurn generates a simple single-turn test case
func (clt *ConversationalLoadTester) generateSimpleSingleTurn() *ConversationalTestCase {
	tenantID := fmt.Sprintf("tenant_%d", rand.Intn(clt.config.TenantCount)+1)
	userID := fmt.Sprintf("user_%d", rand.Intn(clt.config.UserCount)+1)
	conversationID := uuid.New().String()

	simpleQueries := []string{
		"Show me total sales for last month",
		"What is the revenue by region?",
		"Give me customer count by status",
		"Show profit margin trends",
		"List top 10 products by sales",
	}

	query := simpleQueries[rand.Intn(len(simpleQueries))]

	return &ConversationalTestCase{
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		Turns: []ConversationalTurn{
			{
				UserMessage:     query,
				ExpectedIntent:  "simple_query",
				ShouldBeBlocked: false,
				ThinkTime:       0,
			},
		},
		ExpectedTurns: 1,
	}
}

// generateAmbiguousSingleTurn generates an ambiguous single-turn test case
func (clt *ConversationalLoadTester) generateAmbiguousSingleTurn() *ConversationalTestCase {
	tenantID := fmt.Sprintf("tenant_%d", rand.Intn(clt.config.TenantCount)+1)
	userID := fmt.Sprintf("user_%d", rand.Intn(clt.config.UserCount)+1)
	conversationID := uuid.New().String()

	ambiguousQueries := []string{
		"Show me sales",       // Could be gross or net
		"What is the margin?", // Could be gross or net
		"Give me revenue",     // Could be total or recurring
		"Show profit",         // Could be various types
		"List by category",    // Could be product or customer category
	}

	query := ambiguousQueries[rand.Intn(len(ambiguousQueries))]

	return &ConversationalTestCase{
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		Turns: []ConversationalTurn{
			{
				UserMessage:     query,
				ExpectedIntent:  "ambiguous_query",
				ShouldBeBlocked: false,
				ThinkTime:       0,
			},
		},
		ExpectedTurns: 1,
	}
}

// generateMultiTurnClarification generates a multi-turn clarification test case
func (clt *ConversationalLoadTester) generateMultiTurnClarification() *ConversationalTestCase {
	tenantID := fmt.Sprintf("tenant_%d", rand.Intn(clt.config.TenantCount)+1)
	userID := fmt.Sprintf("user_%d", rand.Intn(clt.config.UserCount)+1)
	conversationID := uuid.New().String()

	maxTurns := clt.config.MaxTurnsPerConversation
	if maxTurns == 0 {
		maxTurns = 5
	}

	turns := []ConversationalTurn{
		{
			UserMessage:     "Show me sales",
			ExpectedIntent:  "ambiguous_sales",
			ShouldBeBlocked: false,
			ThinkTime:       clt.config.ThinkTimeBetweenTurns,
		},
		{
			UserMessage:     "gross sales",
			ExpectedIntent:  "clarification_response",
			ShouldBeBlocked: false,
			ThinkTime:       clt.config.ThinkTimeBetweenTurns,
		},
	}

	// Add more turns if needed
	for i := 2; i < maxTurns; i++ {
		turns = append(turns, ConversationalTurn{
			UserMessage:     "by region",
			ExpectedIntent:  "dimension_addition",
			ShouldBeBlocked: false,
			ThinkTime:       clt.config.ThinkTimeBetweenTurns,
		})
	}

	return &ConversationalTestCase{
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		Turns:          turns,
		ExpectedTurns:  maxTurns,
	}
}

// generateHotEntityCampaign generates a hot entity campaign test case
func (clt *ConversationalLoadTester) generateHotEntityCampaign() *ConversationalTestCase {
	// Use a fixed "hot" metric/domain to simulate cache hits
	tenantID := fmt.Sprintf("tenant_%d", rand.Intn(clt.config.TenantCount)+1)
	userID := fmt.Sprintf("user_%d", rand.Intn(clt.config.UserCount)+1)
	conversationID := uuid.New().String()

	hotQueries := []string{
		"Show me total revenue for Q1",
		"What is the revenue trend?",
		"Revenue by sales region",
		"Top revenue generating products",
	}

	query := hotQueries[rand.Intn(len(hotQueries))]

	return &ConversationalTestCase{
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		Turns: []ConversationalTurn{
			{
				UserMessage:     query,
				ExpectedIntent:  "hot_entity_query",
				ShouldBeBlocked: false,
				ThinkTime:       0,
			},
		},
		ExpectedTurns: 1,
	}
}

// categorizeError categorizes errors for taxonomy
func (clt *ConversationalLoadTester) categorizeError(results *ConversationalLoadTestResults, err error) {
	errStr := err.Error()

	switch {
	case containsString(errStr, "timeout"):
		atomic.AddInt64(&results.TimeoutErrors, 1)
	case containsString(errStr, "policy") || containsString(errStr, "governance"):
		atomic.AddInt64(&results.GovernanceErrors, 1)
	case containsString(errStr, "planner") || containsString(errStr, "schema"):
		atomic.AddInt64(&results.PlannerErrors, 1)
	default:
		atomic.AddInt64(&results.OtherErrors, 1)
	}
}

// Helper functions
func (clt *ConversationalLoadTester) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}

	return total / time.Duration(len(latencies))
}

func (clt *ConversationalLoadTester) calculatePercentileLatency(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple sort and pick (not the most efficient for large datasets)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Basic bubble sort for simplicity
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile / 100.0)
	return sorted[index]
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RunConversationalStandardLoadTest runs a standard conversational load test
func RunConversationalStandardLoadTest(nlEngine *query.NLQueryEngine) (*ConversationalLoadTestResults, error) {
	config := ConversationalLoadTestConfig{
		Duration:         10 * time.Minute,
		Concurrency:      20,
		RequestRate:      50, // 50 conversations per second
		TenantCount:      10,
		UserCount:        1000,
		AssetCount:       100,
		WarmupDuration:   2 * time.Minute,
		ProgressInterval: 30 * time.Second,

		// Conversational settings
		MaxTurnsPerConversation: 5,
		ThinkTimeBetweenTurns:   2 * time.Second,
		AmbiguousRequestRatio:   0.25,
		HotEntityRatio:          0.20,
		EnduranceMode:           false,
	}

	tester := NewConversationalLoadTester(nlEngine, config, nil)
	return tester.RunConversationalLoadTest(context.Background())
}
