package drift

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// PriceUpdate represents an incoming price feed event
type PriceUpdate struct {
	Ticker    string    `json:"ticker"`
	Price     float64   `json:"price"`
	Change    float64   `json:"change"`
	ChangePct float64   `json:"change_pct"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// PortfolioPosition represents a position in a portfolio
type PortfolioPosition struct {
	PortfolioID string  `db:"portfolio_id" json:"portfolio_id"`
	TenantID    string  `db:"tenant_id" json:"tenant_id"`
	Ticker      string  `db:"ticker" json:"ticker"`
	Quantity    float64 `db:"quantity" json:"quantity"`
	MarketValue float64 `db:"market_value" json:"market_value"`
	CurrentWgt  float64 `db:"current_weight" json:"current_weight"`
	TargetWgt   float64 `db:"target_weight" json:"target_weight"`
	DriftPct    float64 `json:"drift_pct"`
	LastPrice   float64 `db:"last_price" json:"last_price"`
}

// DriftAlert represents a drift threshold breach
type DriftAlert struct {
	ID            uuid.UUID           `json:"id"`
	TenantID      string              `json:"tenant_id"`
	PortfolioID   string              `json:"portfolio_id"`
	TotalDrift    float64             `json:"total_drift_pct"`
	MaxAssetDrift float64             `json:"max_asset_drift_pct"`
	TopDrifters   []PortfolioPosition `json:"top_drifters"`
	DetectedAt    time.Time           `json:"detected_at"`
	Severity      string              `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
}

// DriftThreshold represents tenant-specific drift thresholds
type DriftThreshold struct {
	TenantID     string  `db:"tenant_id"`
	AlertLevel   string  `db:"alert_level"`
	ThresholdPct float64 `db:"threshold_pct"`
}

// DriftConsumer consumes price updates and detects portfolio drift in real-time
type DriftConsumer struct {
	db             *sqlx.DB
	temporalClient client.Client
	logger         *zap.Logger
	reader         *kafka.Reader
	writer         *kafka.Writer // For testing/simulation publishing

	// Cache for portfolio positions (updated on price changes)
	positionCache map[string][]PortfolioPosition
	cacheMu       sync.RWMutex

	// Thresholds by tenant
	thresholds   map[string][]DriftThreshold
	thresholdsMu sync.RWMutex

	// Configuration
	brokers string
	topic   string

	// Control
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewDriftConsumer creates a new drift detection consumer
func NewDriftConsumer(db *sqlx.DB, temporalClient client.Client, brokers string, logger *zap.Logger) (*DriftConsumer, error) {
	if brokers == "" {
		brokers = "redpanda:9092"
	}
	brokerList := strings.Split(brokers, ",")
	topic := "market.prices"

	// Create reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		GroupID:  "drift-detection-service",
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	// Create writer (optional, for sim)
	w := &kafka.Writer{
		Addr:  kafka.TCP(brokerList...),
		Topic: topic,
	}

	consumer := &DriftConsumer{
		db:             db,
		temporalClient: temporalClient,
		logger:         logger,
		reader:         r,
		writer:         w,
		brokers:        brokers,
		topic:          topic,
		positionCache:  make(map[string][]PortfolioPosition),
		thresholds:     make(map[string][]DriftThreshold),
		stopCh:         make(chan struct{}),
	}

	// Load initial thresholds
	if err := consumer.loadThresholds(context.Background()); err != nil {
		consumer.logger.Warn("Failed to load initial thresholds", zap.Error(err))
	}

	return consumer, nil
}

// Start begins consuming price updates
func (c *DriftConsumer) Start(ctx context.Context) error {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.consumeLoop(ctx)
	}()

	// Periodic threshold refresh
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-ticker.C:
				if err := c.loadThresholds(ctx); err != nil {
					c.logger.Warn("Failed to refresh thresholds", zap.Error(err))
				}
			}
		}
	}()

	c.logger.Info("Drift consumer started",
		zap.String("topic", c.topic),
		zap.String("brokers", c.brokers))

	return nil
}

// consumeLoop processes incoming price updates
func (c *DriftConsumer) consumeLoop(ctx context.Context) {
	defer c.reader.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("Failed to fetch message", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}

			if err := c.handlePriceUpdate(ctx, msg); err != nil {
				c.logger.Error("Failed to handle price update",
					zap.Error(err),
					zap.String("key", string(msg.Key)))
				// Commit anyway to skip bad message?
				c.reader.CommitMessages(ctx, msg)
			} else {
				c.reader.CommitMessages(ctx, msg)
			}
		}
	}
}

// handlePriceUpdate processes a single price update
func (c *DriftConsumer) handlePriceUpdate(ctx context.Context, msg kafka.Message) error {
	var update PriceUpdate
	if err := json.Unmarshal(msg.Value, &update); err != nil {
		return fmt.Errorf("failed to unmarshal price update: %w", err)
	}

	c.logger.Debug("Received price update",
		zap.String("ticker", update.Ticker),
		zap.Float64("price", update.Price),
		zap.Float64("change_pct", update.ChangePct))

	// Find all portfolios holding this ticker
	portfolios, err := c.findPortfoliosWithTicker(ctx, update.Ticker)
	if err != nil {
		return fmt.Errorf("failed to find portfolios: %w", err)
	}

	if len(portfolios) == 0 {
		return nil // No portfolios hold this ticker
	}

	// Update prices and recalculate drift for each affected portfolio
	for _, portfolio := range portfolios {
		if err := c.updateAndCheckDrift(ctx, portfolio.TenantID, portfolio.PortfolioID, update); err != nil {
			c.logger.Error("Failed to check drift for portfolio",
				zap.String("portfolio_id", portfolio.PortfolioID),
				zap.Error(err))
		}
	}

	return nil
}

// findPortfoliosWithTicker finds all portfolios that hold a given ticker
func (c *DriftConsumer) findPortfoliosWithTicker(ctx context.Context, ticker string) ([]PortfolioPosition, error) {
	query := `
		SELECT DISTINCT p.tenant_id, p.portfolio_id
		FROM positions p
		WHERE p.ticker = $1
		  AND p.quantity > 0
	`

	var positions []PortfolioPosition
	if err := c.db.SelectContext(ctx, &positions, query, ticker); err != nil {
		return nil, err
	}

	return positions, nil
}

// updateAndCheckDrift updates position values and checks for drift breaches
func (c *DriftConsumer) updateAndCheckDrift(ctx context.Context, tenantID, portfolioID string, update PriceUpdate) error {
	// Load full portfolio positions
	positions, err := c.loadPortfolioPositions(ctx, tenantID, portfolioID)
	if err != nil {
		return err
	}

	if len(positions) == 0 {
		return nil
	}

	// Update the price for the affected ticker
	var totalValue float64
	for i := range positions {
		if positions[i].Ticker == update.Ticker {
			positions[i].LastPrice = update.Price
			positions[i].MarketValue = positions[i].Quantity * update.Price
		}
		totalValue += positions[i].MarketValue
	}

	// Recalculate weights and drift
	var totalDrift float64
	var maxDrift float64
	for i := range positions {
		if totalValue > 0 {
			positions[i].CurrentWgt = positions[i].MarketValue / totalValue
		}
		positions[i].DriftPct = math.Abs(positions[i].CurrentWgt-positions[i].TargetWgt) * 100

		totalDrift += positions[i].DriftPct
		if positions[i].DriftPct > maxDrift {
			maxDrift = positions[i].DriftPct
		}
	}

	// Check against thresholds
	alert := c.checkThresholds(tenantID, portfolioID, totalDrift, maxDrift, positions)
	if alert != nil {
		// Record drift snapshot
		if err := c.recordDriftSnapshot(ctx, alert); err != nil {
			c.logger.Error("Failed to record drift snapshot", zap.Error(err))
		}

		// Trigger rebalance workflow if severity is HIGH or CRITICAL
		if alert.Severity == "HIGH" || alert.Severity == "CRITICAL" {
			if err := c.triggerRebalanceWorkflow(ctx, alert); err != nil {
				c.logger.Error("Failed to trigger rebalance workflow", zap.Error(err))
			}
		}
	}

	return nil
}

// loadPortfolioPositions loads all positions for a portfolio
func (c *DriftConsumer) loadPortfolioPositions(ctx context.Context, tenantID, portfolioID string) ([]PortfolioPosition, error) {
	query := `
		SELECT 
			p.tenant_id,
			p.portfolio_id,
			p.ticker,
			p.quantity,
			p.quantity * COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) as market_value,
			COALESCE(t.target_weight, 0) as target_weight,
			COALESCE(pr.last_price, p.cost_basis / NULLIF(p.quantity, 0)) as last_price
		FROM positions p
		LEFT JOIN prices pr ON p.ticker = pr.ticker
		LEFT JOIN portfolio_targets t ON p.portfolio_id = t.portfolio_id AND p.ticker = t.ticker
		WHERE p.tenant_id = $1 AND p.portfolio_id = $2 AND p.quantity > 0
	`

	var positions []PortfolioPosition
	if err := c.db.SelectContext(ctx, &positions, query, tenantID, portfolioID); err != nil {
		return nil, err
	}

	return positions, nil
}

// loadThresholds loads drift thresholds from the database
func (c *DriftConsumer) loadThresholds(ctx context.Context) error {
	query := `
		SELECT tenant_id, alert_level, threshold_pct
		FROM drift_thresholds
		WHERE active = true
	`

	var thresholds []DriftThreshold
	if err := c.db.SelectContext(ctx, &thresholds, query); err != nil {
		return err
	}

	c.thresholdsMu.Lock()
	defer c.thresholdsMu.Unlock()

	c.thresholds = make(map[string][]DriftThreshold)
	for _, t := range thresholds {
		c.thresholds[t.TenantID] = append(c.thresholds[t.TenantID], t)
	}

	return nil
}

// checkThresholds checks if drift exceeds configured thresholds
func (c *DriftConsumer) checkThresholds(tenantID, portfolioID string, totalDrift, maxDrift float64, positions []PortfolioPosition) *DriftAlert {
	c.thresholdsMu.RLock()
	thresholds, ok := c.thresholds[tenantID]
	c.thresholdsMu.RUnlock()

	if !ok {
		// Use default thresholds
		thresholds = []DriftThreshold{
			{AlertLevel: "LOW", ThresholdPct: 3.0},
			{AlertLevel: "MEDIUM", ThresholdPct: 5.0},
			{AlertLevel: "HIGH", ThresholdPct: 7.0},
			{AlertLevel: "CRITICAL", ThresholdPct: 10.0},
		}
	}

	var severity string
	for _, t := range thresholds {
		if totalDrift >= t.ThresholdPct || maxDrift >= t.ThresholdPct {
			severity = t.AlertLevel
		}
	}

	if severity == "" {
		return nil // No threshold breached
	}

	// Sort positions by drift and take top 5
	topDrifters := make([]PortfolioPosition, 0, 5)
	for _, p := range positions {
		if len(topDrifters) < 5 {
			topDrifters = append(topDrifters, p)
		} else {
			// Find minimum in topDrifters and replace if current is larger
			minIdx := 0
			for i, tp := range topDrifters {
				if tp.DriftPct < topDrifters[minIdx].DriftPct {
					minIdx = i
				}
			}
			if p.DriftPct > topDrifters[minIdx].DriftPct {
				topDrifters[minIdx] = p
			}
		}
	}

	return &DriftAlert{
		ID:            uuid.New(),
		TenantID:      tenantID,
		PortfolioID:   portfolioID,
		TotalDrift:    totalDrift,
		MaxAssetDrift: maxDrift,
		TopDrifters:   topDrifters,
		DetectedAt:    time.Now(),
		Severity:      severity,
	}
}

// recordDriftSnapshot saves the drift snapshot to the database
func (c *DriftConsumer) recordDriftSnapshot(ctx context.Context, alert *DriftAlert) error {
	topDriftersJSON, _ := json.Marshal(alert.TopDrifters)

	query := `
		INSERT INTO drift_snapshots (
			id, tenant_id, portfolio_id, snapshot_at, total_drift_pct,
			max_asset_drift_pct, top_overweight, triggered_rebalance
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := c.db.ExecContext(ctx, query,
		alert.ID, alert.TenantID, alert.PortfolioID, alert.DetectedAt,
		alert.TotalDrift, alert.MaxAssetDrift, topDriftersJSON,
		alert.Severity == "HIGH" || alert.Severity == "CRITICAL")

	return err
}

// triggerRebalanceWorkflow starts a Temporal rebalance workflow
func (c *DriftConsumer) triggerRebalanceWorkflow(ctx context.Context, alert *DriftAlert) error {
	if c.temporalClient == nil {
		c.logger.Warn("Temporal client not available, skipping workflow trigger")
		return nil
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("rebalance-%s-%s", alert.PortfolioID, alert.ID.String()[:8]),
		TaskQueue: "rebalancer",
	}

	input := map[string]interface{}{
		"tenant_id":       alert.TenantID,
		"portfolio_id":    alert.PortfolioID,
		"trigger_type":    "DRIFT",
		"drift_alert_id":  alert.ID.String(),
		"total_drift":     alert.TotalDrift,
		"max_asset_drift": alert.MaxAssetDrift,
		"severity":        alert.Severity,
	}

	we, err := c.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "RebalanceWorkflow", input)
	if err != nil {
		return fmt.Errorf("failed to start rebalance workflow: %w", err)
	}

	c.logger.Info("Triggered rebalance workflow",
		zap.String("workflow_id", we.GetID()),
		zap.String("portfolio_id", alert.PortfolioID),
		zap.String("severity", alert.Severity))

	return nil
}

// Stop stops the consumer
func (c *DriftConsumer) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// Close closes the connection
func (c *DriftConsumer) Close() error {
	c.Stop()
	if c.reader != nil {
		c.reader.Close()
	}
	if c.writer != nil {
		c.writer.Close()
	}
	return nil
}

// PublishPriceUpdate publishes a price update to Kafka (for testing/simulation)
func (c *DriftConsumer) PublishPriceUpdate(ctx context.Context, update PriceUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	// Key by Ticker for partition locality
	key := []byte(update.Ticker)

	msg := kafka.Message{
		Key:   key,
		Value: body,
		Time:  time.Now(),
	}

	return c.writer.WriteMessages(ctx, msg)
}
