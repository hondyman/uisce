package vm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Operator struct {
	Name        string
	Description string
	Evaluate    func(actual, expected interface{}) (bool, error)
}

type OperatorRegistry struct {
	operators map[string]Operator
}

func NewOperatorRegistry() *OperatorRegistry {
	r := &OperatorRegistry{
		operators: make(map[string]Operator),
	}
	r.registerDefaultOperators()
	return r
}

func (r *OperatorRegistry) registerDefaultOperators() {
	r.Register(Operator{
		Name:        "equals",
		Description: "Exact equality",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected), nil
		},
	})

	r.Register(Operator{
		Name:        "not_equals",
		Description: "Not equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected), nil
		},
	})

	r.Register(Operator{
		Name:        "is_null",
		Description: "Value is null/empty",
		Evaluate: func(actual, _ interface{}) (bool, error) {
			if actual == nil {
				return true, nil
			}
			s := fmt.Sprintf("%v", actual)
			return s == "" || s == "<nil>", nil
		},
	})

	r.Register(Operator{
		Name:        "is_not_null",
		Description: "Value is not null/empty",
		Evaluate: func(actual, _ interface{}) (bool, error) {
			if actual == nil {
				return false, nil
			}
			s := fmt.Sprintf("%v", actual)
			return s != "" && s != "<nil>", nil
		},
	})

	r.Register(Operator{
		Name:        "greater_than",
		Description: "Greater than",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convToFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := convToFloat64(expected)
			if err != nil {
				return false, err
			}
			return a > e, nil
		},
	})

	r.Register(Operator{
		Name:        "greater_than_or_equals",
		Description: "Greater than or equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convToFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := convToFloat64(expected)
			if err != nil {
				return false, err
			}
			return a >= e, nil
		},
	})

	r.Register(Operator{
		Name:        "less_than",
		Description: "Less than",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convToFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := convToFloat64(expected)
			if err != nil {
				return false, err
			}
			return a < e, nil
		},
	})

	r.Register(Operator{
		Name:        "less_than_or_equals",
		Description: "Less than or equal",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convToFloat64(actual)
			if err != nil {
				return false, err
			}
			e, err := convToFloat64(expected)
			if err != nil {
				return false, err
			}
			return a <= e, nil
		},
	})

	r.Register(Operator{
		Name:        "between",
		Description: "Value between two values (inclusive)",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convToFloat64(actual)
			if err != nil {
				return false, err
			}
			bounds, ok := expected.([]interface{})
			if !ok || len(bounds) != 2 {
				return false, fmt.Errorf("between requires [min, max] array")
			}
			min, err := convToFloat64(bounds[0])
			if err != nil {
				return false, err
			}
			max, err := convToFloat64(bounds[1])
			if err != nil {
				return false, err
			}
			return a >= min && a <= max, nil
		},
	})

	r.Register(Operator{
		Name:        "contains",
		Description: "String contains substring",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.Contains(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "not_contains",
		Description: "String does not contain substring",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return !strings.Contains(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "starts_with",
		Description: "String starts with prefix",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.HasPrefix(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "ends_with",
		Description: "String ends with suffix",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			e := fmt.Sprintf("%v", expected)
			return strings.HasSuffix(a, e), nil
		},
	})

	r.Register(Operator{
		Name:        "matches_regex",
		Description: "Matches regular expression",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a := fmt.Sprintf("%v", actual)
			pattern := fmt.Sprintf("%v", expected)
			return regexp.MatchString(pattern, a)
		},
	})

	r.Register(Operator{
		Name:        "in",
		Description: "Value in list",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			list, ok := expected.([]interface{})
			if !ok {
				return false, fmt.Errorf("in operator requires array")
			}
			a := fmt.Sprintf("%v", actual)
			for _, item := range list {
				if fmt.Sprintf("%v", item) == a {
					return true, nil
				}
			}
			return false, nil
		},
	})

	r.Register(Operator{
		Name:        "not_in",
		Description: "Value not in list",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			list, ok := expected.([]interface{})
			if !ok {
				return false, fmt.Errorf("not_in operator requires array")
			}
			a := fmt.Sprintf("%v", actual)
			for _, item := range list {
				if fmt.Sprintf("%v", item) == a {
					return false, nil
				}
			}
			return true, nil
		},
	})

	r.Register(Operator{
		Name:        "date_before",
		Description: "Date is before",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convParseTime(actual)
			if err != nil {
				return false, err
			}
			e, err := convParseTime(expected)
			if err != nil {
				return false, err
			}
			return a.Before(e), nil
		},
	})

	r.Register(Operator{
		Name:        "date_after",
		Description: "Date is after",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convParseTime(actual)
			if err != nil {
				return false, err
			}
			e, err := convParseTime(expected)
			if err != nil {
				return false, err
			}
			return a.After(e), nil
		},
	})

	r.Register(Operator{
		Name:        "days_ago_less_than",
		Description: "Date within N days ago",
		Evaluate: func(actual, expected interface{}) (bool, error) {
			a, err := convParseTime(actual)
			if err != nil {
				return false, err
			}
			days, err := convToFloat64(expected)
			if err != nil {
				return false, err
			}
			threshold := time.Now().AddDate(0, 0, -int(days))
			return a.After(threshold), nil
		},
	})
}

func (r *OperatorRegistry) Register(op Operator) {
	r.operators[op.Name] = op
}

func (r *OperatorRegistry) Get(name string) (Operator, bool) {
	op, ok := r.operators[name]
	return op, ok
}

func convToFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	case json.Number:
		return val.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func convParseTime(v interface{}) (time.Time, error) {
	switch val := v.(type) {
	case time.Time:
		return val, nil
	case string:
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02",
			"01/02/2006",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse time: %s", val)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", v)
	}
}
