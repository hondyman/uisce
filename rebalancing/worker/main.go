package main

import (
	"log"
	"os"
	"strings"

	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	kafka "github.com/segmentio/kafka-go"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Connect to Temporal using centralized helper (env-driven + retries)
	c, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Initialize Kafka writer for publishing trade/notification events
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	brokers := strings.Split(kafkaBrokers, ",")

	wWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}
	defer wWriter.Close()

	// Create worker
	w := worker.New(c, "rebalancing", worker.Options{})

	// Register workflow
	w.RegisterWorkflow(RebalanceAlphaWorkflow)
	w.RegisterWorkflow(SimulateRebalanceWorkflow)
	w.RegisterWorkflow(RiskAlphaWorkflow)
	w.RegisterWorkflow(AttributionAlphaWorkflow)
	w.RegisterWorkflow(RebalanceOrchestrator)

	// Register activities
	activities := &RebalanceActivities{
		kafkaWriter:   wWriter,
		hasuraURL:     os.Getenv("HASURA_URL"),
		xaiAPIKey:     os.Getenv("XAI_API_KEY"),
		finnhubAPIKey: os.Getenv("FINNHUB_API_KEY"),
	}
	w.RegisterActivity(activities.FetchPortfolio)
	w.RegisterActivity(activities.FetchRealTimePrices)
	w.RegisterActivity(activities.FetchHistoricalPrices)
	w.RegisterActivity(activities.AnalyzeDrift)
	w.RegisterActivity(activities.ABACCheck)
	w.RegisterActivity(activities.AIRebalance)
	w.RegisterActivity(activities.ValidatePlan)
	w.RegisterActivity(activities.ExecuteTrades)
	w.RegisterActivity(activities.UpdatePortfolioState)
	w.RegisterActivity(activities.InsertRebalancePlan)
	w.RegisterActivity(activities.NotifyStakeholders)
	w.RegisterActivity(activities.GenerateRebalanceSummary)
	w.RegisterActivity(activities.UpdatePlanWithSummary)
	w.RegisterActivity(activities.AIRiskScore)
	w.RegisterActivity(activities.ExecuteMitigation)
	w.RegisterActivity(activities.UpdateRiskStatus)
	w.RegisterActivity(activities.AIAttribution)
	w.RegisterActivity(activities.ExecuteAttribution)
	w.RegisterActivity(activities.UpdateAttributionStatus)

	// Register Risk Alpha activities (COMPREHENSIVE)
	w.RegisterActivity(activities.AIRiskScoreComprehensive)
	w.RegisterActivity(activities.AIMitigationStrategy)
	w.RegisterActivity(activities.ExecuteRiskMitigation)
	w.RegisterActivity(activities.CreateRiskEvent)
	w.RegisterActivity(activities.UpdateRiskEventMitigated)

	// Register Navigator (Cash Flow Forecasting) activities
	w.RegisterActivity(activities.CalibrateYaleModel)
	w.RegisterActivity(activities.GenerateCashFlowForecast)
	w.RegisterActivity(activities.RunMonteCarloSimulation)
	w.RegisterActivity(activities.ApplyBenchmarkRefinement)
	w.RegisterActivity(activities.ProjectDealJCurve)
	w.RegisterActivity(activities.ReconcileCapitalActivity)

	// Register Portfolio Rebalancing activities
	w.RegisterActivity("FetchPortfolioHoldingsActivity")
	w.RegisterActivity("GetAllocationModelActivity")
	w.RegisterActivity("CalculateDriftActivity")
	w.RegisterActivity("OptimizeTradesActivity")
	w.RegisterActivity("SaveProposedTradesActivity")
	w.RegisterActivity("PublishTradeEventActivity")
	w.RegisterActivity("LogRebalanceAuditActivity")

	// Start worker
	log.Println("Starting Temporal worker on queue: rebalancing")
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal(err)
	}
}
