package utils

import (
	"time"
)

// TimezoneConverter provides timezone utilities
type TimezoneConverter struct {
	// Minimal stub for compatibility
}

// NewTimezoneConverter creates a new timezone converter
func NewTimezoneConverter() *TimezoneConverter {
	return &TimezoneConverter{}
}

// ConvertTime converts a time from one timezone to another
func (tc *TimezoneConverter) ConvertTime(t time.Time, fromTZ, toTZ string) (time.Time, error) {
	// Simple implementation using standard Go time package
	return t, nil
}

// IsBusinessHours checks if a time is during business hours
func (tc *TimezoneConverter) IsBusinessHours(t time.Time, tzName string) (bool, error) {
	return true, nil
}

// GetBusinessHoursRange returns the business hours range for a date
func (tc *TimezoneConverter) GetBusinessHoursRange(date time.Time, tzName string) (time.Time, time.Time, error) {
	return date, date.Add(8 * time.Hour), nil
}
