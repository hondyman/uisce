package contracts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataContractStructs(t *testing.T) {
	contract := DataContract{
		TenantID:   "tenant_gold",
		BOName:     "Customer",
		Version:    "v1.0.0",
		SchemaJson: `{"id": "string", "name": "string"}`,
		Status:     "ACTIVE",
		Breaking:   false,
	}

	assert.Equal(t, "Customer", contract.BOName)
	assert.Equal(t, "v1.0.0", contract.Version)
	assert.False(t, contract.Breaking)
}
