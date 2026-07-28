package simulation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplySimulationTransform(t *testing.T) {
	projections := []string{"account.balance", "account.market_value", "account.status"}

	scenario := &SimulationScenario{
		Name: "Interest Rate Shock",
		Rules: []ShockRule{
			{Field: "balance", Operator: "MULTIPLY", Value: 0.92},
			{Field: "market_value", Operator: "ADD", Value: 500.0},
		},
	}

	transformed := ApplySimulationTransform(projections, scenario)
	assert.Len(t, transformed, 3)

	assert.Contains(t, transformed[0], "* 0.920000")
	assert.Contains(t, transformed[0], "balance_simulated")

	assert.Contains(t, transformed[1], "+ 500.000000")
	assert.Contains(t, transformed[1], "market_value_simulated")

	// Status projection remains unmodified
	assert.Equal(t, "account.status", transformed[2])
}
