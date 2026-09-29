package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestExecuteWindowRule_BreachedThreshold(t *testing.T) {
	dataDB, dataMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dataDB.Close()
	dataSqlx := sqlx.NewDb(dataDB, "sqlmock")

	alphaDB, alphaMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer alphaDB.Close()
	alphaSqlx := sqlx.NewDb(alphaDB, "sqlmock")

	tenantID := uuid.New().String()
	ruleID := uuid.New().String()
	wStart := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
	wEnd := time.Date(2026, 9, 29, 16, 0, 0, 0, time.UTC)

	spec := models.AggregateRuleSpec{
		Window:     "1h",
		Aggregate:  "sum",
		Field:      "OrderQty",
		GroupBy:    []string{"AccountID", "BrokerID"},
		TimeColumn: "OrderTime",
		Threshold:  1000000.0,
		Comparison: "greater_than",
	}

	snap := &RuleSnapshot{
		TenantID: tenantID,
		BOName:   "order",
		ColumnMap: map[string]string{
			"OrderQty":  "qty",
			"AccountID": "account_id",
			"BrokerID":  "broker_id",
			"OrderTime": "order_time",
		},
	}

	rule := &EvaluableRule{
		RuleID:      ruleID,
		RuleKey:     "hourly_broker_limit",
		RuleName:    "Hourly Broker Volume Limit",
		Severity:    "WARN",
		RuleVersion: "1",
	}

	// 1. Mock query on dataDB
	dataMock.ExpectQuery(`SELECT "account_id", "broker_id", date_trunc\('hour', "order_time"\) AS window_start`).
		WithArgs(tenantID, wStart, wEnd).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "broker_id", "window_start", "window_end", "metric_value"}).
			AddRow("acc-101", "brk-88", wStart, wEnd, 1500000.0))

	// 2. Mock persist violation on alphaDB
	alphaMock.ExpectExec(`INSERT INTO validation_rule_violations`).
		WithArgs(tenantID, ruleID, "Hourly Broker Volume Limit", "order", "WARN", "group:order:acc-101_brk-88:2026-09-29 15:00:00 +0000 UTC", sqlmock.AnyArg(), sqlmock.AnyArg(), false, false, 1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := ExecuteWindowRule(context.Background(), dataSqlx, alphaSqlx, spec, "oms.trade_order", snap, rule, wStart, wEnd)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 1, res.Violations)
	assert.Equal(t, 1, res.PersistedCount)
	assert.Equal(t, "group:order:acc-101_brk-88:2026-09-29 15:00:00 +0000 UTC", res.ViolationsList[0].RecordID)
	assert.Equal(t, "WARN", res.ViolationsList[0].Severity)
	assert.False(t, res.ViolationsList[0].WriteBlocked)
}
