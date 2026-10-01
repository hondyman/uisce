package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// compareExtended implements the condition operators the rule editor
// (AdvancedConditionBuilder.tsx) offers beyond numeric comparison and null
// checks. Semantics mirror the editor's client-side evaluator exactly, so a
// rule previews and enforces the same way:
//
//   - string operators compare String(value) case-insensitively
//     (contains, not_contains, starts_with, ends_with)
//   - list operators compare by string form (in, not_in, and the array
//     operators contains_any, contains_all); a list may be an array or a
//     comma-separated string
//   - length_* measure the string form in characters (runes)
//   - between/not_between are inclusive numeric ranges over [low, high]
//   - date operators (before, after, in_last_n_days, is_today, ...) are
//     in date_operators.go
//   - matches_regex uses RE2 (Go regexp); an invalid pattern is an error
//     here rather than the editor's silent false - an unevaluable rule must
//     never read as "data failed"
//
// Absent values never reach this function (evaluateSimpleCondition returns
// before it); is_empty/is_not_empty handle the absent case themselves.
// handled is false for operators this function does not know.
func compareExtended(actual interface{}, operator string, expected interface{}) (ok bool, handled bool, err error) {
	switch operator {
	case "contains":
		return strings.Contains(lowerStr(actual), lowerStr(expected)), true, nil
	case "not_contains":
		return !strings.Contains(lowerStr(actual), lowerStr(expected)), true, nil
	case "starts_with":
		return strings.HasPrefix(lowerStr(actual), lowerStr(expected)), true, nil
	case "ends_with":
		return strings.HasSuffix(lowerStr(actual), lowerStr(expected)), true, nil

	case "matches_regex":
		re, err := cachedRegexp(jsString(expected))
		if err != nil {
			return false, true, fmt.Errorf("matches_regex: invalid pattern %q: %w", jsString(expected), err)
		}
		return re.MatchString(jsString(actual)), true, nil

	case "is_empty":
		return isEmptyValue(actual), true, nil
	case "is_not_empty":
		return !isEmptyValue(actual), true, nil

	case "between", "not_between":
		// Inclusive numeric range; expected is [low, high] (the evaluator
		// packs Value/SecondValue into that form). Bad bounds are a rule
		// error; a non-numeric value simply fails, as in the editor.
		b := listOf(expected)
		if len(b) != 2 {
			return false, true, fmt.Errorf("%s requires [low, high], got %v", operator, expected)
		}
		lo, okLo := toFloatAny(b[0])
		hi, okHi := toFloatAny(b[1])
		if !okLo || !okHi {
			return false, true, fmt.Errorf("%s: bounds must be numeric, got %v", operator, expected)
		}
		v, ok := toFloatAny(actual)
		if !ok {
			return false, true, nil
		}
		if operator == "between" {
			return v >= lo && v <= hi, true, nil
		}
		return v < lo || v > hi, true, nil

	case "length_equals", "length_greater", "length_less":
		n, ok := toFloatAny(expected)
		if !ok {
			return false, true, fmt.Errorf("%s: length must be numeric, got %v", operator, expected)
		}
		l := float64(utf8.RuneCountInString(jsString(actual)))
		switch operator {
		case "length_equals":
			return l == n, true, nil
		case "length_greater":
			return l > n, true, nil
		default:
			return l < n, true, nil
		}

	case "in", "not_in":
		found := false
		a := jsString(actual)
		for _, item := range listOf(expected) {
			if jsString(item) == a {
				found = true
				break
			}
		}
		if operator == "in" {
			return found, true, nil
		}
		return !found, true, nil

	case "contains_any", "contains_all":
		arr, isArr := actual.([]interface{})
		if !isArr {
			if ss, ok := actual.([]string); ok {
				for _, s := range ss {
					arr = append(arr, s)
				}
				isArr = true
			}
		}
		if !isArr {
			return false, true, nil
		}
		have := make(map[string]bool, len(arr))
		for _, v := range arr {
			have[jsString(v)] = true
		}
		cands := listOf(expected)
		if operator == "contains_any" {
			for _, c := range cands {
				if have[jsString(c)] {
					return true, true, nil
				}
			}
			return false, true, nil
		}
		for _, c := range cands {
			if !have[jsString(c)] {
				return false, true, nil
			}
		}
		return true, true, nil

	case "is_true":
		return actual == true || actual == "true", true, nil
	case "is_false":
		return actual == false || actual == "false", true, nil

	case "is_positive", "is_negative", "is_zero":
		n, ok := toFloatAny(actual)
		if !ok {
			return false, true, nil
		}
		switch operator {
		case "is_positive":
			return n > 0, true, nil
		case "is_negative":
			return n < 0, true, nil
		default:
			return n == 0, true, nil
		}
	}
	return compareDate(actual, operator, expected)
}

// jsString renders a value the way the editor's String(v) does for the
// scalar types rules carry: integral floats print without a decimal point.
func jsString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case []interface{}:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = jsString(e)
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprint(v)
}

func lowerStr(v interface{}) string { return strings.ToLower(jsString(v)) }

// listOf accepts the editor's two list encodings: an array, or a
// comma-separated string.
func listOf(v interface{}) []interface{} {
	switch x := v.(type) {
	case []interface{}:
		return x
	case []string:
		out := make([]interface{}, len(x))
		for i, s := range x {
			out[i] = s
		}
		return out
	case string:
		parts := strings.Split(x, ",")
		out := make([]interface{}, len(parts))
		for i, p := range parts {
			out[i] = strings.TrimSpace(p)
		}
		return out
	case nil:
		return nil
	}
	return []interface{}{v}
}

func isEmptyValue(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []interface{}:
		return len(x) == 0
	case []string:
		return len(x) == 0
	case map[string]interface{}:
		return len(x) == 0
	}
	return false
}

func toFloatAny(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

var regexCache sync.Map // pattern -> *regexp.Regexp

func cachedRegexp(pattern string) (*regexp.Regexp, error) {
	if re, ok := regexCache.Load(pattern); ok {
		return re.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}
