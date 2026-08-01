package flight

import (
	"regexp"
	"strings"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

var piiRegex = regexp.MustCompile(`^(.{0,4}).*(.{4})$`)

func MaskStringVector(mem memory.Allocator, arr *array.String, roleExempt bool) *array.String {
	if roleExempt {
		return arr
	}

	builder := array.NewStringBuilder(mem)
	defer builder.Release()

	for i := 0; i < arr.Len(); i++ {
		if arr.IsNull(i) {
			builder.AppendNull()
			continue
		}

		rawVal := arr.Value(i)
		masked := MaskPII(rawVal)
		builder.Append(masked)
	}

	return builder.NewStringArray()
}

func MaskPII(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	matches := piiRegex.FindStringSubmatch(value)
	if len(matches) < 3 {
		return strings.Repeat("*", len(value))
	}
	return matches[1] + "****" + matches[2]
}

func MaskNumericVector(mem memory.Allocator, arr *array.Int64, fractionDigits int, roleExempt bool) *array.Int64Builder {
	if roleExempt {
		return nil
	}

	builder := array.NewInt64Builder(mem)
	defer builder.Release()

	divisor := int64(1)
	for i := 0; i < fractionDigits; i++ {
		divisor *= 10
	}

	for i := 0; i < arr.Len(); i++ {
		if arr.IsNull(i) {
			builder.AppendNull()
			continue
		}
		original := arr.Value(i)
		masked := (original / divisor) * divisor
		builder.Append(masked)
	}

	return builder
}

func MaskFloatVector(mem memory.Allocator, arr *array.Float64, fractionDigits int, roleExempt bool) *array.Float64Builder {
	if roleExempt {
		return nil
	}

	builder := array.NewFloat64Builder(mem)
	defer builder.Release()

	divisor := float64(1)
	for i := 0; i < fractionDigits; i++ {
		divisor *= 10
	}

	for i := 0; i < arr.Len(); i++ {
		if arr.IsNull(i) {
			builder.AppendNull()
			continue
		}
		original := arr.Value(i)
		masked := float64(int(original/divisor)) * divisor
		builder.Append(masked)
	}

	return builder
}
