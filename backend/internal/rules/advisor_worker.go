package rules

import (
	"context"
	"log"
	"time"
)

type AdvisorWorker struct {
	engine    *RuleEngine
	interval  time.Duration
	stopChan  chan struct{}
	threshold uint64
}

func NewAdvisorWorker(engine *RuleEngine, interval time.Duration, threshold uint64) *AdvisorWorker {
	return &AdvisorWorker{
		engine:    engine,
		interval:  interval,
		stopChan:  make(chan struct{}),
		threshold: threshold,
	}
}

func (w *AdvisorWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.checkFallbackThresholds()
			case <-w.stopChan:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (w *AdvisorWorker) checkFallbackThresholds() {
	metrics := w.engine.MetricsSnapshot()
	if metrics.Fallbacks >= w.threshold {
		log.Printf("[AdvisorWorker] Fallback count (%d) exceeded threshold (%d). Analyzing fallback query patterns for StarRocks MV candidates...", metrics.Fallbacks, w.threshold)
	}
}

func (w *AdvisorWorker) Stop() {
	close(w.stopChan)
}
