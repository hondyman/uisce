package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	var err error
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping Postgres: %v", err)
	}

	log.Println("✅ Connected to Postgres")
}

func main() {
	defer db.Close()

	// Setup listener for metric_registry changes
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	listener := pq.NewListener(databaseURL, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[LISTENER] Event: %v, Error: %v", ev, err)
		}
	})
	defer listener.Close()

	// Listen on the metrics_registry_changed channel
	if err := listener.Listen("metrics_registry_changed"); err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("🎧 Semantic Sync Service started. Listening for metrics_registry changes...")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create schema directory if it doesn't exist
	os.MkdirAll("./cube-schemas", 0755)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case notification := <-listener.Notify:
			if notification != nil {
				log.Printf("[NOTIFY] Received notification: %s", notification.Extra)
				if err := regenerateCubeSchemas(); err != nil {
					log.Printf("[ERROR] Failed to regenerate schemas: %v", err)
				} else {
					log.Println("✅ [SUCCESS] Cube schemas regenerated")
				}
			}

		case <-ticker.C:
			log.Println("[PERIODIC] Running periodic schema regeneration...")
			if err := regenerateCubeSchemas(); err != nil {
				log.Printf("[ERROR] Periodic regeneration failed: %v", err)
			}

		case <-sigChan:
			log.Println("🛑 Shutting down gracefully...")
			return
		}
	}
}

type MetricRegistry struct {
	TenantID                 string
	MetricID                 string
	Name                     string
	Domain                   string
	Granularity              string
	AggregationFunction      string
	SlaFreshnessHours        int
	SlaCompletenessThreshold float64
	GoldenPath               bool
	OwnerUserID              string
	StewardGroup             string
}

func regenerateCubeSchemas() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch all metrics from registry
	rows, err := db.QueryContext(ctx, `
		SELECT 
			COALESCE(tenant_id, ''), 
			metric_id, 
			name, 
			COALESCE(domain, 'general'), 
			COALESCE(granularity, 'month'), 
			COALESCE(aggregation_function, 'sum'),
			COALESCE(sla_freshness_hours, 24),
			COALESCE(sla_completeness_threshold, 0.95),
			COALESCE(golden_path, false),
			COALESCE(owner_user_id, ''),
			COALESCE(steward_group, '')
		FROM metrics_registry
		ORDER BY domain, name
	`)
	if err != nil {
		return fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []MetricRegistry
	for rows.Next() {
		var m MetricRegistry

		err := rows.Scan(
			&m.TenantID, &m.MetricID, &m.Name, &m.Domain, &m.Granularity,
			&m.AggregationFunction, &m.SlaFreshnessHours,
			&m.SlaCompletenessThreshold, &m.GoldenPath, &m.OwnerUserID,
			&m.StewardGroup,
		)
		if err != nil {
			log.Printf("Failed to scan metric: %v", err)
			continue
		}

		metrics = append(metrics, m)
	}

	if len(metrics) == 0 {
		log.Println("ℹ️  No metrics found in registry")
		return nil
	}

	// Generate PoP schema
	if err := generatePopSchema(metrics); err != nil {
		return fmt.Errorf("failed to generate PoP schema: %w", err)
	}

	// Generate Anomaly schema
	if err := generateAnomalySchema(metrics); err != nil {
		return fmt.Errorf("failed to generate Anomaly schema: %w", err)
	}

	// Generate Base Metrics schema
	if err := generateBaseMetricsSchema(metrics); err != nil {
		return fmt.Errorf("failed to generate Base Metrics schema: %w", err)
	}

	log.Printf("📊 Successfully generated schemas for %d metrics", len(metrics))
	return nil
}

func generatePopSchema(metrics []MetricRegistry) error {
	popSchema := `cube('MetricsPop', {
  sql: ` + "`" + `SELECT 
    tenant_id, 
    metric_id, 
    period_start, 
    period_end, 
    period_label,
    record_count, 
    current_value, 
    previous_value, 
    delta, 
    percent_change,
    computation_status, 
    computation_id, 
    last_updated, 
    created_at
  FROM iceberg.catalog.metrics_pop` + "`" + `,

  measures: {
    currentValue: {
      sql: 'current_value',
      type: 'sum',
      title: 'Current Value'
    },
    previousValue: {
      sql: 'previous_value',
      type: 'sum',
      title: 'Previous Value'
    },
    delta: {
      sql: 'delta',
      type: 'sum',
      title: 'Delta (Absolute Change)'
    },
    percentChange: {
      sql: 'percent_change',
      type: 'avg',
      title: 'Percent Change (%)'
    },
    recordCount: {
      sql: 'record_count',
      type: 'sum',
      title: 'Record Count'
    },
    successCount: {
      sql: 'CASE WHEN computation_status = \'success\' THEN 1 ELSE 0 END',
      type: 'count',
      title: 'Successful Computations'
    }
  },

  dimensions: {
    tenantId: {
      sql: 'tenant_id',
      type: 'string',
      title: 'Tenant ID',
      primaryKey: true
    },
    metricId: {
      sql: 'metric_id',
      type: 'string',
      title: 'Metric ID'
    },
    periodLabel: {
      sql: 'period_label',
      type: 'string',
      title: 'Period'
    },
    periodStart: {
      sql: 'period_start',
      type: 'time',
      title: 'Period Start'
    },
    computationStatus: {
      sql: 'computation_status',
      type: 'string',
      title: 'Status'
    }
  },

  preAggregations: {
    popMonthly: {
      type: 'rollup',
      measures: [
        'currentValue',
        'previousValue',
        'delta',
        'percentChange',
        'recordCount',
        'successCount'
      ],
      dimensions: [
        'tenantId',
        'metricId',
        'periodLabel',
        'computationStatus'
      ],
      timeDimension: 'periodStart',
      granularity: 'month',
      partitionGranularity: 'month',
      refreshKey: {
        sql: ` + "`" + `SELECT MAX(last_updated) FROM iceberg.catalog.metrics_pop` + "`" + `,
        every: '10 minutes'
      },
      incremental: true,
      updateWindow: '7 day'
    }
  }
});
`

	return writeSchemaFile("metrics_pop.js", popSchema)
}

func generateAnomalySchema(metrics []MetricRegistry) error {
	anomalySchema := `cube('MetricsAnomalies', {
  sql: ` + "`" + `SELECT 
    tenant_id, 
    metric_id, 
    anomaly_type, 
    detected_at, 
    severity,
    confidence, 
    actual_value, 
    expected_value, 
    expected_range_min,
    expected_range_max, 
    detection_params, 
    computation_id, 
    status, 
    created_at,
    resolved_at
  FROM iceberg.catalog.metrics_anomalies` + "`" + `,

  measures: {
    anomalyCount: {
      type: 'count',
      title: 'Total Anomalies'
    },
    avgConfidence: {
      sql: 'confidence',
      type: 'avg',
      title: 'Average Confidence'
    },
    criticalAnomalyCount: {
      sql: 'CASE WHEN severity = \'critical\' THEN 1 ELSE 0 END',
      type: 'count',
      title: 'Critical Anomalies'
    },
    highAnomalyCount: {
      sql: 'CASE WHEN severity = \'high\' THEN 1 ELSE 0 END',
      type: 'count',
      title: 'High Severity Anomalies'
    }
  },

  dimensions: {
    tenantId: {
      sql: 'tenant_id',
      type: 'string',
      title: 'Tenant ID',
      primaryKey: true
    },
    metricId: {
      sql: 'metric_id',
      type: 'string',
      title: 'Metric ID'
    },
    anomalyType: {
      sql: 'anomaly_type',
      type: 'string',
      title: 'Anomaly Type'
    },
    severity: {
      sql: 'severity',
      type: 'string',
      title: 'Severity'
    },
    detectedAt: {
      sql: 'detected_at',
      type: 'time',
      title: 'Detected At'
    },
    status: {
      sql: 'status',
      type: 'string',
      title: 'Status'
    }
  },

  preAggregations: {
    anomaliesDaily: {
      type: 'rollup',
      measures: [
        'anomalyCount',
        'avgConfidence',
        'criticalAnomalyCount',
        'highAnomalyCount'
      ],
      dimensions: [
        'tenantId',
        'metricId',
        'anomalyType',
        'severity',
        'status'
      ],
      timeDimension: 'detectedAt',
      granularity: 'day',
      partitionGranularity: 'day',
      refreshKey: {
        sql: ` + "`" + `SELECT MAX(detected_at) FROM iceberg.catalog.metrics_anomalies` + "`" + `,
        every: '10 minutes'
      },
      incremental: true,
      updateWindow: '14 day'
    }
  }
});
`

	return writeSchemaFile("metrics_anomalies.js", anomalySchema)
}

func generateBaseMetricsSchema(metrics []MetricRegistry) error {
	baseSchema := `cube('MetricsAtomic', {
  sql: ` + "`" + `SELECT 
    tenant_id, 
    metric_id, 
    name, 
    as_of_date, 
    value,
    tags, 
    details, 
    data_quality, 
    created_at, 
    updated_at
  FROM iceberg.catalog.metrics_atomic` + "`" + `,

  measures: {
    value: {
      sql: 'value',
      type: 'sum',
      title: 'Metric Value'
    },
    avgValue: {
      sql: 'value',
      type: 'avg',
      title: 'Average Value'
    },
    recordCount: {
      type: 'count',
      title: 'Records'
    }
  },

  dimensions: {
    tenantId: {
      sql: 'tenant_id',
      type: 'string',
      title: 'Tenant ID',
      primaryKey: true
    },
    metricId: {
      sql: 'metric_id',
      type: 'string',
      title: 'Metric ID'
    },
    metricName: {
      sql: 'name',
      type: 'string',
      title: 'Metric Name'
    },
    asOfDate: {
      sql: 'as_of_date',
      type: 'time',
      title: 'As Of Date'
    }
  },

  preAggregations: {
    atomicDaily: {
      type: 'rollup',
      measures: ['value', 'avgValue', 'recordCount'],
      dimensions: ['tenantId', 'metricId', 'metricName'],
      timeDimension: 'asOfDate',
      granularity: 'day',
      partitionGranularity: 'day',
      refreshKey: {
        sql: ` + "`" + `SELECT MAX(updated_at) FROM iceberg.catalog.metrics_atomic` + "`" + `,
        every: '5 minutes'
      },
      incremental: true,
      updateWindow: '3 day'
    }
  }
});
`

	return writeSchemaFile("metrics_atomic.js", baseSchema)
}

func writeSchemaFile(filename string, content string) error {
	path := fmt.Sprintf("./cube-schemas/%s", filename)
	return os.WriteFile(path, []byte(content), 0644)
}
