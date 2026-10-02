package main

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestDecodeRecord(t *testing.T) {
	t.Run("Valid record with decimal encoding", func(t *testing.T) {
		val := big.NewInt(450000) // 45.0000 with scale 4
		b64Price := base64.StdEncoding.EncodeToString(val.Bytes())

		sample := map[string]interface{}{
			"schema": map[string]interface{}{
				"type": "struct",
				"fields": []map[string]interface{}{
					{
						"type":  "struct",
						"field": "after",
						"fields": []map[string]interface{}{
							{
								"field": "price",
								"name":  "org.apache.kafka.connect.data.Decimal",
								"parameters": map[string]string{
									"scale": "4",
								},
							},
						},
					},
				},
			},
			"payload": map[string]interface{}{
				"op": "c",
				"after": map[string]interface{}{
					"id":        "ord-1",
					"tenant_id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
					"price":     b64Price,
					"quantity":  1000,
				},
			},
		}

		raw, err := json.Marshal(sample)
		require.NoError(t, err)

		rowJSON, afterMap, skip, err := decodeRecord(raw)
		require.NoError(t, err)
		assert.False(t, skip)
		assert.NotNil(t, rowJSON)
		assert.Equal(t, "45.0000", afterMap["price"])
		assert.Equal(t, "ord-1", afterMap["id"])
		assert.Equal(t, "910638ba-a459-4a3f-bb2d-78391b0595f6", extractTenantID(afterMap))
	})

	t.Run("Tombstone record skips", func(t *testing.T) {
		sample := map[string]interface{}{
			"payload": map[string]interface{}{
				"op":    "d",
				"after": nil,
			},
		}
		raw, err := json.Marshal(sample)
		require.NoError(t, err)

		rowJSON, afterMap, skip, err := decodeRecord(raw)
		require.NoError(t, err)
		assert.True(t, skip)
		assert.Nil(t, rowJSON)
		assert.Nil(t, afterMap)
	})

	t.Run("Malformed JSON returns error", func(t *testing.T) {
		_, _, _, err := decodeRecord([]byte("invalid json"))
		assert.Error(t, err)
	})
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

func TestStreamLoadErrorOnInvalidEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg := Config{
		StarRocksHTTP:  ts.URL,
		StarRocksDB:    "test_db",
		StarRocksTable: "test_table",
	}

	err := streamLoad(cfg, []byte(`{"id": 1}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad status")
}
