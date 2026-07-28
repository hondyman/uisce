package regulatory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateForm13FXML(t *testing.T) {
	xmlBytes, err := GenerateForm13FXML(context.Background(), "0001234567", "", nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, xmlBytes)
	xmlStr := string(xmlBytes)

	assert.Contains(t, xmlStr, "<?xml version=")
	assert.Contains(t, xmlStr, "13F-HR")
	assert.Contains(t, xmlStr, "APPLE INC")
	assert.Contains(t, xmlStr, "037833100")
}
