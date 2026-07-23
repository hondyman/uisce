package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// DataQuality represents data quality metrics for an answer
type DataQuality struct {
	Freshness   string  `json:"freshness"`    // e.g. "3h ago"
	NullRate    float64 `json:"null_rate"`    // e.g. 0.021 (2.1%)
	SLA         string  `json:"sla"`          // e.g. "99.9% availability"
	LineageNote string  `json:"lineage_note"` // e.g. "Upstream node stale"
	FreshnessStatus string `json:"freshness_status"` // RED, AMBER, GREEN
}

// DataQualityService computes data quality metrics
type DataQualityService struct {
	db *sqlx.DB
}

// NewDataQualityService creates a new data quality service
func NewDataQualityService(db *sqlx.DB) *DataQualityService {
	return &DataQualityService{db: db}
}

// ComputeQuality calculates data quality metrics for given sources
func (s *DataQualityService) ComputeQuality(ctx context.Context, sources []string, tenantID string) (*DataQuality, error) {
	if len(sources) == 0 {
		return &DataQuality{
			Freshness: "N/A",
			NullRate:  0,
			SLA:       "N/A",
			FreshnessStatus: "GREEN",
		}, nil
	}
	
	// Aggregate quality metrics from all sources
	var totalNullRate float64
	var oldestFreshness time.Duration
	var slaViolations int
	var lineageNotes []string
	
	for _, source := range sources {
		metrics, err := s.getNodeQuality(ctx, source, tenantID)
		if err != nil {
			// Non-fatal, continue with other sources
			continue
		}
		
		if metrics != nil {
			totalNullRate += metrics.NullRate
			if metrics.Freshness > oldestFreshness {
				oldestFreshness = metrics.Freshness
			}
			if !metrics.SLAMet {
				slaViolations++
			}
			if metrics.LineageNote != "" {
				lineageNotes = append(lineageNotes, metrics.LineageNote)
			}
		}
	}
	
	avgNullRate := totalNullRate / float64(len(sources))
	
	// Determine freshness status (RED/AMBER/GREEN)
	freshnessStatus := "GREEN"
	if oldestFreshness > 24*time.Hour {
		freshnessStatus = "RED"
	} else if oldestFreshness > 6*time.Hour {
		freshnessStatus = "AMBER"
	}
	
	slaCompliance := float64(len(sources)-slaViolations) / float64(len(sources)) * 100
	
	lineageNote := ""
	if len(lineageNotes) > 0 {
		lineageNote = fmt.Sprintf("%d upstream issues detected", len(lineageNotes))
	}
	
	return &DataQuality{
		Freshness:       formatDuration(oldestFreshness),
		NullRate:        avgNullRate,
		SLA:             fmt.Sprintf("%.1f%% availability", slaCompliance),
		LineageNote:     lineageNote,
		FreshnessStatus: freshnessStatus,
	}, nil
}

type nodeQualityMetrics struct {
	NullRate     float64
	Freshness    time.Duration
	SLAMet       bool
	LineageNote  string
}

// getNodeQuality retrieves quality metrics for a single catalog node
func (s *DataQualityService) getNodeQuality(ctx context.Context, qualifiedPath string, tenantID string) (*nodeQualityMetrics, error) {
	query := `
		SELECT 
			COALESCE(updated_at, created_at) as last_update,
			data_quality_contract,
			sla
		FROM catalog_node
		WHERE qualified_path = $1 AND tenant_id = $2
	`
	
	var lastUpdate time.Time
	var dqContract sql.NullString
	var slaData sql.NullString
	
	err := s.db.GetContext(ctx, &struct {
		LastUpdate time.Time      `db:"last_update"`
		DQContract sql.NullString `db:"data_quality_contract"`
		SLA        sql.NullString `db:"sla"`
	}{LastUpdate: lastUpdate, DQContract: dqContract, SLA: slaData}, query, qualifiedPath, tenantID)
	
	if err != nil {
		return nil, err
	}
	
	metrics := &nodeQualityMetrics{
		Freshness: time.Since(lastUpdate),
		SLAMet:    true,
	}
	
	// Parse data quality contract
	if dqContract.Valid {
		var dq struct {
			NullRateThreshold   float64 `json:"null_rate_threshold"`
			FreshnessSLAHours   int     `json:"freshness_sla_hours"`
			CompletenessTarget  float64 `json:"completeness_target"`
		}
		if err := json.Unmarshal([]byte(dqContract.String), &dq); err == nil {
			metrics.NullRate = dq.NullRateThreshold
			
			// Check if freshness SLA is met
			if dq.FreshnessSLAHours > 0 {
				if metrics.Freshness > time.Duration(dq.FreshnessSLAHours)*time.Hour {
					metrics.SLAMet = false
					metrics.LineageNote = fmt.Sprintf("Data older than %dh SLA", dq.FreshnessSLAHours)
				}
			}
		}
	}
	
	// Parse SLA
	if slaData.Valid {
		var sla struct {
			AvailabilityTarget float64 `json:"availability_target"`
			UpdateFrequency    string  `json:"update_frequency"`
		}
		if err := json.Unmarshal([]byte(slaData.String), &sla); err == nil {
			// In production, you'd check actual availability vs target
			// For now, assume SLA is met if data is fresh
			if metrics.Freshness > 48*time.Hour {
				metrics.SLAMet = false
			}
		}
	}
	
	return metrics, nil
}

// formatDuration formats a duration into human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}
