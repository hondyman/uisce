package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeLayoutSpec_NoConflicts(t *testing.T) {
	ancestor := []byte(`{"id":"order-page","span":12,"title":"Orders"}`)
	modified := []byte(`{"id":"order-page","span":8,"title":"Orders"}`)  // Tenant changed span 12 -> 8
	target := []byte(`{"id":"order-page","span":12,"title":"Orders v2"}`) // Core updated title "Orders" -> "Orders v2"

	mergedBytes, conflicts, err := MergeLayoutSpec(ancestor, modified, target)
	assert.NoError(t, err)
	assert.Empty(t, conflicts, "Non-conflicting changes must merge cleanly without conflicts")

	mergedStr := string(mergedBytes)
	assert.Contains(t, mergedStr, `"span": 8`, "Client customization (span=8) must be preserved")
	assert.Contains(t, mergedStr, `"title": "Orders v2"`, "Core upgrade (title=Orders v2) must be merged in")
}

func TestMergeLayoutSpec_ConflictDetected(t *testing.T) {
	ancestor := []byte(`{"id":"order-page","span":12}`)
	modified := []byte(`{"id":"order-page","span":8}`)  // Client modified span -> 8
	target := []byte(`{"id":"order-page","span":6}`)    // Core upgrade modified span -> 6

	_, conflicts, err := MergeLayoutSpec(ancestor, modified, target)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1, "Conflict must be detected when both client and target modify same property")
	assert.Equal(t, "span", conflicts[0].PropertyPath)
}
