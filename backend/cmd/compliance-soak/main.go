package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"

	"github.com/hondyman/uisce/backend/internal/compliance/engine"
	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
)

type CycleMetric struct {
	Timestamp  string  `json:"ts"`
	Cycle      int     `json:"cycle"`
	HeapAlloc  uint64  `json:"heap_alloc"`
	TotalAlloc uint64  `json:"total_alloc"`
	NumGC      uint32  `json:"num_gc"`
	GCPauseNS  uint64  `json:"gcpause_ns"`
	LatencyMS  float64 `json:"latency_ms"`
}

func main() {
	cyclesFlag := flag.Int("cycles", 50, "Number of continuous evaluation soak cycles to run (used if duration is 0)")
	durationFlag := flag.Duration("duration", 0, "Time-bounded soak duration (e.g. 8h, 10m, 30s). Overrides cycles if > 0")
	rateLimitFlag := flag.Duration("ratelimit", 0, "Rate-limiting sleep between cycles (e.g. 8s, 100ms)")
	logCSVFlag := flag.String("csv", "", "Path to write CSV time-series log")
	logJSONLFlag := flag.String("jsonl", "", "Path to write JSONL time-series log")
	flag.Parse()

	fmt.Println("======================================================================")
	fmt.Println("PRODUCTION CONTINUOUS COMPLIANCE SOAK TEST RUNNER (SPRINT C)")
	if *durationFlag > 0 {
		fmt.Printf("Mode: Time-Bounded (%v) | RateLimit: %v | Concurrent Accounts: 5 | Rules: 94\n", *durationFlag, *rateLimitFlag)
	} else {
		fmt.Printf("Mode: Fixed Cycles (%d) | RateLimit: %v | Concurrent Accounts: 5 | Rules: 94\n", *cyclesFlag, *rateLimitFlag)
	}
	fmt.Println("======================================================================")

	sandboxDB := testutil.GetEphemeralTestDB(nil)
	if sandboxDB == nil {
		log.Fatal("Failed to initialize ephemeral soak sandbox DB")
	}
	defer sandboxDB.Close()

	sandboxDB.SetMaxOpenConns(10)
	sandboxDB.SetMaxIdleConns(5)
	sandboxDB.SetConnMaxLifetime(15 * time.Minute)

	ctx := context.Background()
	if *durationFlag > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *durationFlag+10*time.Minute)
		defer cancel()
	}

	tenantID := uuid.MustParse("00000000-0000-4000-a000-000000000003") // Dedicated Production Real Data Tenant

	_, err := sandboxDB.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy, is_active, status, plan)
		VALUES ($1, 'alpha_production', 'Production Alpha Data Tenant', false, true, 'active', 'enterprise')
		ON CONFLICT (id) DO NOTHING;
	`, tenantID)
	if err != nil {
		log.Fatalf("Failed to ensure soak tenant: %v", err)
	}

	evaluator := engine.NewPostTradeEvaluator(sandboxDB)

	var memStatsBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStatsBefore)

	startTime := time.Now()
	var totalEvaluations int64 = 0
	var totalBreaches int64 = 0
	var totalWarnings int64 = 0
	var latencies []time.Duration
	var cycleMetrics []CycleMetric
	var gcPauses []uint64

	accounts := []uuid.UUID{
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(),
	}

	securities := []struct {
		ID       string
		Symbol   string
		IssuerID string
		Sector   string
		Rating   string
	}{
		{"ff74616b-910a-45ad-8ad3-6c8df6391ca6", "AAPL", "ISS-AAPL", "Information Technology", "AAA"},
		{"e3eed040-b506-41db-821b-1a203111f007", "MSFT", "ISS-MSFT", "Information Technology", "AAA"},
		{"1c2a330a-c374-4336-a22d-1790022f54d1", "GOOGL", "ISS-GOOGL", "Communication Services", "AA+"},
		{"702c64da-6ff8-47b8-9472-d4c06a1349fc", "AMZN", "ISS-AMZN", "Consumer Discretionary", "AA"},
		{"6fcc58a3-72bc-4304-8db9-f23b68067eaa", "JPM", "ISS-JPM", "Financials", "A+"},
	}

	// CSV file setup
	var csvFile *os.File
	var csvWriter *csv.Writer
	if *logCSVFlag != "" {
		if err := os.MkdirAll(filepath.Dir(*logCSVFlag), 0755); err == nil {
			if f, err := os.Create(*logCSVFlag); err == nil {
				csvFile = f
				defer csvFile.Close()
				csvWriter = csv.NewWriter(f)
				defer csvWriter.Flush()
				_ = csvWriter.Write([]string{"ts", "cycle", "heap_alloc", "total_alloc", "num_gc", "gcpause_ns", "latency_ms"})
			}
		}
	}

	// JSONL file setup
	var jsonlFile *os.File
	if *logJSONLFlag != "" {
		if err := os.MkdirAll(filepath.Dir(*logJSONLFlag), 0755); err == nil {
			if f, err := os.Create(*logJSONLFlag); err == nil {
				jsonlFile = f
				defer jsonlFile.Close()
			}
		}
	}

	fmt.Println("\n[Phase 1] Executing Continuous Evaluation Stream Soak Cycles...")

	cycle := 0
	for {
		cycle++
		if *durationFlag > 0 {
			if time.Since(startTime) >= *durationFlag {
				break
			}
		} else {
			if cycle > *cyclesFlag {
				break
			}
		}

		cycleStart := time.Now()
		for _, acctID := range accounts {
			nav := decimal.RequireFromString("100000000.000000")
			cashRatio := 0.05 + rand.Float64()*0.15 // 5% - 20% cash
			cash := nav.Mul(decimal.NewFromFloat(cashRatio))

			var positions []engine.PortfolioPosition
			remainingMV := nav.Sub(cash)

			for _, s := range securities {
				weight := 0.15 + (rand.Float64() * 0.05) // ~15-20% each
				posMV := remainingMV.Mul(decimal.NewFromFloat(weight))
				positions = append(positions, engine.PortfolioPosition{
					SecurityID:              s.ID,
					Symbol:                  s.Symbol,
					IssuerID:                s.IssuerID,
					IssuerName:              s.Symbol + " Inc.",
					CountryOfRisk:           "US",
					Sector:                  s.Sector,
					CreditRating:            s.Rating,
					MarketValue:             posMV,
					SharesHeld:              posMV.Div(decimal.RequireFromString("150.0")),
					SharesOutstanding:       decimal.RequireFromString("100000000.0"),
					VotingSharesHeld:        posMV.Div(decimal.RequireFromString("150.0")),
					VotingSharesOutstanding: decimal.RequireFromString("100000000.0"),
					AssetClass:              "equity",
					IsQIBEligible:           true,
				})
			}

			isPass := true
			state := engine.PortfolioState{
				TenantID:        tenantID,
				AccountID:       acctID,
				AsOfDate:        time.Now().UTC().AddDate(0, 0, -cycle),
				NAV:             nav,
				GrossExposure:   nav,
				NetExposure:     nav,
				CashBalance:     cash,
				IsPassiveIntent: &isPass,
				Positions:       positions,
				PriorSnapshotMetrics: map[string]decimal.Decimal{
					"portfolio.firmwide_equity_voting_pct": decimal.Zero,
				},
			}

			var results []engine.PostTradeEvaluationResult
			var evalErr error
			for attempt := 1; attempt <= 3; attempt++ {
				evalCtx, evalCancel := context.WithTimeout(context.Background(), 30*time.Second)
				t0 := time.Now()
				results, evalErr = evaluator.EvaluateAndPersist(evalCtx, state)
				evalCancel()
				if evalErr == nil {
					elapsed := time.Since(t0)
					latencies = append(latencies, elapsed)
					break
				}
				log.Printf("[WARN] Soak evaluation attempt %d failed on cycle %d (retrying in 2s): %v", attempt, cycle, evalErr)
				time.Sleep(2 * time.Second)
			}

			if evalErr != nil {
				log.Fatalf("FATAL: Soak evaluation failed on cycle %d after 3 attempts: %v", cycle, evalErr)
			}

			totalEvaluations += int64(len(results))
			for _, r := range results {
				if r.Action == "BREACHED" {
					totalBreaches++
				} else if r.Action == "WARNING" {
					totalWarnings++
				}
			}
		}

		cycleElapsed := time.Since(cycleStart)

		// Collect cycle memory metric after GC to measure true retained heap
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		var latestPause uint64 = 0
		if m.NumGC > 0 {
			latestPause = m.PauseNs[(m.NumGC+255)%256]
			gcPauses = append(gcPauses, latestPause)
		}

		metric := CycleMetric{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Cycle:      cycle,
			HeapAlloc:  m.Alloc,
			TotalAlloc: m.TotalAlloc,
			NumGC:      m.NumGC,
			GCPauseNS:  latestPause,
			LatencyMS:  float64(cycleElapsed.Microseconds()) / 1000.0,
		}
		cycleMetrics = append(cycleMetrics, metric)

		// Write to CSV
		if csvWriter != nil {
			_ = csvWriter.Write([]string{
				metric.Timestamp,
				strconv.Itoa(metric.Cycle),
				strconv.FormatUint(metric.HeapAlloc, 10),
				strconv.FormatUint(metric.TotalAlloc, 10),
				strconv.FormatUint(uint64(metric.NumGC), 10),
				strconv.FormatUint(metric.GCPauseNS, 10),
				fmt.Sprintf("%.3f", metric.LatencyMS),
			})
			csvWriter.Flush()
		}

		// Write to JSONL
		if jsonlFile != nil {
			line, _ := json.Marshal(metric)
			_, _ = jsonlFile.Write(append(line, '\n'))
		}

		if cycle <= 5 || cycle%10 == 0 || cycle == *cyclesFlag || (*durationFlag > 0 && time.Since(startTime) >= *durationFlag) {
			fmt.Printf("  -> Completed Cycle %4d | Elapsed: %v | Heap: %.2f MB | Decisions: %d\n",
				cycle, time.Since(startTime).Round(time.Millisecond), float64(m.Alloc)/1024/1024, totalEvaluations)
		}

		if *rateLimitFlag > 0 {
			time.Sleep(*rateLimitFlag)
		}
	}

	var memStatsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStatsAfter)

	totalElapsed := time.Since(startTime)

	// Calculate latency percentiles
	var sumLat time.Duration
	for _, l := range latencies {
		sumLat += l
	}
	avgLat := time.Duration(0)
	if len(latencies) > 0 {
		avgLat = sumLat / time.Duration(len(latencies))
	}

	var sumCycleLat float64
	for _, cm := range cycleMetrics {
		sumCycleLat += cm.LatencyMS
	}
	avgCycleLatMS := 0.0
	if len(cycleMetrics) > 0 {
		avgCycleLatMS = sumCycleLat / float64(len(cycleMetrics))
	}

	// Calculate GC Pause p99
	var p99GCPauseNS uint64 = 0
	if len(gcPauses) > 0 {
		sortedPauses := make([]uint64, len(gcPauses))
		copy(sortedPauses, gcPauses)
		sort.Slice(sortedPauses, func(i, j int) bool { return sortedPauses[i] < sortedPauses[j] })
		p99Idx := int(float64(len(sortedPauses)) * 0.99)
		if p99Idx >= len(sortedPauses) {
			p99Idx = len(sortedPauses) - 1
		}
		p99GCPauseNS = sortedPauses[p99Idx]
	}

	// Linear regression on steady-state heap allocations (skipping warmup cycle 1 if N >= 3)
	regressionData := cycleMetrics
	if len(cycleMetrics) >= 3 {
		regressionData = cycleMetrics[1:]
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(regressionData))
	for i, cm := range regressionData {
		x := float64(i + 1)
		y := float64(cm.HeapAlloc) / 1024.0 / 1024.0 // MB
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	slopeMBPerCycle := 0.0
	percentGrowth := 0.0
	if n > 1 && (n*sumX2-sumX*sumX) != 0 {
		slopeMBPerCycle = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
		meanY := sumY / n
		if meanY > 0 {
			percentGrowth = (slopeMBPerCycle * n) / meanY
		}
	}

	fmt.Println("\n======================================================================")
	fmt.Println("PRODUCTION SOAK TEST RESULTS & STABILITY RECEIPT:")
	fmt.Println("======================================================================")
	fmt.Printf("  - Total Soak Duration      : %v\n", totalElapsed.Round(time.Millisecond))
	fmt.Printf("  - Total Completed Cycles   : %d\n", len(cycleMetrics))
	fmt.Printf("  - Total Rule Decisions     : %d\n", totalEvaluations)
	fmt.Printf("  - Breaches Flagged         : %d\n", totalBreaches)
	fmt.Printf("  - Warnings Flagged         : %d\n", totalWarnings)
	fmt.Printf("  - Avg Per-Account Latency  : %v\n", avgLat.Round(time.Microsecond))
	fmt.Printf("  - Avg Per-Cycle Latency    : %.2f ms (5 accounts evaluated)\n", avgCycleLatMS)
	fmt.Printf("  - Memory Allocated (Start) : %.2f MB\n", float64(memStatsBefore.Alloc)/1024/1024)
	fmt.Printf("  - Memory Allocated (End)   : %.2f MB\n", float64(memStatsAfter.Alloc)/1024/1024)
	fmt.Printf("  - Heap Growth Delta        : %+.2f MB\n", float64(memStatsAfter.Alloc-memStatsBefore.Alloc)/1024/1024)
	if len(cycleMetrics) < 30 {
		fmt.Printf("  - Heap Slope Per Cycle     : %+.4f MB/cycle (Growth: %+.2f%%) [INSUFFICIENT DATA FOR SLOPE VERDICT (N < 30)]\n", slopeMBPerCycle, percentGrowth*100)
	} else {
		fmt.Printf("  - Heap Slope Per Cycle     : %+.4f MB/cycle (Growth: %+.2f%%)\n", slopeMBPerCycle, percentGrowth*100)
	}
	fmt.Printf("  - Total GC Collections     : %d\n", memStatsAfter.NumGC-memStatsBefore.NumGC)
	fmt.Printf("  - GC Pause p99             : %.2f ms (Threshold: 10.0 ms)\n", float64(p99GCPauseNS)/1e6)
	fmt.Println("======================================================================")

	// Invariant Checks & Exit Criteria
	failed := false
	if len(cycleMetrics) >= 30 {
		if percentGrowth > 0.15 {
			fmt.Printf("❌ FAIL: Heap growth slope (%.2f%%) exceeded 15%% leak threshold\n", percentGrowth*100)
			failed = true
		}
	} else {
		fmt.Printf("⚠️  NOTE: Run cycle count (%d) < 30; slope threshold assertion skipped.\n", len(cycleMetrics))
	}

	if float64(p99GCPauseNS)/1e6 > 10.0 {
		fmt.Printf("❌ FAIL: GC Pause p99 (%.2f ms) exceeded 10.0 ms threshold\n", float64(p99GCPauseNS)/1e6)
		failed = true
	}

	if failed {
		os.Exit(1)
	}

	fmt.Println("STATUS: 100% PASS — Continuous Evaluation Stream Confirmed Stable.")
}
