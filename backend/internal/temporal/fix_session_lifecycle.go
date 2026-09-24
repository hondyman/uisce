package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// FIXSessionLifecycleInput is the workflow input for FIXSessionLifecycleWorkflow.
type FIXSessionLifecycleInput struct {
	TenantID uuid.UUID `json:"tenant_id"`
	BrokerID uuid.UUID `json:"broker_id"`
	// AdminURL is the admin API endpoint of the acceptor process
	// (e.g. "http://127.0.0.1:8981"). Parameterized so the workflow
	// doesn't bake in a hostname — see HANDOFF_FIX_OVER_PIPELINE.md
	// §19 "Admin-API network co-location" for the container caveat.
	AdminURL string `json:"admin_url"`
	// AdminToken is the shared-secret header value. The workflow
	// carries it through to activities; activities send it as the
	// X-Fix-Admin-Token header.
	AdminToken string `json:"admin_token"`
	// SessionID is the FIX session id (e.g. "FIX.4.4:BUYER->SELLER").
	SessionID string `json:"session_id"`
}

// FIXSessionLifecycleState tracks the workflow's view of session state.
// The actual TCP session state lives in the acceptor process; this is
// the workflow's durable record. See HANDOFF_FIX_OVER_PIPELINE.md §10.
type FIXSessionLifecycleState struct {
	State            string    `json:"state"` // INITIALIZING | LOGGING_ON | STREAMING | RECONNECTING | LOGGING_OUT | STOPPED
	LastTransitionAt time.Time `json:"last_transition_at"`
	LastError        string    `json:"last_error,omitempty"`
}

// FIXSessionLifecycleWorkflow is the long-running workflow that
// controls session lifecycle for one (tenant, broker) pair. Per
// HANDOFF_FIX_OVER_PIPELINE.md §10 and §6 (Amendment 1), the workflow
// owns *state and policy* — never the TCP socket. It calls the
// acceptor's admin API via activities.
func FIXSessionLifecycleWorkflow(ctx workflow.Context, input FIXSessionLifecycleInput) error {
	logger := workflow.GetLogger(ctx)

	state := FIXSessionLifecycleState{
		State:            "INITIALIZING",
		LastTransitionAt: workflow.Now(ctx),
	}

	// Signal channels for external control.
	startSig := workflow.GetSignalChannel(ctx, "Start")
	stopSig := workflow.GetSignalChannel(ctx, "Stop")
	healthSig := workflow.GetSignalChannel(ctx, "HealthCheck")

	// Periodic liveness check (Amendment 1: SessionLivenessCheckActivity,
	// not a duplicate of the FIX-protocol heartbeat).
	livenessTimer := workflow.NewTimer(ctx, 30*time.Second)

	// Activity options.
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		},
	}
	actx := workflow.WithActivityOptions(ctx, ao)

	transition := func(next string) {
		logger.Info("FIXSessionLifecycle: state transition", "from", state.State, "to", next)
		state.State = next
		state.LastTransitionAt = workflow.Now(ctx)
	}

	// Initial logon attempt.
	transition("LOGGING_ON")
	if err := workflow.ExecuteActivity(actx, "LogonActivity", input).Get(actx, nil); err != nil {
		// The acceptor returns 501 for acceptor-side logon (logon is
		// broker-driven). Treat that as "no error" — we just wait for
		// the broker to log us on.
		state.LastError = fmt.Sprintf("logon: %v", err)
		transition("RECONNECTING")
	}

	// Main loop: select on signals and the liveness timer.
	for {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(startSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("FIXSessionLifecycle: Start signal received", "payload", sig)
		})

		selector.AddReceive(stopSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("FIXSessionLifecycle: Stop signal received", "payload", sig)
			transition("LOGGING_OUT")
			_ = workflow.ExecuteActivity(actx, "LogoutActivity", input).Get(ctx, nil)
			transition("STOPPED")
		})

		selector.AddReceive(healthSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("FIXSessionLifecycle: HealthCheck signal received", "payload", sig)
		})

		selector.AddFuture(livenessTimer, func(f workflow.Future) {
			var healthResult HealthResult
			_ = workflow.ExecuteActivity(actx, "SessionLivenessCheckActivity", input).Get(ctx, &healthResult)
			if !healthResult.Healthy {
				state.LastError = fmt.Sprintf("liveness check failed at %s", healthResult.CheckedAt)
				transition("RECONNECTING")
				_ = workflow.ExecuteActivity(actx, "ReconnectActivity", input).Get(ctx, nil)
				transition("LOGGING_ON")
			} else if state.State == "RECONNECTING" || state.State == "INITIALIZING" {
				transition("STREAMING")
			}
			livenessTimer = workflow.NewTimer(ctx, 30*time.Second)
		})

		selector.Select(ctx)

		if state.State == "STOPPED" {
			return nil
		}
	}
}

// HealthResult is the response shape for SessionLivenessCheckActivity.
type HealthResult struct {
	Healthy   bool      `json:"healthy"`
	CheckedAt time.Time `json:"checked_at"`
}

// LogonActivity calls POST /sessions/{id}/logon on the acceptor admin API.
// Note: on acceptors, this returns 501 (broker-driven logon). The
// workflow treats 501 as "no-op" — see workflow code above.
func LogonActivity(ctx context.Context, input FIXSessionLifecycleInput) error {
	url := fmt.Sprintf("%s/sessions/%s/logon", input.AdminURL, input.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Fix-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotImplemented {
		// 501: acceptor logon is broker-driven. Not an error.
		activity.GetLogger(ctx).Info("LogonActivity: 501 — broker-driven, no-op")
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logon: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// LogoutActivity calls POST /sessions/{id}/logout on the admin API.
func LogoutActivity(ctx context.Context, input FIXSessionLifecycleInput) error {
	url := fmt.Sprintf("%s/sessions/%s/logout", input.AdminURL, input.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Fix-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logout: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// SessionLivenessCheckActivity calls GET /sessions/{id}/health.
func SessionLivenessCheckActivity(ctx context.Context, input FIXSessionLifecycleInput) (HealthResult, error) {
	url := fmt.Sprintf("%s/sessions/%s/health", input.AdminURL, input.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return HealthResult{}, err
	}
	req.Header.Set("X-Fix-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return HealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}

	var parsed struct {
		Healthy   bool      `json:"healthy"`
		CheckedAt time.Time `json:"checked_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return HealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}
	return parsed, nil
}

// ReconnectActivity is invoked when liveness fails. Calls POST
// /sessions/{id}/logout then POST /sessions/{id}/logon (the latter
// 501s on acceptors — broker will re-establish on next inbound).
func ReconnectActivity(ctx context.Context, input FIXSessionLifecycleInput) error {
	if err := LogoutActivity(ctx, input); err != nil {
		activity.GetLogger(ctx).Warn("ReconnectActivity: logout failed", "err", err)
	}
	// Brief sleep so the broker side has time to detect the disconnect.
	time.Sleep(2 * time.Second)
	return LogonActivity(ctx, input)
}

// ensure bytes is referenced for future use (test fixtures, raw-byte
// payload construction); avoids the "imported and not used" lint if
// the imports list grows.
var _ = bytes.NewReader
