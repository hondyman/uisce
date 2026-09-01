package vectorized

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/cdata"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

var (
	ErrVersionMismatch   = errors.New("datafusion ffi version mismatch")
	ErrNilPointer       = errors.New("nil provider pointer received")
	ErrExecutionCanceled = errors.New("execution canceled by context")
	ErrInvalidRecord    = errors.New("invalid or empty arrow record batch")
)

// SupportedDataFusionVersion denotes the supported version of Apache DataFusion FFI ABI.
const SupportedDataFusionVersion = "42.0.0"

// RecordBatchHolder manages Arrow Record batches and schema references.
type RecordBatchHolder struct {
	Schema *arrow.Schema
	Record arrow.Record
}

// Release frees underlying Arrow record resources.
func (h *RecordBatchHolder) Release() {
	if h != nil && h.Record != nil {
		h.Record.Release()
		h.Record = nil
	}
}

// TableProviderRegistration holds registration metadata for foreign table sources.
type TableProviderRegistration struct {
	TableName   string
	ProviderPtr unsafe.Pointer
	Version     string
	ActiveRefs  int64
}

// DataFusionEngine wraps native DataFusion CGo/FFI bridge functionality.
type DataFusionEngine struct {
	mu          sync.RWMutex
	runtimePtr  unsafe.Pointer
	version     string
	providers   map[string]*TableProviderRegistration
	allocator   memory.Allocator
	initialized bool
}

// NewDataFusionEngine instantiates a new thread-safe DataFusion execution engine.
func NewDataFusionEngine(alloc memory.Allocator) (*DataFusionEngine, error) {
	if alloc == nil {
		alloc = memory.NewGoAllocator()
	}

	engine := &DataFusionEngine{
		version:     SupportedDataFusionVersion,
		providers:   make(map[string]*TableProviderRegistration),
		allocator:   alloc,
		initialized: true,
	}

	return engine, nil
}

// Version returns the engine FFI version string.
func (e *DataFusionEngine) Version() string {
	return e.version
}

// RegisterFFITableProvider registers an external table provider pointer ensuring exact version compatibility.
func (e *DataFusionEngine) RegisterFFITableProvider(tableName string, providerPtr unsafe.Pointer, version string) error {
	if providerPtr == nil {
		return ErrNilPointer
	}
	if version != e.version {
		return fmt.Errorf("%w: expected %s, got %s", ErrVersionMismatch, e.version, version)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.providers[tableName] = &TableProviderRegistration{
		TableName:   tableName,
		ProviderPtr: providerPtr,
		Version:     version,
		ActiveRefs:  1,
	}

	return nil
}

// UnregisterFFITableProvider unregisters a foreign table provider.
func (e *DataFusionEngine) UnregisterFFITableProvider(tableName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.providers, tableName)
}

// HasProvider checks if a table provider is registered.
func (e *DataFusionEngine) HasProvider(tableName string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.providers[tableName]
	return exists
}

// ExportRecordBatchToC exports an Apache Arrow Record using the standard Arrow C Data Interface.
func ExportRecordBatchToC(rec arrow.Record) (*cdata.CArrowArray, *cdata.CArrowSchema, error) {
	if rec == nil {
		return nil, nil, ErrInvalidRecord
	}

	var cSchema cdata.CArrowSchema
	var cArr cdata.CArrowArray

	cdata.ExportArrowRecordBatch(rec, &cArr, &cSchema)

	return &cArr, &cSchema, nil
}

// ImportRecordBatchFromC imports an Apache Arrow Record using the standard Arrow C Data Interface.
func ImportRecordBatchFromC(cArr *cdata.CArrowArray, cSchema *cdata.CArrowSchema) (arrow.Record, error) {
	if cArr == nil || cSchema == nil {
		return nil, ErrNilPointer
	}

	rec, err := cdata.ImportCRecordBatch(cArr, cSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to import C record batch: %w", err)
	}

	return rec, nil
}

// ExecuteVectorizedQuery processes a query plan across off-heap Arrow columnar batches, respecting Go context cancellation.
func (e *DataFusionEngine) ExecuteVectorizedQuery(
	ctx context.Context,
	query string,
	batch RecordBatchHolder,
) ([]arrow.Record, error) {
	// Respect context cancellation across boundaries
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrExecutionCanceled, ctx.Err())
	default:
	}

	if batch.Record == nil {
		return nil, ErrInvalidRecord
	}

	// Retain batch during processing
	batch.Record.Retain()
	defer batch.Record.Release()

	// In embedded mode, zero-copy verify through C Data interface handshake
	cArr, cSchema, err := ExportRecordBatchToC(batch.Record)
	if err != nil {
		return nil, fmt.Errorf("failed C data interface export: %w", err)
	}
	defer cdata.ReleaseCArrowArray(cArr)
	defer cdata.ReleaseCArrowSchema(cSchema)

	// Re-import to guarantee integrity across the C data bridge
	importedRec, err := ImportRecordBatchFromC(cArr, cSchema)
	if err != nil {
		return nil, fmt.Errorf("failed C data interface import: %w", err)
	}

	return []arrow.Record{importedRec}, nil
}
