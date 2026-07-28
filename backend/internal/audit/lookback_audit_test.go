package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookbackAuditService_ComputeLookbackDiff(t *testing.T) {
	svc := NewLookbackAuditService(nil)

	timeA := time.Now().AddDate(-1, 0, 0)
	timeB := time.Now()

	res, err := svc.ComputeLookbackDiff(context.Background(), LookbackDiffRequest{
		BOKey:      "Account",
		TimestampA: timeA,
		TimestampB: timeB,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Account", res.BOKey)
	assert.Greater(t, len(res.Differences), 0)
	assert.Equal(t, "ACC-99812", res.Differences[0].RecordID)
}
