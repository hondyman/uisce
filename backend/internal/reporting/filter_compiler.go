package reporting

import (
	"fmt"
	"strings"
)

// CompileFilterModel converts a FilterModel into a SQL WHERE clause fragment.
// Returns an empty string if there are no active filters.
// params maps parameter names to their runtime values (used for substitution).
func CompileFilterModel(model *FilterModel, params map[string]interface{}, defaults *TenantDefaults) string {
	if model == nil || len(model.Groups) == 0 {
		return ""
	}
	var groupClauses []string
	for _, g := range model.Groups {
		clause := compileGroup(&g, params, defaults)
		if clause != "" {
			if len(model.Groups) == 1 {
				groupClauses = append(groupClauses, clause)
			} else {
				groupClauses = append(groupClauses, "("+clause+")")
			}
		}
	}
	if len(groupClauses) == 0 {
		return ""
	}
	combinator := model.GroupCombinator
	if combinator == "" {
		combinator = "AND"
	}
	return strings.Join(groupClauses, " "+combinator+" ")
}

func compileGroup(g *FilterGroup, params map[string]interface{}, defaults *TenantDefaults) string {
	var clauses []string
	for _, f := range g.Filters {
		if !f.Enabled {
			continue
		}
		c := compileFilter(&f, params, defaults)
		if c != "" {
			clauses = append(clauses, c)
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	combinator := g.Combinator
	if combinator == "" {
		combinator = "AND"
	}
	return strings.Join(clauses, " "+combinator+" ")
}

// compileFilter turns a single Filter into a SQL predicate string.
func compileFilter(f *Filter, params map[string]interface{}, defaults *TenantDefaults) string {
	fieldRef := quoteField(f.Field)
	vs := f.ValueSource

	switch vs.Kind {
	case ValueSourceCalendar:
		return compileCalendarOp(fieldRef, f.Operator, vs.CalendarCode, f.Values)
	case ValueSourceTenantDefault:
		cal := defaults.DefaultCalendarCode
		if cal == "" {
			cal = "US"
		}
		return compileCalendarOp(fieldRef, f.Operator, cal, f.Values)
	case ValueSourceInstanceDefault:
		cal := defaults.DefaultCalendarCode
		if cal == "" {
			cal = "US"
		}
		return compileCalendarOp(fieldRef, f.Operator, cal, f.Values)
	case ValueSourceConstant:
		return compileStandardOp(fieldRef, f.Operator, vs.Value, f.Values)
	case ValueSourceParameter:
		pname := vs.ParameterName
		if pname == "" {
			pname = vs.ParameterID
		}
		if !strings.HasPrefix(pname, "@") {
			pname = "@" + pname
		}
		return compileParamOp(fieldRef, f.Operator, pname, f.Values)
	case ValueSourceFunction:
		return compileFunctionOp(fieldRef, f.Operator, vs.Expression)
	default:
		return ""
	}
}

// compileCalendarOp handles calendar-aware operators.
func compileCalendarOp(fieldRef string, op FilterOperator, calCode string, values []string) string {
	switch op {
	case OpNextBusinessDay:
		return fmt.Sprintf("%s = calendar_next_business_day(%s, '%s')", fieldRef, fieldRef, calCode)
	case OpPreviousBusinessDay:
		return fmt.Sprintf("%s = calendar_previous_business_day(%s, '%s')", fieldRef, fieldRef, calCode)
	case OpAddBusinessDays:
		n := "1"
		if len(values) > 0 {
			n = values[0]
		}
		return fmt.Sprintf("%s = calendar_add_business_days(%s, '%s', %s)", fieldRef, fieldRef, calCode, n)
	case OpIsBusinessDay:
		return fmt.Sprintf("calendar_is_business_day(%s, '%s')", fieldRef, calCode)
	case OpIsHoliday:
		return fmt.Sprintf("calendar_is_holiday(%s, '%s')", fieldRef, calCode)
	case OpLastNBusinessDays:
		n := "5"
		if len(values) > 0 {
			n = values[0]
		}
		return fmt.Sprintf("%s >= calendar_add_business_days(CURRENT_DATE, '%s', -%s) AND %s <= CURRENT_DATE",
			fieldRef, calCode, n, fieldRef)
	case OpNextNBusinessDays:
		n := "5"
		if len(values) > 0 {
			n = values[0]
		}
		return fmt.Sprintf("%s >= CURRENT_DATE AND %s <= calendar_add_business_days(CURRENT_DATE, '%s', %s)",
			fieldRef, fieldRef, calCode, n)
	default:
		// Fall through to standard op for other operators
		return ""
	}
}

// compileStandardOp handles standard SQL operators with constant values.
func compileStandardOp(fieldRef string, op FilterOperator, value string, values []string) string {
	switch op {
	case OpEquals:
		return fmt.Sprintf("%s = %s", fieldRef, quoteValue(value))
	case OpNotEquals:
		return fmt.Sprintf("%s != %s", fieldRef, quoteValue(value))
	case OpGreaterThan:
		return fmt.Sprintf("%s > %s", fieldRef, quoteValue(value))
	case OpLessThan:
		return fmt.Sprintf("%s < %s", fieldRef, quoteValue(value))
	case OpGreaterEqual:
		return fmt.Sprintf("%s >= %s", fieldRef, quoteValue(value))
	case OpLessEqual:
		return fmt.Sprintf("%s <= %s", fieldRef, quoteValue(value))
	case OpBetween:
		if len(values) < 2 {
			return ""
		}
		return fmt.Sprintf("%s BETWEEN %s AND %s", fieldRef, quoteValue(values[0]), quoteValue(values[1]))
	case OpNotBetween:
		if len(values) < 2 {
			return ""
		}
		return fmt.Sprintf("%s NOT BETWEEN %s AND %s", fieldRef, quoteValue(values[0]), quoteValue(values[1]))
	case OpIn:
		if len(values) == 0 {
			return ""
		}
		quoted := make([]string, len(values))
		for i, v := range values {
			quoted[i] = quoteValue(v)
		}
		return fmt.Sprintf("%s IN (%s)", fieldRef, strings.Join(quoted, ", "))
	case OpNotIn:
		if len(values) == 0 {
			return ""
		}
		quoted := make([]string, len(values))
		for i, v := range values {
			quoted[i] = quoteValue(v)
		}
		return fmt.Sprintf("%s NOT IN (%s)", fieldRef, strings.Join(quoted, ", "))
	case OpIsNull:
		return fmt.Sprintf("%s IS NULL", fieldRef)
	case OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", fieldRef)
	case OpContains:
		return fmt.Sprintf("%s ILIKE '%%%s%%'", fieldRef, escapeLike(value))
	case OpStartsWith:
		return fmt.Sprintf("%s ILIKE '%s%%'", fieldRef, escapeLike(value))
	case OpEndsWith:
		return fmt.Sprintf("%s ILIKE '%%%s'", fieldRef, escapeLike(value))
	case OpToday:
		return fmt.Sprintf("%s = CURRENT_DATE", fieldRef)
	case OpYesterday:
		return fmt.Sprintf("%s = CURRENT_DATE - INTERVAL '1 day'", fieldRef)
	case OpTomorrow:
		return fmt.Sprintf("%s = CURRENT_DATE + INTERVAL '1 day'", fieldRef)
	case OpStartOfWeek:
		return fmt.Sprintf("%s >= date_trunc('week', CURRENT_DATE)", fieldRef)
	case OpEndOfWeek:
		return fmt.Sprintf("%s <= date_trunc('week', CURRENT_DATE) + INTERVAL '6 days'", fieldRef)
	case OpStartOfMonth:
		return fmt.Sprintf("%s >= date_trunc('month', CURRENT_DATE)", fieldRef)
	case OpEndOfMonth:
		return fmt.Sprintf("%s <= (date_trunc('month', CURRENT_DATE) + INTERVAL '1 month - 1 day')::date", fieldRef)
	case OpStartOfQuarter:
		return fmt.Sprintf("%s >= date_trunc('quarter', CURRENT_DATE)", fieldRef)
	case OpEndOfQuarter:
		return fmt.Sprintf("%s <= (date_trunc('quarter', CURRENT_DATE) + INTERVAL '3 months - 1 day')::date", fieldRef)
	case OpStartOfYear:
		return fmt.Sprintf("%s >= date_trunc('year', CURRENT_DATE)", fieldRef)
	case OpEndOfYear:
		return fmt.Sprintf("%s <= (date_trunc('year', CURRENT_DATE) + INTERVAL '1 year - 1 day')::date", fieldRef)
	case OpLastNDays:
		n := "1"
		if len(values) > 0 {
			n = values[0]
		}
		return fmt.Sprintf("%s >= CURRENT_DATE - INTERVAL '%s days' AND %s <= CURRENT_DATE", fieldRef, n, fieldRef)
	case OpPrevious:
		period := "day"
		if len(values) > 0 {
			period = values[0]
		}
		return previousOffsetExpr(fieldRef, period)
	case OpNext:
		period := "day"
		if len(values) > 0 {
			period = values[0]
		}
		return nextOffsetExpr(fieldRef, period)
	default:
		return ""
	}
}

// compileParamOp handles operators with parameter references.
func compileParamOp(fieldRef string, op FilterOperator, paramName string, values []string) string {
	switch op {
	case OpEquals:
		return fmt.Sprintf("%s = %s", fieldRef, paramName)
	case OpNotEquals:
		return fmt.Sprintf("%s != %s", fieldRef, paramName)
	case OpIn:
		return fmt.Sprintf("%s = ANY(%s)", fieldRef, paramName)
	case OpNotIn:
		return fmt.Sprintf("%s != ALL(%s)", fieldRef, paramName)
	case OpGreaterThan:
		return fmt.Sprintf("%s > %s", fieldRef, paramName)
	case OpLessThan:
		return fmt.Sprintf("%s < %s", fieldRef, paramName)
	case OpGreaterEqual:
		return fmt.Sprintf("%s >= %s", fieldRef, paramName)
	case OpLessEqual:
		return fmt.Sprintf("%s <= %s", fieldRef, paramName)
	case OpBetween:
		// For BETWEEN with params: field >= @param_start AND field <= @param_end
		startParam := paramName + "_start"
		endParam := paramName + "_end"
		return fmt.Sprintf("%s >= %s AND %s <= %s", fieldRef, "@"+startParam, fieldRef, "@"+endParam)
	default:
		return compileStandardOp(fieldRef, op, paramName, values)
	}
}

// compileFunctionOp handles arbitrary function expressions.
func compileFunctionOp(fieldRef string, op FilterOperator, expr string) string {
	switch op {
	case OpEquals:
		return fmt.Sprintf("%s = %s", fieldRef, expr)
	case OpNotEquals:
		return fmt.Sprintf("%s != %s", fieldRef, expr)
	case OpGreaterThan:
		return fmt.Sprintf("%s > %s", fieldRef, expr)
	case OpLessThan:
		return fmt.Sprintf("%s < %s", fieldRef, expr)
	case OpGreaterEqual:
		return fmt.Sprintf("%s >= %s", fieldRef, expr)
	case OpLessEqual:
		return fmt.Sprintf("%s <= %s", fieldRef, expr)
	case OpIn:
		return fmt.Sprintf("%s = ANY(%s)", fieldRef, expr)
	default:
		return fmt.Sprintf("%s = %s", fieldRef, expr)
	}
}

func previousOffsetExpr(fieldRef, period string) string {
	switch period {
	case "day":
		return fmt.Sprintf("%s = CURRENT_DATE - INTERVAL '1 day'", fieldRef)
	case "week":
		return fmt.Sprintf("%s = date_trunc('week', CURRENT_DATE) - INTERVAL '1 week'", fieldRef)
	case "month":
		return fmt.Sprintf("%s = date_trunc('month', CURRENT_DATE) - INTERVAL '1 month'", fieldRef)
	case "quarter":
		return fmt.Sprintf("%s = date_trunc('quarter', CURRENT_DATE) - INTERVAL '3 months'", fieldRef)
	case "year":
		return fmt.Sprintf("%s = date_trunc('year', CURRENT_DATE) - INTERVAL '1 year'", fieldRef)
	default:
		return fmt.Sprintf("%s = CURRENT_DATE - INTERVAL '1 day'", fieldRef)
	}
}

func nextOffsetExpr(fieldRef, period string) string {
	switch period {
	case "day":
		return fmt.Sprintf("%s = CURRENT_DATE + INTERVAL '1 day'", fieldRef)
	case "week":
		return fmt.Sprintf("%s = date_trunc('week', CURRENT_DATE) + INTERVAL '1 week'", fieldRef)
	case "month":
		return fmt.Sprintf("%s = date_trunc('month', CURRENT_DATE) + INTERVAL '1 month'", fieldRef)
	case "quarter":
		return fmt.Sprintf("%s = date_trunc('quarter', CURRENT_DATE) + INTERVAL '3 months'", fieldRef)
	case "year":
		return fmt.Sprintf("%s = date_trunc('year', CURRENT_DATE) + INTERVAL '1 year'", fieldRef)
	default:
		return fmt.Sprintf("%s = CURRENT_DATE + INTERVAL '1 day'", fieldRef)
	}
}

// quoteField wraps a field/column name in double quotes if needed.
func quoteField(name string) string {
	if name == "" {
		return `"unknown"`
	}
	// Already quoted
	if strings.HasPrefix(name, `"`) {
		return name
	}
	// Contains dots (schema.table.column)
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		for i, p := range parts {
			if !strings.HasPrefix(p, `"`) {
				parts[i] = `"` + p + `"`
			}
		}
		return strings.Join(parts, ".")
	}
	// Contains special chars
	if strings.ContainsAny(name, " -") {
		return `"` + name + `"`
	}
	return `"` + name + `"`
}

// quoteValue wraps a string constant in single quotes, escaping embedded single quotes.
func quoteValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// escapeLike escapes special characters for ILIKE patterns.
func escapeLike(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "%", "\\%")
}
