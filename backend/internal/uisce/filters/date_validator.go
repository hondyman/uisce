package filters

import (
	"context"
	"fmt"
	"time"
)

// DateValidatorFilter validates date fields against various rules
type DateValidatorFilter struct {
	FieldName     string    // Field to validate
	Rule          string    // "after", "before", "between", "future", "past"
	ReferenceDate time.Time // For after/before rules
	EndDate       time.Time // For between rule
}

func (f *DateValidatorFilter) Name() string {
	return "Date Validator"
}

func (f *DateValidatorFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	value, ok := data[f.FieldName]
	if !ok {
		return fmt.Errorf("field '%s' not found in data", f.FieldName)
	}

	var dateValue time.Time
	switch v := value.(type) {
	case time.Time:
		dateValue = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", v)
			if err != nil {
				return fmt.Errorf("field '%s' is not a valid date format", f.FieldName)
			}
		}
		dateValue = parsed
	default:
		return fmt.Errorf("field '%s' is not a date", f.FieldName)
	}

	now := time.Now()

	switch f.Rule {
	case "future":
		if !dateValue.After(now) {
			return fmt.Errorf("date must be in the future")
		}
	case "past":
		if !dateValue.Before(now) {
			return fmt.Errorf("date must be in the past")
		}
	case "after":
		if !dateValue.After(f.ReferenceDate) {
			return fmt.Errorf("date must be after %s", f.ReferenceDate.Format("2006-01-02"))
		}
	case "before":
		if !dateValue.Before(f.ReferenceDate) {
			return fmt.Errorf("date must be before %s", f.ReferenceDate.Format("2006-01-02"))
		}
	case "between":
		if dateValue.Before(f.ReferenceDate) || dateValue.After(f.EndDate) {
			return fmt.Errorf("date must be between %s and %s",
				f.ReferenceDate.Format("2006-01-02"),
				f.EndDate.Format("2006-01-02"))
		}
	}

	return nil
}
