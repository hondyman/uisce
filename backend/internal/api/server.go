package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hondyman/uisce/backend/internal/ebpf"
	"github.com/hondyman/uisce/backend/internal/fix"
	"github.com/hondyman/uisce/backend/internal/flight"
	"github.com/hondyman/uisce/backend/internal/governance"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/hondyman/uisce/backend/internal/shadow"
	"github.com/hondyman/uisce/backend/internal/streaming"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func StartServer() {
	log.Println("Initializing schema validator...")
	// TODO: Add schema validation initialization
	// if err := validate.Init(); err != nil {
	//     log.Fatalf("FATAL: Failed to initialize schema validator: %v", err)
	// }

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		panic("POSTGRES_DSN environment variable is required")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}

	// Initialize temporal client with retry logic
	temporalC, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("FATAL: Failed to create temporal client: %v", err)
	}
	defer temporalC.Close()

	// Initialize QoSManager
	qosManager := services.NewQoSManager(db)

	var complianceDeps *ComplianceDeps
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		ctx := context.Background()

		redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
		ruleEngine := rules.NewRuleEngine(nil)

		profiler := rules.NewLatencyProfiler()
		driftHealer := governance.NewSelfHealingService(db.DB)
		survivorshipEngine := mdm.NewSurvivorshipEngine()

		ruleEngine.SetProfiler(profiler)
		ruleEngine.SetDriftHealer(driftHealer)

		advisorWorker := rules.NewAdvisorWorker(ruleEngine, 5*time.Minute, 1000)
		advisorWorker.Start(ctx)

		fixAdapter := fix.NewAdapter(fix.RuleEngineToComplianceEvaluator(ruleEngine))
		fixServer, err := fix.NewServer(fixAdapter, "")
		if err != nil {
			log.Printf("[Warning] FIX server failed to create: %v", err)
		} else {
			go func() { _ = fixServer.Start(ctx) }()
		}

		rewarmerAdapter := streaming.NewTenantRewarmerAdapter(ruleEngine)
		ruleRepoAdapter := streaming.NewRuleRepoAdapter(ruleEngine)
		cdcConsumer := streaming.NewSchemaCDCConsumer(rewarmerAdapter, ruleRepoAdapter, 60)

		if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
			log.Printf("[CDC] Redpanda/Kafka CDC schema listener starting on brokers: %s", kafkaBrokers)
		}

		flightPort := 8090
		if envPort := os.Getenv("FLIGHT_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
				flightPort = p
			}
		}
		flightServer := flight.NewFlightServer(flightPort)
		go func() { _ = flightServer.Start(ctx) }()

		ebpfEnabled := os.Getenv("EBPF_ENABLED") == "true"
		ebpfIfName := os.Getenv("EBPF_IFNAME")
		if ebpfIfName == "" {
			ebpfIfName = "eth0"
		}
		ebpfService := ebpf.NewEBPFIngestionService(ruleEngine)
		ebpfCfg := &ebpf.EBPFConfig{
			IfName:         ebpfIfName,
			XDPAttachMode:  "skb",
			RingBufferSize: 1024 * 1024,
			Enabled:        ebpfEnabled,
		}
		if err := ebpfService.Start(ctx, ebpfCfg); err != nil {
			log.Printf("[Warning] eBPF ingestion service failed to start: %v", err)
		}

		shadowEngine := shadow.NewReplayEngine(db.DB, vm.NewSymbolDict(), vm.NewEnumDict())

		complianceDeps = &ComplianceDeps{
			RuleEngine:         ruleEngine,
			RedisClient:        redisClient,
			DB:                 db.DB,
			KafkaBrokers:       os.Getenv("KAFKA_BROKERS"),
			SurvivorshipEngine: survivorshipEngine,
			DriftHealer:        driftHealer,
			AdvisorWorker:      advisorWorker,
			FIXServer:          fixServer,
			CDCConsumer:        cdcConsumer,
			FlightServer:       flightServer,
			ShadowEngine:       shadowEngine,
		}
		log.Println("[Compliance] Pre-trade compliance engine initialized")
	}

	router := SetupRouter(db.DB, nil, nil, temporalC, qosManager, nil, nil, nil, nil, complianceDeps)
	log.Println("Server listening on :8080")
	http.ListenAndServe(":8080", router)
}
