package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveBusinessObject(t *testing.T) {
	tests := []struct {
		table    string
		expected string
	}{
		{"orm_order", "order"},
		{"orm_execution", "execution"},
		{"oms_position", "position"},
		{"mdm_security", "security"},
		{"placement", "placement"},
		{"orm_order_allocation", "order_allocation"},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			assert.Equal(t, tt.expected, deriveBusinessObject(tt.table))
		})
	}
}

func TestExtractTenantID(t *testing.T) {
	assert.Equal(t, "t-123", extractTenantID(map[string]interface{}{"tenant_id": "t-123"}))
	assert.Equal(t, "t-456", extractTenantID(map[string]interface{}{"tenantId": "t-456"}))
	assert.Equal(t, "t-789", extractTenantID(map[string]interface{}{"TenantID": "t-789"}))
	assert.Equal(t, "", extractTenantID(map[string]interface{}{"other": "val"}))
	assert.Equal(t, "", extractTenantID(nil))
}

func TestDecodeDebeziumDecimal(t *testing.T) {
	// 123.4567 with scale 4 -> 1234567 in base64 big-endian
	val := big.NewInt(1234567)
	b64 := base64.StdEncoding.EncodeToString(val.Bytes())

	decoded, err := decodeDebeziumDecimal(b64, "4")
	require.NoError(t, err)
	assert.Equal(t, "123.4567", decoded)

	// Zero scale
	valZero := big.NewInt(500)
	b64Zero := base64.StdEncoding.EncodeToString(valZero.Bytes())
	decodedZero, err := decodeDebeziumDecimal(b64Zero, "0")
	require.NoError(t, err)
	assert.Equal(t, "500", decodedZero)
}

// envelope builds a Debezium change event with a connector schema describing one
// logical-typed field inside "after".
func envelope(op string, before, after map[string]interface{}, schemaFieldName string, scale string) []byte {
	fields := []map[string]interface{}{}
	if schemaFieldName != "" {
		f := map[string]interface{}{"field": "price", "name": schemaFieldName}
		if scale != "" {
			f["parameters"] = map[string]string{"scale": scale}
		}
		fields = append(fields, f)
	}
	sample := map[string]interface{}{
		"schema": map[string]interface{}{
			"type": "struct",
			"fields": []map[string]interface{}{
				{"type": "struct", "field": "after", "fields": fields},
			},
		},
		"payload": map[string]interface{}{"op": op},
	}
	p := sample["payload"].(map[string]interface{})
	if before != nil {
		p["before"] = before
	}
	if after != nil {
		p["after"] = after
	}
	raw, _ := json.Marshal(sample)
	return raw
}

// jsonEnvelopeWithField declares a specific logical-typed column inside "after".
func jsonEnvelopeWithField(op string, before, after map[string]interface{}, field, logicalType string) []byte {
	sample := map[string]interface{}{
		"schema": map[string]interface{}{
			"type": "struct",
			"fields": []map[string]interface{}{
				{"type": "struct", "field": "after", "fields": []map[string]interface{}{
					{"field": field, "type": logicalType},
				}},
			},
		},
		"payload": map[string]interface{}{"op": op},
	}
	p := sample["payload"].(map[string]interface{})
	if before != nil {
		p["before"] = before
	}
	if after != nil {
		p["after"] = after
	}
	raw, _ := json.Marshal(sample)
	return raw
}

func TestDecodeRecordUpsert(t *testing.T) {
	val := big.NewInt(450000) // 45.0000 with scale 4
	b64Price := base64.StdEncoding.EncodeToString(val.Bytes())

	raw := envelope("c", nil, map[string]interface{}{
		"id":        "ord-1",
		"tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
		"price":     b64Price,
		"quantity":  1000,
	}, "org.apache.kafka.connect.data.Decimal", "4")

	ev, err := decodeRecord(raw)
	require.NoError(t, err)
	assert.False(t, ev.tombstone)
	assert.False(t, ev.isDelete)
	require.NotNil(t, ev.row)
	assert.Equal(t, "45.0000", ev.after["price"])
	assert.Equal(t, "ord-1", ev.after["id"])
	assert.Equal(t, "910638ba-a459-4a3f-bb2d-78391b0595f6", extractTenantID(ev.after))
}

func TestDecodeRecordDeleteCarriesBefore(t *testing.T) {
	// A delete has after=null and the previous row image in before. Its key lives
	// there, so the loader must be able to build a predicate.
	raw := envelope("d", map[string]interface{}{"id": "ord-9", "tenant_id": "t-1"}, nil, "", "")

	ev, err := decodeRecord(raw)
	require.NoError(t, err)
	assert.False(t, ev.tombstone, "a delete with a usable before image is NOT a tombstone")
	assert.True(t, ev.isDelete)
	require.NotNil(t, ev.before)
	assert.Equal(t, "ord-9", ev.before["id"])
}

func TestDecodeRecordDeleteWithoutBeforeIsTombstone(t *testing.T) {
	raw := envelope("d", nil, nil, "", "")
	ev, err := decodeRecord(raw)
	require.NoError(t, err)
	assert.True(t, ev.tombstone, "a delete with no before image cannot be represented; drop it")
}

func TestDecodeRecordTombstone(t *testing.T) {
	sample := map[string]interface{}{
		"payload": map[string]interface{}{"op": "d", "after": nil},
	}
	raw, err := json.Marshal(sample)
	require.NoError(t, err)

	ev, err := decodeRecord(raw)
	require.NoError(t, err)
	assert.True(t, ev.tombstone)
	assert.Nil(t, ev.row)
	assert.Nil(t, ev.after)
}

func TestDecodeRecordMalformedJSON(t *testing.T) {
	_, err := decodeRecord([]byte("invalid json"))
	assert.Error(t, err)
}

func TestNormalizeTemporalTypes(t *testing.T) {
	// StarRocks cannot ingest Debezium's temporal encodings directly; every one of
	// these must arrive as a SQL literal.
	t.Run("ZonedTimestamp becomes a UTC datetime literal", func(t *testing.T) {
		lit, ok := zonedToDatetime("2026-09-10T02:25:16.012357Z")
		require.True(t, ok)
		assert.Equal(t, "2026-09-10 02:25:16.012357", lit)
	})

	t.Run("ZonedTimestamp with an offset is converted to UTC", func(t *testing.T) {
		lit, ok := zonedToDatetime("2026-09-10T04:25:16+02:00")
		require.True(t, ok)
		assert.Equal(t, "2026-09-10 02:25:16.000000", lit)
	})

	t.Run("Date epoch days becomes a calendar date", func(t *testing.T) {
		days := int(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC).Sub(time.Unix(0, 0).UTC()).Hours() / 24)
		assert.Equal(t, "2026-09-10", epochDaysToDate(int64(days)))
	})

	t.Run("MicroTimestamp epoch micros becomes a datetime literal", func(t *testing.T) {
		micros := time.Date(2026, 9, 10, 2, 25, 16, 123000000, time.UTC).UnixMicro()
		lit, ok := epochToDatetime(micros, "MicroTimestamp")
		require.True(t, ok)
		assert.Equal(t, "2026-09-10 02:25:16.123000", lit)
	})

	t.Run("Json is decoded from base64", func(t *testing.T) {
		raw := jsonEnvelopeWithField("c",
			map[string]interface{}{"id": "ord-2"},
			map[string]interface{}{
				"id":                "ord-2",
				"custom_attributes": base64.StdEncoding.EncodeToString([]byte(`{"a":1}`)),
			},
			"custom_attributes", "io.debezium.data.Json")
		ev, err := decodeRecord(raw)
		require.NoError(t, err)
		assert.Equal(t, `{"a":1}`, ev.after["custom_attributes"])
	})
}

func TestPrimaryKeyOf(t *testing.T) {
	row := map[string]interface{}{"id": "abc", "other": 1}
	k, ok := primaryKeyOf([]string{"id"}, row)
	assert.True(t, ok)
	assert.Equal(t, "abc", k)

	_, ok = primaryKeyOf([]string{"id", "missing"}, row)
	assert.False(t, ok, "a key with an absent column is not usable")

	_, ok = primaryKeyOf([]string{"id"}, map[string]interface{}{"id": nil})
	assert.False(t, ok)
}

func TestWriteBatchCoalescesLastOperationWins(t *testing.T) {
	var gotBody string
	var gotLabel string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oms/orm_order/_stream_load" {
			w.Write([]byte(`{"Status":"Success"}`))
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotLabel = r.Header.Get("label")
		w.Write([]byte(`{"Status":"Success","Message":"ok"}`))
	}))
	defer ts.Close()

	gk := &StreamingGatekeeper{
		cfg: Config{
			StarRocksHTTP:  ts.URL,
			StarRocksDB:    "oms",
			StarRocksTable: "orm_order",
			Topic:          "orm_oms.orm.order",
			PrimaryKeys:    []string{"id"},
		},
		metrics: &GatekeeperMetrics{},
	}

	b := &batch{
		ops: []pendingOp{
			// deleted, then created: the upsert must win.
			{key: "k2", isDelete: true},
			{key: "k2", row: json.RawMessage(`{"id":"k2","v":2}`)},
		},
	}

	require.NoError(t, gk.writeBatch(context.Background(), b))

	assert.JSONEq(t, `[{"id":"k2","v":2}]`, gotBody)
	assert.NotEmpty(t, gotLabel, "every batch must carry a label so a retry is idempotent")
	assert.Equal(t, int64(1), gk.metrics.TotalLoaded.Load())
}

func TestWriteBatchDeleteWinsOverEarlierUpsert(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oms/orm_order/_stream_load" {
			w.Write([]byte(`{"Status":"Success"}`))
			return
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"Status":"Success"}`))
	}))
	defer ts.Close()

	gk := &StreamingGatekeeper{
		cfg: Config{
			StarRocksHTTP:  ts.URL,
			StarRocksDB:    "oms",
			StarRocksTable: "orm_order",
			Topic:          "orm_oms.orm.order",
			PrimaryKeys:    []string{"id"},
		},
		metrics: &GatekeeperMetrics{},
	}

	// created then deleted in one batch: the row must not be loaded at all.
	b := &batch{
		ops: []pendingOp{
			{key: "k1", row: json.RawMessage(`{"id":"k1","v":1}`)},
			{key: "k1", isDelete: true},
		},
	}

	// With no query DSN the delete cannot be applied, and the error must name the
	// delete path — proving the earlier upsert was suppressed rather than loaded.
	err := gk.writeBatch(context.Background(), b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete 1 rows")
	assert.Empty(t, gotBody, "a key deleted later in the batch must not be upserted")
	assert.Equal(t, int64(0), gk.metrics.TotalLoaded.Load())
}

func TestDeleteSQL(t *testing.T) {
	cfg := Config{StarRocksDB: "oms", StarRocksTable: "orm_order"}

	q, ok := deleteSQL(cfg, []string{"id"}, []string{"aaa", "bbb"})
	require.True(t, ok)
	assert.Equal(t, "DELETE FROM `oms`.`orm_order` WHERE (`id` = 'aaa') OR (`id` = 'bbb')", q)

	// A composite key must AND its columns, never collapse to one.
	q, ok = deleteSQL(cfg, []string{"tenant_id", "id"}, []string{"t-1\x1fk-1"})
	require.True(t, ok)
	assert.Equal(t, "DELETE FROM `oms`.`orm_order` WHERE (`tenant_id` = 't-1' AND `id` = 'k-1')", q)

	// A key that does not match the configured columns is dropped rather than
	// producing a partial predicate that would over-delete.
	_, ok = deleteSQL(cfg, []string{"id"}, []string{"a\x1fb"})
	assert.False(t, ok)

	_, ok = deleteSQL(cfg, []string{"id"}, nil)
	assert.False(t, ok)
}

func TestHostOf(t *testing.T) {
	assert.Equal(t, "starrocks-fe", hostOf("http://starrocks-fe:8030"))
	assert.Equal(t, "100.84.50.65", hostOf("http://100.84.50.65:8030"))
	assert.Equal(t, "starrocks-fe", hostOf("https://starrocks-fe/"))
	assert.Equal(t, "", hostOf(""))
}

func TestStreamLoadBatchTreatsDuplicateLabelAsSuccess(t *testing.T) {
	// StarRocks answers a replayed label with 200/"Label Already Exists". Treating
	// that as success is what makes an at-least-once redelivery safe.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Status":"Label Already Exists","Message":"label already used"}`))
	}))
	defer ts.Close()

	cfg := Config{StarRocksHTTP: ts.URL, StarRocksDB: "oms", StarRocksTable: "orm_order"}
	err := streamLoadBatch(context.Background(), cfg, []json.RawMessage{json.RawMessage(`{"id":"1"}`)}, "lbl-1")
	assert.NoError(t, err)
}

func TestStreamLoadBatchSurfacesRejectedLoad(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Status":"Fail","Message":"too many filtered rows"}`))
	}))
	defer ts.Close()

	cfg := Config{StarRocksHTTP: ts.URL, StarRocksDB: "oms", StarRocksTable: "orm_order"}
	err := streamLoadBatch(context.Background(), cfg, []json.RawMessage{json.RawMessage(`{"id":"1"}`)}, "lbl-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many filtered rows")
}

func TestQuoteLiteral(t *testing.T) {
	assert.Equal(t, "'abc-123'", quoteLiteral("abc-123"))
	// A value carrying a quote must not be able to terminate the literal.
	assert.Equal(t, "'a''b'", quoteLiteral("a'b"))
	// A newline cannot be smuggled into the statement.
	assert.Equal(t, "NULL", quoteLiteral("a\nb"))
}

func TestGatekeeperStats(t *testing.T) {
	gk := &StreamingGatekeeper{
		cfg: Config{
			Topic:             "orm_oms.orm.order",
			StarRocksTable:    "orm_order",
			AssignedTenantID:  "910638ba-a459-4a3f-bb2d-78391b0595f6",
			ValidationEnabled: true,
		},
		metrics: &GatekeeperMetrics{},
	}

	gk.metrics.TotalConsumed.Add(10)
	gk.metrics.TotalLoaded.Add(8)
	gk.metrics.TenantMismatches.Add(1)
	gk.metrics.RuleViolationsBlocked.Add(1)

	stats := gk.GetStats()
	assert.Equal(t, int64(10), stats.TotalConsumed)
	assert.Equal(t, int64(8), stats.TotalLoaded)
	assert.Equal(t, int64(1), stats.TenantMismatches)
	assert.Equal(t, int64(1), stats.RuleViolationsBlocked)
	assert.Equal(t, "orm_oms.orm.order", stats.Topic)
	assert.Equal(t, "orm_order", stats.StarRocksTable)
	assert.True(t, stats.ValidationEnabled)
}

func TestSanitizeLabel(t *testing.T) {
	// StarRocks validates labels against ^[-\w]{1,128}$: a dotted topic name must not
	// keep its dots or every load is rejected as "Label format error".
	got := sanitizeLabel("orm_oms.orm.order")
	assert.Equal(t, "orm_oms_orm_order", got)
	for _, r := range sanitizeLabel("orm_oms.orm.order_4_4") {
		assert.True(t, r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'),
			"label contains a character StarRocks rejects: %q", r)
	}
	assert.NotContains(t, sanitizeLabel("a b/c"), " ")
	assert.LessOrEqual(t, len(sanitizeLabel(strings.Repeat("x", 500))), 100)
}

func TestParsePositiveInt(t *testing.T) {
	assert.Equal(t, 2000, parsePositiveInt("2000"))
	assert.Equal(t, 0, parsePositiveInt("abc"))
	assert.Equal(t, 0, parsePositiveInt("-5"))
	assert.Equal(t, 0, parsePositiveInt(""))
}
