package metadata

import (
	"fmt"
	"strings"
)

type BiTemporalConfig struct {
	ValidStartCol       string `json:"valid_time_start_col"`
	ValidEndCol         string `json:"valid_time_end_col"`
	TransactionStartCol string `json:"transaction_time_start_col"`
	TransactionEndCol   string `json:"transaction_time_end_col"`
}

// BuildBiTemporalWhereClause appends time-travel bounds to OLAP analytics queries for Valid Time and Transaction Time
func BuildBiTemporalWhereClause(config BiTemporalConfig, asOfValidTime, asOfTransactionTime string) string {
	var predicates []string

	if asOfValidTime != "" && config.ValidStartCol != "" && config.ValidEndCol != "" {
		// Valid Time: Fact was true in reality at this timestamp
		p := fmt.Sprintf("('%s' >= %s AND '%s' < COALESCE(%s, '9999-12-31 23:59:59'))",
			asOfValidTime, config.ValidStartCol, asOfValidTime, config.ValidEndCol)
		predicates = append(predicates, p)
	}

	if asOfTransactionTime != "" && config.TransactionStartCol != "" && config.TransactionEndCol != "" {
		// Transaction Time: Fact was known to the system at this timestamp
		p := fmt.Sprintf("('%s' >= %s AND '%s' < COALESCE(%s, '9999-12-31 23:59:59'))",
			asOfTransactionTime, config.TransactionStartCol, asOfTransactionTime, config.TransactionEndCol)
		predicates = append(predicates, p)
	}

	if len(predicates) == 0 {
		return ""
	}
	return "AND " + strings.Join(predicates, " AND ")
}
