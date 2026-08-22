package boresolver

import (
	"testing"
)

func TestSQLDialects_MultiEngineTranspilation(t *testing.T) {
	pg := ResolveDialect(DialectPostgreSQL)
	sf := ResolveDialect(DialectSnowflake)
	sr := ResolveDialect(DialectStarRocks)
	duck := ResolveDialect(DialectDuckDB)

	// 1. DateTrunc
	if pg.DateTrunc("month", "trade_date") != "DATE_TRUNC('month', trade_date)" {
		t.Errorf("unexpected Postgres DateTrunc: %s", pg.DateTrunc("month", "trade_date"))
	}
	if sf.DateTrunc("month", "trade_date") != "DATE_TRUNC('MONTH', trade_date)" {
		t.Errorf("unexpected Snowflake DateTrunc: %s", sf.DateTrunc("month", "trade_date"))
	}
	if sr.DateTrunc("day", "created_at") != "DATE_TRUNC('day', created_at)" {
		t.Errorf("unexpected StarRocks DateTrunc: %s", sr.DateTrunc("day", "created_at"))
	}
	if duck.DateTrunc("month", "trade_date") != "DATE_TRUNC('month', trade_date)" {
		t.Errorf("unexpected DuckDB DateTrunc: %s", duck.DateTrunc("month", "trade_date"))
	}

	// 2. NullSafeCoalesce
	if pg.NullSafeCoalesce("col", "'N/A'") != "COALESCE(col, 'N/A')" {
		t.Errorf("unexpected Postgres Coalesce: %s", pg.NullSafeCoalesce("col", "'N/A'"))
	}
	if sf.NullSafeCoalesce("col", "'N/A'") != "IFNULL(col, 'N/A')" {
		t.Errorf("unexpected Snowflake Coalesce: %s", sf.NullSafeCoalesce("col", "'N/A'"))
	}

	// 3. JSONExtract
	if pg.JSONExtract("payload", "customer_id") != "payload->>'customer_id'" {
		t.Errorf("unexpected Postgres JSON: %s", pg.JSONExtract("payload", "customer_id"))
	}
	if sf.JSONExtract("payload", "customer_id") != "GET_PATH(payload, 'customer_id')::VARCHAR" {
		t.Errorf("unexpected Snowflake JSON: %s", sf.JSONExtract("payload", "customer_id"))
	}
	if sr.JSONExtract("payload", "customer_id") != "json_query(payload, '$.customer_id')" {
		t.Errorf("unexpected StarRocks JSON: %s", sr.JSONExtract("payload", "customer_id"))
	}

	// 4. QuoteIdentifier
	if pg.QuoteIdentifier("user_table") != `"user_table"` {
		t.Errorf("unexpected Postgres quote: %s", pg.QuoteIdentifier("user_table"))
	}
	if sf.QuoteIdentifier("user_table") != `"USER_TABLE"` {
		t.Errorf("unexpected Snowflake quote: %s", sf.QuoteIdentifier("user_table"))
	}
	if sr.QuoteIdentifier("user_table") != "`user_table`" {
		t.Errorf("unexpected StarRocks quote: %s", sr.QuoteIdentifier("user_table"))
	}
	if duck.QuoteIdentifier("user_table") != `"user_table"` {
		t.Errorf("unexpected DuckDB quote: %s", duck.QuoteIdentifier("user_table"))
	}
}
