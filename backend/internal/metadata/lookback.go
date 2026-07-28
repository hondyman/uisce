package metadata

import (
	"fmt"
	"time"
)

// LookbackContext holds temporal parameters passed from the page runtime or compliance audit workspace
type LookbackContext struct {
	Enabled       bool      `json:"enabled"`
	AsOfTimestamp time.Time `json:"asOfTimestamp,omitempty"`
	HistoryMode   string    `json:"historyMode,omitempty"` // SNAPSHOT, SYSTEM_VERSIONED, EXPLICIT_RANGE
}

// ApplyLookbackToTableExpression wraps source table references with temporal syntax based on datasource capabilities
func ApplyLookbackToTableExpression(tableName string, datasourceType string, lookback LookbackContext) string {
	if !lookback.Enabled || lookback.AsOfTimestamp.IsZero() {
		return tableName
	}

	formattedTime := lookback.AsOfTimestamp.UTC().Format("2006-01-02 15:04:05.000")

	switch datasourceType {
	case "ICEBERG":
		// Apache Iceberg native time-travel syntax
		return fmt.Sprintf("%s FOR SYSTEM_TIME AS OF TIMESTAMP '%s'", tableName, formattedTime)

	case "POSTGRES_OLTP", "CITUS", "SYSTEM_VERSIONED":
		// For Postgres/Citus tables utilizing temporal history tracking
		return fmt.Sprintf("(SELECT * FROM %s WHERE sys_period @> TIMESTAMP '%s') AS %s", tableName, formattedTime, tableName)

	case "STARROCKS", "ARCHIVE", "SNAPSHOT":
		// For pre-aggregated or date-partitioned archive stores
		return fmt.Sprintf("%s /* AS_OF: %s */", tableName, formattedTime)

	default:
		// Default fallback
		return fmt.Sprintf("%s /* AS_OF: %s */", tableName, formattedTime)
	}
}
