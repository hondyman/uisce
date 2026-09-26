package datapipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func TestScheduleValidateAndNext(t *testing.T) {
	for _, c := range []struct {
		s       Schedule
		wantErr string
	}{
		{Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin"}, ""},
		{Schedule{Cron: "*/15 * * * *"}, ""},
		{Schedule{Cron: "* * * * *"}, "at most every 5 minutes"},
		{Schedule{Cron: "0 6 * *"}, "not a valid 5-field cron"},
		{Schedule{Cron: "0 6 * * *", TimeZone: "Mars/Olympus"}, "unknown time zone"},
	} {
		err := c.s.Validate()
		if (c.wantErr == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%+v: err = %v, want %q", c.s, err, c.wantErr)
		}
	}
	// Weekdays 06:00 Dublin, from Friday 2026-09-25 12:00 UTC: Mon 28, Tue 29 at 06:00 local (05:00 UTC, IST).
	next, err := Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin"}.Next(2, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if next[0].UTC() != time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC) || next[1].UTC() != time.Date(2026, 9, 29, 5, 0, 0, 0, time.UTC) {
		t.Errorf("next = %v", next)
	}
}

func TestScheduledWorkflowRunsItsActivityOnce(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	calls := 0
	env.RegisterActivityWithOptions(func(_ context.Context, in ScheduledInput) error {
		calls++
		if in.PipelineID != "p1" || in.TenantID != "t1" {
			t.Errorf("input %+v", in)
		}
		return errors.New("failed")
	}, activity.RegisterOptions{Name: ScheduledActivityName})
	env.ExecuteWorkflow(ScheduledWorkflow, ScheduledInput{TenantID: "t1", PipelineID: "p1"})
	if env.GetWorkflowError() == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", env.GetWorkflowError(), calls)
	}
}

// Apply against a real Temporal dev server: create, update, disable.
func TestTemporalSchedulerApply(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "temporalio/temporal:latest",
			Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
			ExposedPorts: []string{"7233/tcp"},
			WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(ctx) })
	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "7233")
	var tc client.Client
	for i := 0; i < 30; i++ { // the frontend may need a moment after the port opens
		if tc, err = client.Dial(client.Options{HostPort: host + ":" + port.Port()}); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer tc.Close()

	sch := &TemporalScheduler{Client: tc}
	id := ScheduleID("t1", "p1")
	describe := func() (*client.ScheduleDescription, error) {
		return tc.ScheduleClient().GetHandle(ctx, id).Describe(ctx)
	}
	var lastErr error
	for i := 0; i < 20; i++ { // namespace readiness
		if lastErr = sch.Apply(ctx, "t1", "p1", &Schedule{Cron: "0 6 * * 1-5", TimeZone: "Europe/Dublin", Enabled: true}); lastErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		t.Fatal(lastErr)
	}
	d, err := describe()
	if err != nil {
		t.Fatal(err)
	}
	if d.Schedule.Spec.TimeZoneName != "Europe/Dublin" || d.Schedule.Policy.Overlap.String() != "Skip" {
		t.Errorf("created: tz=%q overlap=%v", d.Schedule.Spec.TimeZoneName, d.Schedule.Policy.Overlap)
	}
	act := d.Schedule.Action.(*client.ScheduleWorkflowAction)
	if act.TaskQueue != TaskQueue {
		t.Errorf("task queue %q", act.TaskQueue)
	}
	first := d.Info.NextActionTimes

	// Update in place.
	if err := sch.Apply(ctx, "t1", "p1", &Schedule{Cron: "30 22 * * *", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	d, _ = describe()
	if len(d.Info.NextActionTimes) == 0 || (len(first) > 0 && d.Info.NextActionTimes[0].Equal(first[0])) {
		t.Errorf("update did not change the next run times: %v", d.Info.NextActionTimes)
	}
	if n := d.Info.NextActionTimes[0].UTC(); n.Hour() != 22 || n.Minute() != 30 {
		t.Errorf("next run %v, want 22:30 UTC", n)
	}

	// Disable removes it; disabling again is harmless.
	for i := 0; i < 2; i++ {
		if err := sch.Apply(ctx, "t1", "p1", &Schedule{Cron: "30 22 * * *", Enabled: false}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := describe(); err == nil {
		t.Error("disabled schedule must be removed")
	}
}
