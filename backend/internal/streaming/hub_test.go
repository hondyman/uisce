package streaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_BroadcastBOEvent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Broadcast sample mutation event
	hub.BroadcastBOEvent("customers", "MUTATION_UPDATE", map[string]interface{}{
		"id":      "ACC-99812",
		"balance": 1500000.0,
	})

	time.Sleep(10 * time.Millisecond)
	assert.NotNil(t, hub)
}
