package contracts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBIManifestGenerator(t *testing.T) {
	gen := NewBIConnectorManifestGenerator()

	pbids := gen.GeneratePowerBIManifest("localhost", 5433, "uisce")
	assert.Contains(t, pbids, "postgresql")
	assert.Contains(t, pbids, "5433")

	tds := gen.GenerateTableauDataSourcesManifest("localhost", 5433, "uisce", "Customer", []string{"id", "company_name"})
	assert.Contains(t, tds, "<datasource formatted-name='UisceSemanticOS'")
	assert.Contains(t, tds, "table='[Customer]'")

	cubeSchema := gen.GenerateCubeDevSchema("Customer", []string{"id", "region"}, []string{"revenue"})
	assert.Contains(t, cubeSchema, "cube('Customer'")
	assert.Contains(t, cubeSchema, "total_revenue:")
}
