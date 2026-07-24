package models

// SemanticTypeCategory represents the category of a semantic type
type SemanticTypeCategory string

const (
	SemanticTypeCategoryDimension SemanticTypeCategory = "Dimension"
	SemanticTypeCategoryMeasure   SemanticTypeCategory = "Measure"
	SemanticTypeCategoryTime      SemanticTypeCategory = "Time"
	SemanticTypeCategoryHierarchy SemanticTypeCategory = "Hierarchy"
)

// DataType represents the underlying data type of a semantic type
type DataType string

const (
	DataTypeString              DataType = "string"
	DataTypeNumber              DataType = "number"
	DataTypeBoolean             DataType = "boolean"
	DataTypeTime                DataType = "time"
	DataTypeGeo                 DataType = "geo"
	DataTypeNumberAgg           DataType = "number_agg"
	DataTypeCount               DataType = "count"
	DataTypeCountDistinct       DataType = "count_distinct"
	DataTypeCountDistinctApprox DataType = "count_distinct_approx"
	DataTypeSum                 DataType = "sum"
	DataTypeAvg                 DataType = "avg"
	DataTypeMin                 DataType = "min"
	DataTypeMax                 DataType = "max"
)

// Format represents the display format of a semantic type
type Format string

const (
	FormatDefault  Format = "default"
	FormatImageUrl Format = "imageUrl"
	FormatLink     Format = "link"
	FormatCurrency Format = "currency"
	FormatPercent  Format = "percent"
	FormatId       Format = "id"
)

// SemanticTypeValue represents the unique identifier of a semantic type combination
type SemanticTypeValue string

// Dimension types
const (
	DimensionStringDefault  SemanticTypeValue = "dimension_string_default"
	DimensionStringImageUrl SemanticTypeValue = "dimension_string_imageurl"
	DimensionStringLink     SemanticTypeValue = "dimension_string_link"
	DimensionStringCurrency SemanticTypeValue = "dimension_string_currency"
	DimensionStringPercent  SemanticTypeValue = "dimension_string_percent"

	DimensionNumberDefault  SemanticTypeValue = "dimension_number_default"
	DimensionNumberId       SemanticTypeValue = "dimension_number_id"
	DimensionNumberCurrency SemanticTypeValue = "dimension_number_currency"
	DimensionNumberPercent  SemanticTypeValue = "dimension_number_percent"

	DimensionBooleanDefault SemanticTypeValue = "dimension_boolean_default"
	DimensionTimeDefault    SemanticTypeValue = "dimension_time_default"
	DimensionGeoDefault     SemanticTypeValue = "dimension_geo_default"
)

// Measure types
const (
	MeasureStringDefault  SemanticTypeValue = "measure_string_default"
	MeasureTimeDefault    SemanticTypeValue = "measure_time_default"
	MeasureBooleanDefault SemanticTypeValue = "measure_boolean_default"

	MeasureNumberDefault  SemanticTypeValue = "measure_number_default"
	MeasureNumberPercent  SemanticTypeValue = "measure_number_percent"
	MeasureNumberCurrency SemanticTypeValue = "measure_number_currency"

	MeasureNumberAggDefault  SemanticTypeValue = "measure_number_agg_default"
	MeasureNumberAggPercent  SemanticTypeValue = "measure_number_agg_percent"
	MeasureNumberAggCurrency SemanticTypeValue = "measure_number_agg_currency"

	MeasureCountDefault               SemanticTypeValue = "measure_count_default"
	MeasureCountDistinctDefault       SemanticTypeValue = "measure_count_distinct_default"
	MeasureCountDistinctApproxDefault SemanticTypeValue = "measure_count_distinct_approx_default"

	MeasureSumDefault  SemanticTypeValue = "measure_sum_default"
	MeasureSumCurrency SemanticTypeValue = "measure_sum_currency"

	MeasureAvgDefault SemanticTypeValue = "measure_avg_default"
	MeasureMinDefault SemanticTypeValue = "measure_min_default"
	MeasureMaxDefault SemanticTypeValue = "measure_max_default"
)

// Time type
const (
	TimeTimeDefault SemanticTypeValue = "time_time_default"
)

// SemanticTypeMetadata represents the metadata stored in the JSONB column
type SemanticTypeMetadata struct {
	SemanticType SemanticTypeCategory `json:"semantic_type"`
	DataType     DataType             `json:"data_type"`
	Format       Format               `json:"format"`
	Notes        string               `json:"notes"`
}

// SemanticTypeLookupValue represents a semantic type entry from the lookup_values table
type SemanticTypeLookupValue struct {
	ID        string               `json:"id"`
	LookupID  string               `json:"lookup_id"`
	TenantID  string               `json:"tenant_id"`
	Value     SemanticTypeValue    `json:"value"`
	Label     string               `json:"label"`
	Metadata  SemanticTypeMetadata `json:"metadata"`
	ParentID  *string              `json:"parent_id,omitempty"`
	CreatedAt *string              `json:"created_at,omitempty"`
}

// SemanticTypeLookup represents the lookup entry
type SemanticTypeLookup struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	Name        string  `json:"name"` // Always "semantic_types"
	Description string  `json:"description"`
	CreatedAt   *string `json:"created_at,omitempty"`
	UpdatedAt   *string `json:"updated_at,omitempty"`
}

// Helper functions

// IsDimension checks if a semantic type is a dimension
func IsDimension(v SemanticTypeValue) bool {
	dimensionTypes := []SemanticTypeValue{
		DimensionStringDefault, DimensionStringImageUrl, DimensionStringLink,
		DimensionStringCurrency, DimensionStringPercent,
		DimensionNumberDefault, DimensionNumberId, DimensionNumberCurrency,
		DimensionNumberPercent, DimensionBooleanDefault,
		DimensionTimeDefault, DimensionGeoDefault,
	}
	for _, dt := range dimensionTypes {
		if v == dt {
			return true
		}
	}
	return false
}

// IsMeasure checks if a semantic type is a measure
func IsMeasure(v SemanticTypeValue) bool {
	measureTypes := []SemanticTypeValue{
		MeasureStringDefault, MeasureTimeDefault, MeasureBooleanDefault,
		MeasureNumberDefault, MeasureNumberPercent, MeasureNumberCurrency,
		MeasureNumberAggDefault, MeasureNumberAggPercent, MeasureNumberAggCurrency,
		MeasureCountDefault, MeasureCountDistinctDefault, MeasureCountDistinctApproxDefault,
		MeasureSumDefault, MeasureSumCurrency,
		MeasureAvgDefault, MeasureMinDefault, MeasureMaxDefault,
	}
	for _, mt := range measureTypes {
		if v == mt {
			return true
		}
	}
	return false
}

// IsTimeType checks if a semantic type is a time type
func IsTimeType(v SemanticTypeValue) bool {
	return v == TimeTimeDefault
}

// GetCategory returns the category of a semantic type
func GetCategory(v SemanticTypeValue) SemanticTypeCategory {
	if IsDimension(v) {
		return SemanticTypeCategoryDimension
	}
	if IsMeasure(v) {
		return SemanticTypeCategoryMeasure
	}
	if IsTimeType(v) {
		return SemanticTypeCategoryTime
	}
	return ""
}

// GetMetadata returns the metadata for a semantic type
// Note: In real usage, this would come from the database, but here are the definitions
func GetMetadata(v SemanticTypeValue) *SemanticTypeMetadata {
	metadataMap := map[SemanticTypeValue]SemanticTypeMetadata{
		// Dimensions - String
		DimensionStringDefault: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeString,
			Format:       FormatDefault,
			Notes:        "",
		},
		DimensionStringImageUrl: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeString,
			Format:       FormatImageUrl,
			Notes:        "Dimension Format",
		},
		DimensionStringLink: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeString,
			Format:       FormatLink,
			Notes:        "Dimension Format",
		},
		DimensionStringCurrency: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeString,
			Format:       FormatCurrency,
			Notes:        "Dimension Format (If underlying type is number and formatted as string in SQL)",
		},
		DimensionStringPercent: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeString,
			Format:       FormatPercent,
			Notes:        "Dimension Format (If underlying type is number and formatted as string in SQL)",
		},

		// Dimensions - Number
		DimensionNumberDefault: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeNumber,
			Format:       FormatDefault,
			Notes:        "",
		},
		DimensionNumberId: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeNumber,
			Format:       FormatId,
			Notes:        "Dimension Format",
		},
		DimensionNumberCurrency: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeNumber,
			Format:       FormatCurrency,
			Notes:        "Dimension Format",
		},
		DimensionNumberPercent: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeNumber,
			Format:       FormatPercent,
			Notes:        "Dimension Format",
		},

		// Dimensions - Boolean
		DimensionBooleanDefault: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeBoolean,
			Format:       FormatDefault,
			Notes:        "",
		},

		// Dimensions - Time
		DimensionTimeDefault: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeTime,
			Format:       FormatDefault,
			Notes:        "",
		},

		// Dimensions - Geo
		DimensionGeoDefault: {
			SemanticType: SemanticTypeCategoryDimension,
			DataType:     DataTypeGeo,
			Format:       FormatDefault,
			Notes:        "",
		},

		// Measures - Simple
		MeasureStringDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeString,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureTimeDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeTime,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureBooleanDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeBoolean,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},

		// Measures - Number
		MeasureNumberDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumber,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureNumberPercent: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumber,
			Format:       FormatPercent,
			Notes:        "Measure Format",
		},
		MeasureNumberCurrency: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumber,
			Format:       FormatCurrency,
			Notes:        "Measure Format",
		},

		// Measures - Aggregations
		MeasureNumberAggDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumberAgg,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureNumberAggPercent: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumberAgg,
			Format:       FormatPercent,
			Notes:        "Measure Format",
		},
		MeasureNumberAggCurrency: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeNumberAgg,
			Format:       FormatCurrency,
			Notes:        "Measure Format",
		},

		MeasureCountDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeCount,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureCountDistinctDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeCountDistinct,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureCountDistinctApproxDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeCountDistinctApprox,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},

		MeasureSumDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeSum,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureSumCurrency: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeSum,
			Format:       FormatCurrency,
			Notes:        "Measure Format",
		},

		MeasureAvgDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeAvg,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureMinDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeMin,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},
		MeasureMaxDefault: {
			SemanticType: SemanticTypeCategoryMeasure,
			DataType:     DataTypeMax,
			Format:       FormatDefault,
			Notes:        "Measure Type",
		},

		// Time
		TimeTimeDefault: {
			SemanticType: SemanticTypeCategoryTime,
			DataType:     DataTypeTime,
			Format:       FormatDefault,
			Notes:        "Dedicated Semantic Time Object",
		},
	}

	if metadata, ok := metadataMap[v]; ok {
		return &metadata
	}
	return nil
}
