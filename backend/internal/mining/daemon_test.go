package mining

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestQueryMiningDaemon(t *testing.T) {
	daemon := NewQueryMiningDaemon(nil)

	err := daemon.ProcessQueryLogEntry(
		context.Background(),
		uuid.New(),
		"STARROCKS",
		"SELECT c.name, SUM(o.total * 0.85) AS net_revenue FROM orders o JOIN customers c ON o.cust_id = c.id GROUP BY c.name",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
