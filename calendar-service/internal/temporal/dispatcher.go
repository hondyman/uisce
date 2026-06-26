package temporal

import (
	"fmt"
	"strings"
)

// PriorityTier represents the priority classification for job routing
type PriorityTier string

const (
	// CriticalTier: Priority 1-2, SLA < 1 hour
	CriticalTier PriorityTier = "critical"
	// StandardTier: Priority 3-7, SLA 1-24 hours
	StandardTier PriorityTier = "standard"
	// BulkTier: Priority 8-10, SLA > 24 hours
	BulkTier PriorityTier = "bulk"
)

// ValidRegions defines supported regions for data residency
var ValidRegions = map[string]bool{
	"us-east-1":      true,
	"eu-west-1":      true,
	"ap-southeast-1": true,
	"us-west-2":      true,
	"eu-central-1":   true,
}

// GetTaskQueueName returns Temporal task queue name for routing
// Examples:
//   - us-east-1-critical-queue (urgent scheduling jobs)
//   - eu-west-1-standard-queue (normal availability checks)
//   - ap-southeast-1-bulk-queue (batch operations)
func GetTaskQueueName(region string, priority int) string {
	tier := getPriorityTier(priority)
	return fmt.Sprintf("%s-%s-queue", normalizeRegion(region), tier)
}

// getPriorityTier classifies priority into tier for worker pool selection
// Priority range: 1-10
//
//	1-2:   Critical (urgent)
//	3-7:   Standard (normal)
//	8-10:  Bulk (deferred)
func getPriorityTier(priority int) string {
	if priority <= 0 || priority > 10 {
		// Default to standard for invalid priorities
		return string(StandardTier)
	}
	if priority <= 2 {
		return string(CriticalTier)
	} else if priority >= 8 {
		return string(BulkTier)
	}
	return string(StandardTier)
}

// normalizeRegion ensures region name is valid and lowercase
func normalizeRegion(region string) string {
	normalized := strings.ToLower(region)
	// Validate region
	if !ValidRegions[normalized] {
		// Default to us-east-1 for invalid regions
		return "us-east-1"
	}
	return normalized
}

// WorkerPoolConfig defines scaling parameters for a priority tier
type WorkerPoolConfig struct {
	Tier                      PriorityTier
	MaxConcurrentWorkflows    int
	MaxConcurrentActivities   int
	WorkerActivitiesPerSecond float64
	PollerCount               int // Number of pollers per region
	EstimatedJobsPerHour      int // For capacity planning
}

// GetWorkerPoolConfigs returns recommended worker configuration for each priority tier
// These settings balance resource utilization against latency requirements
func GetWorkerPoolConfigs() map[PriorityTier]WorkerPoolConfig {
	return map[PriorityTier]WorkerPoolConfig{
		CriticalTier: {
			Tier:                      CriticalTier,
			MaxConcurrentWorkflows:    20,  // Handle spike bursts
			MaxConcurrentActivities:   30,  // Parallel activity execution
			WorkerActivitiesPerSecond: 100, // High throughput needed
			PollerCount:               3,   // Multiple pollers for responsiveness
			EstimatedJobsPerHour:      500, // ~8-9/min peak load
		},
		StandardTier: {
			Tier:                      StandardTier,
			MaxConcurrentWorkflows:    50,   // Typical workload
			MaxConcurrentActivities:   50,   // Moderate parallelism
			WorkerActivitiesPerSecond: 200,  // Normal throughput
			PollerCount:               2,    // Standard polling cadence
			EstimatedJobsPerHour:      2000, // ~33/min typical
		},
		BulkTier: {
			Tier:                      BulkTier,
			MaxConcurrentWorkflows:    10,   // Batch processing
			MaxConcurrentActivities:   15,   // Limited concurrency
			WorkerActivitiesPerSecond: 50,   // Lower throughput requirement
			PollerCount:               1,    // Single poller is sufficient
			EstimatedJobsPerHour:      1000, // ~16/min background
		},
	}
}

// QueueNames returns all queue names for a given region
func QueueNames(region string) map[PriorityTier]string {
	normalized := normalizeRegion(region)
	return map[PriorityTier]string{
		CriticalTier: fmt.Sprintf("%s-critical-queue", normalized),
		StandardTier: fmt.Sprintf("%s-standard-queue", normalized),
		BulkTier:     fmt.Sprintf("%s-bulk-queue", normalized),
	}
}

// AllQueueNames returns all queue names across all regions and priority tiers
func AllQueueNames(regions []string) map[string]bool {
	queues := make(map[string]bool)
	for _, region := range regions {
		qnames := QueueNames(region)
		for _, qname := range qnames {
			queues[qname] = true
		}
	}
	return queues
}
