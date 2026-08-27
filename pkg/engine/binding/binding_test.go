package binding

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessObjectBinding_ProjectToRecordBatch(t *testing.T) {
	mem := memory.NewGoAllocator()

	customFields := map[string]string{
		"hurdle_rate_pct": "float64",
		"strike_price":    "float64",
		"sponsor_name":    "string",
		"is_esg_compliant": "bool",
	}

	b, err := NewBusinessObjectBinding("oms.account", "oms.account", customFields)
	require.NoError(t, err)

	records := []DynamicRecord{
		{
			EntityID:    "E101",
			SubtypeCode: "ALT_INVESTMENT",
			BaseAmount:  1000000.0,
			TenantID:    "tenant-1",
			CustomValues: map[string]interface{}{
				"hurdle_rate_pct":  0.08,
				"sponsor_name":     "Blackstone",
				"is_esg_compliant": true,
			},
		},
		{
			EntityID:    "E102",
			SubtypeCode: "EQUITY_OPTION",
			BaseAmount:  250000.0,
			TenantID:    "tenant-1",
			CustomValues: map[string]interface{}{
				"strike_price":     150.0,
				"is_esg_compliant": false,
			},
		},
	}

	batch, err := b.ProjectToRecordBatch(mem, records)
	require.NoError(t, err)
	defer batch.Release()

	assert.Equal(t, int64(2), batch.NumRows())
	assert.Equal(t, int64(8), batch.NumCols()) // 4 base + 4 custom

	// Verify column values
	hurdleIdx, ok := b.ColumnIndex("hurdle_rate_pct")
	assert.True(t, ok)
	hurdleCol := batch.Column(hurdleIdx).(*array.Float64)
	assert.True(t, hurdleCol.IsValid(0))
	assert.Equal(t, 0.08, hurdleCol.Value(0))
	assert.False(t, hurdleCol.IsValid(1)) // NULL for EQUITY_OPTION

	strikeIdx, ok := b.ColumnIndex("strike_price")
	assert.True(t, ok)
	strikeCol := batch.Column(strikeIdx).(*array.Float64)
	assert.False(t, strikeCol.IsValid(0)) // NULL for ALT_INVESTMENT
	assert.True(t, strikeCol.IsValid(1))
	assert.Equal(t, 150.0, strikeCol.Value(1))
}
