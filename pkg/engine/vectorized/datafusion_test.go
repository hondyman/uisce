package vectorized

import (
	"context"
	"testing"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataFusionEngine_ProviderRegistration(t *testing.T) {
	mem := memory.NewGoAllocator()
	engine, err := NewDataFusionEngine(mem)
	require.NoError(t, err)

	dummyPtr := unsafe.Pointer(engine)

	// Valid registration
	err = engine.RegisterFFITableProvider("oms_account", dummyPtr, SupportedDataFusionVersion)
	assert.NoError(t, err)
	assert.True(t, engine.HasProvider("oms_account"))

	// Version mismatch
	err = engine.RegisterFFITableProvider("oms_position", dummyPtr, "1.0.0")
	assert.ErrorIs(t, err, ErrVersionMismatch)

	// Nil pointer
	err = engine.RegisterFFITableProvider("oms_position", nil, SupportedDataFusionVersion)
	assert.ErrorIs(t, err, ErrNilPointer)

	// Unregister
	engine.UnregisterFFITableProvider("oms_account")
	assert.False(t, engine.HasProvider("oms_account"))
}

func TestDataFusionEngine_ArrowCDataRoundTrip(t *testing.T) {
	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "account_id", Type: arrow.PrimitiveTypes.Int64},
		{Name: "nav", Type: arrow.PrimitiveTypes.Float64},
	}, nil)

	builder := array.NewRecordBuilder(mem, schema)
	defer builder.Release()

	builder.Field(0).(*array.Int64Builder).AppendValues([]int64{101, 102, 103}, nil)
	builder.Field(1).(*array.Float64Builder).AppendValues([]float64{1000000.0, 2500000.0, 500000.0}, nil)

	rec := builder.NewRecord()
	defer rec.Release()

	cArr, cSchema, err := ExportRecordBatchToC(rec)
	require.NoError(t, err)
	require.NotNil(t, cArr)
	require.NotNil(t, cSchema)

	importedRec, err := ImportRecordBatchFromC(cArr, cSchema)
	require.NoError(t, err)
	defer importedRec.Release()

	assert.Equal(t, int64(3), importedRec.NumRows())
	assert.Equal(t, int64(2), importedRec.NumCols())
}

func TestDataFusionEngine_ExecuteVectorizedQuery(t *testing.T) {
	mem := memory.NewGoAllocator()
	engine, err := NewDataFusionEngine(mem)
	require.NoError(t, err)

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "account_id", Type: arrow.PrimitiveTypes.Int64},
		{Name: "nav", Type: arrow.PrimitiveTypes.Float64},
	}, nil)

	builder := array.NewRecordBuilder(mem, schema)
	defer builder.Release()

	builder.Field(0).(*array.Int64Builder).AppendValues([]int64{1, 2}, nil)
	builder.Field(1).(*array.Float64Builder).AppendValues([]float64{100.0, 200.0}, nil)

	rec := builder.NewRecord()

	holder := RecordBatchHolder{
		Schema: schema,
		Record: rec,
	}
	defer holder.Release()

	ctx := context.Background()
	results, err := engine.ExecuteVectorizedQuery(ctx, "SELECT * FROM oms_account", holder)
	require.NoError(t, err)
	require.Len(t, results, 1)
	defer results[0].Release()

	assert.Equal(t, int64(2), results[0].NumRows())

	// Test context cancellation
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = engine.ExecuteVectorizedQuery(cancelCtx, "SELECT * FROM oms_account", holder)
	assert.ErrorIs(t, err, ErrExecutionCanceled)
}
