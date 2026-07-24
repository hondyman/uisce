package sync

import (
	"time"
)

type TimezoneService struct {
	// Default system timezone if needed
}

func NewTimezoneService() *TimezoneService {
	return &TimezoneService{}
}

// ConvertToUTC converts a time to UTC
func (s *TimezoneService) ConvertToUTC(t time.Time, originalTimezone string) time.Time {
	// Record metric
	timezoneConversions.WithLabelValues(originalTimezone, "UTC").Inc()
	return t.UTC()
}

// ConvertToTimezone converts a time to a specific location
func (s *TimezoneService) ConvertToTimezone(t time.Time, location string) (time.Time, error) {
	loc, err := time.LoadLocation(location)
	if err != nil {
		timezoneConversionErrors.WithLabelValues("load_location").Inc()
		return t, err
	}
	timezoneConversions.WithLabelValues("UTC", location).Inc()
	return t.In(loc), nil
}
