package binding

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

var (
	ErrNilBinding     = errors.New("business object binding is nil")
	ErrEmptyRecords   = errors.New("empty polymorphic records supplied")
	ErrUnknownType    = errors.New("unsupported arrow data type string")
)

// BusinessObjectBinding manages schema projections and dynamic STI attribute mapping to Arrow schemas.
type BusinessObjectBinding struct {
	EntityName   string            `json:"entity_name"`   // e.g. "oms.account", "altinv.alternative_investment"
	TableMapping string            `json:"table_mapping"` // Physical table name
	CustomFields map[string]string `json:"custom_fields"` // Field Name -> Arrow Type ("float64", "string", "int64", "bool")

	mu          sync.RWMutex
	arrowSchema *arrow.Schema
	fieldIndex  map[string]int
}

// NewBusinessObjectBinding creates a new dynamic binding configuration.
func NewBusinessObjectBinding(entityName, tableMapping string, customFields map[string]string) (*BusinessObjectBinding, error) {
	b := &BusinessObjectBinding{
		EntityName:   entityName,
		TableMapping: tableMapping,
		CustomFields: customFields,
		fieldIndex:   make(map[string]int),
	}

	if err := b.buildArrowSchema(); err != nil {
		return nil, err
	}

	return b, nil
}

// buildArrowSchema compiles the Arrow schema with standard STI columns plus dynamic client attributes.
func (b *BusinessObjectBinding) buildArrowSchema() error {
	fields := []arrow.Field{
		{Name: "entity_id", Type: arrow.BinaryTypes.String},
		{Name: "subtype_code", Type: arrow.BinaryTypes.String},
		{Name: "base_amount", Type: arrow.PrimitiveTypes.Float64},
		{Name: "tenant_id", Type: arrow.BinaryTypes.String},
	}

	for name, typeStr := range b.CustomFields {
		var dt arrow.DataType
		switch strings.ToLower(typeStr) {
		case "float64", "double", "numeric", "number":
			dt = arrow.PrimitiveTypes.Float64
		case "int64", "integer", "bigint":
			dt = arrow.PrimitiveTypes.Int64
		case "string", "text", "varchar":
			dt = arrow.BinaryTypes.String
		case "bool", "boolean":
			dt = arrow.FixedWidthTypes.Boolean
		default:
			return fmt.Errorf("%w: %s for field %s", ErrUnknownType, typeStr, name)
		}
		fields = append(fields, arrow.Field{Name: name, Type: dt, Nullable: true})
	}

	b.arrowSchema = arrow.NewSchema(fields, nil)
	for i, f := range fields {
		b.fieldIndex[f.Name] = i
	}

	return nil
}

// Schema returns the compiled Arrow schema.
func (b *BusinessObjectBinding) Schema() *arrow.Schema {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.arrowSchema
}

// ColumnIndex returns the offset position of a column in the projected Arrow schema.
func (b *BusinessObjectBinding) ColumnIndex(name string) (int, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	idx, found := b.fieldIndex[name]
	return idx, found
}

// DynamicRecord represents a row from an STI table containing static attributes and dynamic custom fields.
type DynamicRecord struct {
	EntityID     string
	SubtypeCode  string
	BaseAmount   float64
	TenantID     string
	CustomValues map[string]interface{}
}

// ProjectToRecordBatch projects an array of dynamic polymorphic rows directly into an off-heap Arrow RecordBatch.
func (b *BusinessObjectBinding) ProjectToRecordBatch(mem memory.Allocator, records []DynamicRecord) (arrow.Record, error) {
	if b == nil {
		return nil, ErrNilBinding
	}
	if len(records) == 0 {
		return nil, ErrEmptyRecords
	}
	if mem == nil {
		mem = memory.NewGoAllocator()
	}

	b.mu.RLock()
	schema := b.arrowSchema
	b.mu.RUnlock()

	builder := array.NewRecordBuilder(mem, schema)
	defer builder.Release()

	// Base columns
	entityIDBuilder := builder.Field(0).(*array.StringBuilder)
	subtypeBuilder := builder.Field(1).(*array.StringBuilder)
	baseAmountBuilder := builder.Field(2).(*array.Float64Builder)
	tenantIDBuilder := builder.Field(3).(*array.StringBuilder)

	for _, rec := range records {
		entityIDBuilder.Append(rec.EntityID)
		subtypeBuilder.Append(rec.SubtypeCode)
		baseAmountBuilder.Append(rec.BaseAmount)
		tenantIDBuilder.Append(rec.TenantID)
	}

	// Custom dynamic fields
	for fieldName, typeStr := range b.CustomFields {
		idx, ok := b.ColumnIndex(fieldName)
		if !ok {
			continue
		}

		switch strings.ToLower(typeStr) {
		case "float64", "double", "numeric", "number":
			fBuilder := builder.Field(idx).(*array.Float64Builder)
			for _, rec := range records {
				val, exists := rec.CustomValues[fieldName]
				if exists && val != nil {
					switch v := val.(type) {
					case float64:
						fBuilder.Append(v)
					case float32:
						fBuilder.Append(float64(v))
					case int:
						fBuilder.Append(float64(v))
					case int64:
						fBuilder.Append(float64(v))
					default:
						fBuilder.AppendNull()
					}
				} else {
					fBuilder.AppendNull()
				}
			}
		case "int64", "integer", "bigint":
			iBuilder := builder.Field(idx).(*array.Int64Builder)
			for _, rec := range records {
				val, exists := rec.CustomValues[fieldName]
				if exists && val != nil {
					switch v := val.(type) {
					case int64:
						iBuilder.Append(v)
					case int:
						iBuilder.Append(int64(v))
					case float64:
						iBuilder.Append(int64(v))
					default:
						iBuilder.AppendNull()
					}
				} else {
					iBuilder.AppendNull()
				}
			}
		case "string", "text", "varchar":
			sBuilder := builder.Field(idx).(*array.StringBuilder)
			for _, rec := range records {
				val, exists := rec.CustomValues[fieldName]
				if exists && val != nil {
					sBuilder.Append(fmt.Sprintf("%v", val))
				} else {
					sBuilder.AppendNull()
				}
			}
		case "bool", "boolean":
			bBuilder := builder.Field(idx).(*array.BooleanBuilder)
			for _, rec := range records {
				val, exists := rec.CustomValues[fieldName]
				if exists && val != nil {
					if bv, ok := val.(bool); ok {
						bBuilder.Append(bv)
					} else {
						bBuilder.AppendNull()
					}
				} else {
					bBuilder.AppendNull()
				}
			}
		}
	}

	return builder.NewRecord(), nil
}
