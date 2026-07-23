package capacity

import (
	"context"
	"time"
)

type ResourceType string

const (
	ResourceCompute ResourceType = "compute"
	ResourceStorage ResourceType = "storage"
	ResourceNetwork ResourceType = "network"
	ResourceCache   ResourceType = "cache"
)

type CapacityForecast struct {
	TenantID       string       `json:"tenant_id"`
	Resource       ResourceType `json:"resource"`
	CurrentUsage   float64      `json:"current_usage"`
	ForecastUsage  float64      `json:"forecast_usage"`
	ForecastPeriod string       `json:"forecast_period"` // 30_days, 90_days, 1_year
	GrowthRate     float64      `json:"growth_rate"`
	Confidence     float64      `json:"confidence"`
	Drivers        []string     `json:"drivers"`
}

type ScalingRecommendation struct {
	Resource  ResourceType `json:"resource"`
	Action    string       `json:"action"` // scale_up, pre_allocate, pre_warm
	Amount    float64      `json:"amount"`
	Timing    time.Time    `json:"timing"`
	Rationale string       `json:"rationale"`
}

type CapacityPlanner struct{}

func NewCapacityPlanner() *CapacityPlanner {
	return &CapacityPlanner{}
}

func (cp *CapacityPlanner) Forecast(ctx context.Context, tenantID string) ([]CapacityForecast, error) {
	// Mock: Generate forecasts
	// Real: Analyze historical traffic, tenant growth, seasonality, pre-agg costs

	forecasts := []CapacityForecast{
		{
			TenantID:       tenantID,
			Resource:       ResourceCompute,
			CurrentUsage:   100,
			ForecastUsage:  200,
			ForecastPeriod: "90_days",
			GrowthRate:     1.0,
			Confidence:     0.85,
			Drivers: []string{
				"New app rollout (Portfolio Analytics)",
				"User growth (+45% projected)",
				"Increased workflow complexity",
			},
		},
		{
			TenantID:       tenantID,
			Resource:       ResourceStorage,
			CurrentUsage:   500,
			ForecastUsage:  590,
			ForecastPeriod: "30_days",
			GrowthRate:     0.18,
			Confidence:     0.92,
			Drivers: []string{
				"Positions API data growth",
				"Pre-agg materialization",
			},
		},
		{
			TenantID:       tenantID,
			Resource:       ResourceCache,
			CurrentUsage:   10,
			ForecastUsage:  30,
			ForecastPeriod: "30_days",
			GrowthRate:     2.0,
			Confidence:     0.78,
			Drivers: []string{
				"Risk Dashboard usage spike during market open",
				"3× edge cache capacity needed",
			},
		},
	}

	return forecasts, nil
}

func (cp *CapacityPlanner) Recommend(ctx context.Context, forecasts []CapacityForecast) ([]ScalingRecommendation, error) {
	// Mock: Generate recommendations
	// Real: Generate scaling actions based on forecasts

	recommendations := make([]ScalingRecommendation, 0)

	for _, forecast := range forecasts {
		if forecast.GrowthRate > 0.5 {
			recommendations = append(recommendations, ScalingRecommendation{
				Resource:  forecast.Resource,
				Action:    "scale_up",
				Amount:    forecast.ForecastUsage - forecast.CurrentUsage,
				Timing:    time.Now().Add(7 * 24 * time.Hour),
				Rationale: "Proactive scaling to meet forecasted demand",
			})
		}
	}

	return recommendations, nil
}
