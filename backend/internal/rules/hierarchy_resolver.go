// backend/internal/rules/hierarchy_resolver.go

package rules

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// ============================================================================
// TYPES
// ============================================================================

type HierarchyPath struct {
	Segments []string `json:"segments"`  // ["line_items", "product"]
	FullPath string   `json:"full_path"` // "order.line_items.product"
	Depth    int      `json:"depth"`
	IsArray  bool     `json:"is_array"`
}

type AggregationType string

const (
	AggregationSum   AggregationType = "sum"
	AggregationCount AggregationType = "count"
	AggregationAvg   AggregationType = "avg"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
)

type HierarchyResolver struct {
	// Caching or schema registry can be added here for performance
}

// ============================================================================
// RESOLVER IMPLEMENTATION
// ============================================================================

func NewHierarchyResolver() *HierarchyResolver {
	return &HierarchyResolver{}
}

// ResolveFieldPath resolves a field through a hierarchy path for a single value.
// Example: resolveFieldPath(orderData, "total")
func (hr *HierarchyResolver) ResolveFieldPath(
	data interface{},
	path string,
) (interface{}, bool) {
	segments := strings.Split(path, ".")
	current := data

	for _, segment := range segments {
		current = hr.navigateSegment(current, segment)
		if current == nil {
			return nil, false
		}
	}

	return current, true
}

// ResolveFieldPathArray resolves a path that may traverse arrays/slices,
// returning all matching values from the leaf nodes.
// Example: resolveFieldPathArray(orderData, "line_items.price") -> [25, 15]
func (hr *HierarchyResolver) ResolveFieldPathArray(
	data interface{},
	path string,
) ([]interface{}, bool) {
	segments := strings.Split(path, ".")
	results := []interface{}{data}

	for _, segment := range segments {
		newResults := []interface{}{}
		for _, current := range results {
			if hr.isArray(current) {
				arrayVals := hr.toArray(current)
				for _, item := range arrayVals {
					navigated := hr.navigateSegment(item, segment)
					if navigated != nil {
						newResults = append(newResults, navigated)
					}
				}
			} else {
				navigated := hr.navigateSegment(current, segment)
				if navigated != nil {
					newResults = append(newResults, navigated)
				}
			}
		}

		if len(newResults) == 0 {
			return nil, false
		}
		results = newResults
	}

	// The final result might contain nested arrays if the last segment pointed to an array.
	// We flatten it to get all individual values.
	finalValues := []interface{}{}
	for _, res := range results {
		if hr.isArray(res) {
			finalValues = append(finalValues, hr.toArray(res)...)
		} else {
			finalValues = append(finalValues, res)
		}
	}

	return finalValues, len(finalValues) > 0
}

// ResolveWithAggregation resolves a path and applies an aggregation function.
func (hr *HierarchyResolver) ResolveWithAggregation(
	data interface{},
	path string,
	aggregation AggregationType,
	field string,
) (interface{}, bool) {
	// First, resolve the path to get the array of objects.
	values, ok := hr.ResolveFieldPathArray(data, path)
	if !ok {
		return nil, false
	}

	// Then, aggregate a specific field from those objects.
	return hr.Aggregate(values, field, aggregation)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (hr *HierarchyResolver) navigateSegment(
	current interface{},
	segment string,
) interface{} {
	// Handle map navigation (most common for JSON data)
	if m, ok := current.(map[string]interface{}); ok {
		if val, exists := m[segment]; exists {
			return val
		}
		return nil
	}

	// Handle struct navigation via reflection
	v := reflect.ValueOf(current)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		// Try to find a field with the exact name
		field := v.FieldByName(segment)
		if field.IsValid() {
			return field.Interface()
		}
		// Try to find a field by JSON tag
		for i := 0; i < v.NumField(); i++ {
			tag := v.Type().Field(i).Tag.Get("json")
			if strings.Split(tag, ",")[0] == segment {
				return v.Field(i).Interface()
			}
		}
	}

	return nil
}

func (hr *HierarchyResolver) isArray(val interface{}) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Slice || reflect.TypeOf(val).Kind() == reflect.Array
}

func (hr *HierarchyResolver) toArray(val interface{}) []interface{} {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result
}

// Aggregate applies an aggregation function to a specific field across a slice of objects.
func (hr *HierarchyResolver) Aggregate(
	values []interface{},
	field string,
	aggregationType AggregationType,
) (interface{}, bool) {
	if len(values) == 0 {
		return nil, false
	}

	var numbers []float64
	for _, val := range values {
		// For each item in the resolved array, get the value of the aggregation field.
		fieldVal := hr.navigateSegment(val, field)

		if num, ok := hr.toNumber(fieldVal); ok {
			numbers = append(numbers, num)
		}
	}

	if len(numbers) == 0 {
		// This can happen if the field didn't exist or wasn't numeric.
		return nil, false
	}

	switch aggregationType {
	case AggregationSum:
		result := 0.0
		for _, n := range numbers {
			result += n
		}
		return result, true

	case AggregationCount:
		return float64(len(numbers)), true

	case AggregationAvg:
		sum := 0.0
		for _, n := range numbers {
			sum += n
		}
		return sum / float64(len(numbers)), true

	case AggregationMin:
		min := numbers[0]
		for _, n := range numbers {
			if n < min {
				min = n
			}
		}
		return min, true

	case AggregationMax:
		max := numbers[0]
		for _, n := range numbers {
			if n > max {
				max = n
			}
		}
		return max, true

	default:
		return nil, false
	}
}

func (hr *HierarchyResolver) toNumber(val interface{}) (float64, bool) {
	if val == nil {
		return 0, false
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.String:
		num, err := strconv.ParseFloat(v.String(), 64)
		return num, err == nil
	default:
		// Handle types like json.Number
		if num, ok := val.(json.Number); ok {
			f, err := num.Float64()
			return f, err == nil
		}
		return 0, false
	}
}

// GetPathDepth returns the depth of a path.
func GetPathDepth(path string) int {
	if path == "" {
		return 0
	}
	return len(strings.Split(path, "."))
}

// BuildHierarchyPath constructs a HierarchyPath from segments.
func BuildHierarchyPath(segments []string) HierarchyPath {
	return HierarchyPath{
		Segments: segments,
		FullPath: strings.Join(segments, "."),
		Depth:    len(segments),
		IsArray:  true, // Assume paths involving sub-entities are arrays
	}
}
