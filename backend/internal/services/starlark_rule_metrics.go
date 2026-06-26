package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type starlarkRuleMetricChildren struct {
	duration  prometheus.Observer
	lastRun   prometheus.Gauge
	lastError prometheus.Gauge
}

var (
	starlarkRuleEvalTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_starlark_rule_evaluations_total",
			Help: "Count of Starlark rule evaluations by rule and outcome.",
		},
		[]string{"rule_id", "mode", "outcome"},
	)

	starlarkRuleEvalDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semlayer_starlark_rule_evaluation_duration_seconds",
			Help:    "Wall-clock duration of Starlark rule evaluations.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"rule_id", "mode"},
	)

	starlarkRuleLastRunUnix = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semlayer_starlark_rule_last_run_unixtime",
			Help: "Unix timestamp of the most recent Starlark rule evaluation.",
		},
		[]string{"rule_id", "mode"},
	)

	starlarkRuleLastErrorUnix = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semlayer_starlark_rule_last_error_unixtime",
			Help: "Unix timestamp of the most recent Starlark rule evaluation that resulted in an error.",
		},
		[]string{"rule_id", "mode"},
	)

	starlarkRuleMetricsOnce sync.Once
	starlarkRuleChildren    sync.Map // key: ruleID|mode -> *starlarkRuleMetricChildren
)

func ensureStarlarkRuleMetricsRegistered() {
	starlarkRuleMetricsOnce.Do(func() {
		registerOrReuse := func(c prometheus.Collector) {
			if err := prometheus.Register(c); err != nil {
				if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
					return
				}
			}
		}

		registerOrReuse(starlarkRuleEvalTotal)
		registerOrReuse(starlarkRuleEvalDurationSeconds)
		registerOrReuse(starlarkRuleLastRunUnix)
		registerOrReuse(starlarkRuleLastErrorUnix)
	})
}

func starlarkRuleIDFromScript(script string) string {
	sum := sha256.Sum256([]byte(script))
	// 12 hex chars is enough for practical uniqueness while keeping label size sane.
	return "script_" + hex.EncodeToString(sum[:])[:12]
}

func normalizeRuleID(ruleID string) string {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID == "" {
		return "unknown"
	}
	// Keep Prometheus label values from growing unbounded.
	if len(ruleID) > 96 {
		sum := sha256.Sum256([]byte(ruleID))
		return "id_" + hex.EncodeToString(sum[:])[:16]
	}
	return ruleID
}

func classifyStarlarkOutcome(res *StarlarkValidationResult, err error) string {
	if err != nil {
		return "error"
	}
	if res == nil {
		return "error"
	}
	if res.IsValid {
		return "pass"
	}

	msg := strings.ToLower(strings.TrimSpace(res.Message))
	if strings.HasPrefix(msg, "script error") || strings.HasPrefix(msg, "runtime error") || strings.Contains(msg, "did not define") || strings.Contains(msg, "not found") {
		return "error"
	}
	return "fail"
}

func starlarkRuleChildrenFor(ruleID, mode string) *starlarkRuleMetricChildren {
	ensureStarlarkRuleMetricsRegistered()

	normID := normalizeRuleID(ruleID)
	key := normID + "|" + mode
	if v, ok := starlarkRuleChildren.Load(key); ok {
		return v.(*starlarkRuleMetricChildren)
	}
	child := &starlarkRuleMetricChildren{
		duration:  starlarkRuleEvalDurationSeconds.WithLabelValues(normID, mode),
		lastRun:   starlarkRuleLastRunUnix.WithLabelValues(normID, mode),
		lastError: starlarkRuleLastErrorUnix.WithLabelValues(normID, mode),
	}
	actual, _ := starlarkRuleChildren.LoadOrStore(key, child)
	return actual.(*starlarkRuleMetricChildren)
}

func observeStarlarkRule(ruleID, mode string, duration time.Duration, outcome string) {
	ensureStarlarkRuleMetricsRegistered()

	normID := normalizeRuleID(ruleID)
	starlarkRuleEvalTotal.WithLabelValues(normID, mode, outcome).Inc()

	child := starlarkRuleChildrenFor(normID, mode)
	child.duration.Observe(duration.Seconds())
	child.lastRun.Set(float64(time.Now().Unix()))
	if outcome == "error" {
		child.lastError.Set(float64(time.Now().Unix()))
	}
}
