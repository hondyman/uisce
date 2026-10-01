package analytics

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCompileWindowRuleSQL_HappyPath(t *testing.T) {
	spec := models.AggregateRuleSpec{
		Window:     "1h",
		Aggregate:  "sum",
		Field:      "OrderQty",
		GroupBy:    []string{"AccountID", "BrokerID"},
		TimeColumn: "OrderTime",
		Threshold:  1000000.0,
		Comparison: "greater_than",
	}

	columnMap := map[string]string{
		"OrderQty":   "qty",
		"AccountID":  "account_id",
		"BrokerID":   "broker_id",
		"OrderTime":  "order_time",
	}

	res, err := CompileWindowRuleSQL(spec, "oms.trade_order", columnMap)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.Contains(t, res.Query, `SELECT "account_id", "broker_id", date_trunc('hour', "order_time") AS window_start, date_trunc('hour', "order_time") + interval '1 hour' AS window_end, SUM("qty") AS metric_value`)
	assert.Contains(t, res.Query, `FROM "oms"."trade_order"`)
	assert.Contains(t, res.Query, `WHERE tenant_id = $1 AND "order_time" >= $2 AND "order_time" < $3`)
	assert.Contains(t, res.Query, `GROUP BY "account_id", "broker_id", date_trunc('hour', "order_time")`)
	assert.Contains(t, res.Query, `HAVING SUM("qty") > 1000000.000000`)
	assert.Equal(t, []string{"account_id", "broker_id"}, res.GroupCols)
}

func TestCompileWindowRuleSQL_CountAll(t *testing.T) {
	spec := models.AggregateRuleSpec{
		Window:     "1d",
		Aggregate:  "count",
		GroupBy:    []string{"AccountID"},
		TimeColumn: "CreatedAt",
		Threshold:  500.0,
		Comparison: "greater_or_equal",
	}

	columnMap := map[string]string{
		"AccountID": "account_id",
		"CreatedAt": "created_at",
	}

	res, err := CompileWindowRuleSQL(spec, "trade_order", columnMap)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Contains(t, res.Query, `COUNT(*) AS metric_value`)
	assert.Contains(t, res.Query, `date_trunc('day', "created_at")`)
	assert.Contains(t, res.Query, `HAVING COUNT(*) >= 500.000000`)
}

func TestCompileWindowRuleSQL_InvalidIdentifierRejection(t *testing.T) {
	spec := models.AggregateRuleSpec{
		Window:     "1h",
		Aggregate:  "sum",
		Field:      "OrderQty; DROP TABLE users; --",
		GroupBy:    []string{"AccountID"},
		TimeColumn: "OrderTime",
		Threshold:  100.0,
		Comparison: "greater_than",
	}

	_, err := CompileWindowRuleSQL(spec, "oms.trade_order", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid identifier")
}
