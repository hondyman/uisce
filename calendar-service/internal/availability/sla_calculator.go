package availability

import "time"

// SLACalculator handles SLA and fulfillment time calculations
type SLACalculator struct {
	window *time.Time // Availability window
	metric string     // Metric type: "fulfillment_time", "compliance_rate", etc.
}

// SLAMetrics represents calculated SLA metrics
type SLAMetrics struct {
	IsCompliant      bool          `json:"is_compliant"`
	FulfillmentTime  time.Duration `json:"fulfillment_time"`
	ComplianceRate   float32       `json:"compliance_rate"`
	TargetSLA        time.Duration `json:"target_sla"`
	BreachDuration   time.Duration `json:"breach_duration"`
	LastCalculatedAt time.Time     `json:"last_calculated_at"`
}

// NewSLACalculator creates a new SLA calculator
func NewSLACalculator() *SLACalculator {
	return &SLACalculator{}
}

// CalculateFulfillmentTime calculates the fulfillment time for a given request
// It accounts for blackout periods and availability windows
func (s *SLACalculator) CalculateFulfillmentTime(
	startTime time.Time,
	availabilityOccurrences []Occurrence,
	blackoutOccurrences []Occurrence,
) time.Duration {
	if len(availabilityOccurrences) == 0 {
		return 0
	}

	// Simple implementation: find first available slot after start time
	for _, avail := range availabilityOccurrences {
		if avail.StartTime.After(startTime) {
			// Check if this slot is not blocked by blackouts
			isBlocked := false
			for _, blackout := range blackoutOccurrences {
				if !(avail.EndTime.Before(blackout.StartTime) || avail.StartTime.After(blackout.EndTime)) {
					isBlocked = true
					break
				}
			}

			if !isBlocked {
				return avail.StartTime.Sub(startTime)
			}
		}
	}

	// No available slot found
	return 0
}

// CalculateComplianceRate calculates the SLA compliance rate
// Compliance is measured as (available_time / total_time) * 100
func (s *SLACalculator) CalculateComplianceRate(
	availabilityOccurrences []Occurrence,
	blackoutOccurrences []Occurrence,
	periodStart time.Time,
	periodEnd time.Time,
) float32 {
	if periodEnd.Before(periodStart) {
		return 0
	}

	totalDuration := periodEnd.Sub(periodStart)
	availableDuration := time.Duration(0)

	for _, avail := range availabilityOccurrences {
		// Clamp to period boundaries
		start := avail.StartTime
		if start.Before(periodStart) {
			start = periodStart
		}

		end := avail.EndTime
		if end.After(periodEnd) {
			end = periodEnd
		}

		// Subtract blackouts from this availability window
		blockDuration := time.Duration(0)
		for _, blackout := range blackoutOccurrences {
			// Calculate overlap between availability and blackout
			overlapStart := start
			if overlapStart.Before(blackout.StartTime) {
				overlapStart = blackout.StartTime
			}

			overlapEnd := end
			if overlapEnd.After(blackout.EndTime) {
				overlapEnd = blackout.EndTime
			}

			if overlapStart.Before(overlapEnd) {
				blockDuration += overlapEnd.Sub(overlapStart)
			}
		}

		windowDuration := end.Sub(start) - blockDuration
		if windowDuration > 0 {
			availableDuration += windowDuration
		}
	}

	if totalDuration == 0 {
		return 0
	}

	complianceRate := float32(availableDuration.Seconds()) / float32(totalDuration.Seconds()) * 100
	if complianceRate > 100 {
		complianceRate = 100
	}

	return complianceRate
}
