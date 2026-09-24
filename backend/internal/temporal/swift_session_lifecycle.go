package temporal

import (
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

type SWIFTChannelLifecycleInput struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	CustodianID uuid.UUID `json:"custodian_id"`
	AdminURL    string    `json:"admin_url"`
	AdminToken  string    `json:"admin_token"`
	ChannelID   string    `json:"channel_id"`
}

type SWIFTChannelState struct {
	State            string    `json:"state"`
	LastTransitionAt time.Time `json:"last_transition_at"`
	LastError        string    `json:"last_error,omitempty"`
}

func SWIFTChannelLifecycleWorkflow(ctx workflow.Context, input SWIFTChannelLifecycleInput) error {
	logger := workflow.GetLogger(ctx)

	state := SWIFTChannelState{
		State:            "INITIALIZING",
		LastTransitionAt: workflow.Now(ctx),
	}

	startSig := workflow.GetSignalChannel(ctx, "Start")
	stopSig := workflow.GetSignalChannel(ctx, "Stop")
	healthSig := workflow.GetSignalChannel(ctx, "HealthCheck")

	// Liveness timer: 60s (SWIFT sessions are less time-sensitive than FIX)
	livenessTimer := workflow.NewTimer(ctx, 60*time.Second)

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
		logger.Info("SWIFTChannelLifecycle: state transition", "from", state.State, "to", next)
		state.State = next
		state.LastTransitionAt = workflow.Now(ctx)
	}

	transition("CONNECTING")
	if err := workflow.ExecuteActivity(actx, SWIFTConnectActivity, input).Get(actx, nil); err != nil {
		state.LastError = fmt.Sprintf("connect: %v", err)
		transition("RECONNECTING")
	}

	for {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(startSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("SWIFTChannelLifecycle: Start signal received", "payload", sig)
		})

		selector.AddReceive(stopSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("SWIFTChannelLifecycle: Stop signal received", "payload", sig)
			transition("DISCONNECTING")
			_ = workflow.ExecuteActivity(actx, SWIFTDisconnectActivity, input).Get(ctx, nil)
			transition("STOPPED")
		})

		selector.AddReceive(healthSig, func(c workflow.ReceiveChannel, more bool) {
			var sig string
			c.Receive(ctx, &sig)
			logger.Info("SWIFTChannelLifecycle: HealthCheck signal received", "payload", sig)
		})

		selector.AddFuture(livenessTimer, func(f workflow.Future) {
			var healthResult SWIFTHealthResult
			_ = workflow.ExecuteActivity(actx, SWIFTChannelLivenessCheckActivity, input).Get(ctx, &healthResult)
			if !healthResult.Healthy {
				state.LastError = fmt.Sprintf("liveness check failed at %s", healthResult.CheckedAt)
				transition("RECONNECTING")
				_ = workflow.ExecuteActivity(actx, SWIFTReconnectActivity, input).Get(ctx, nil)
				transition("CONNECTING")
			} else if state.State == "RECONNECTING" || state.State == "INITIALIZING" {
				transition("ACTIVE")
			}
			livenessTimer = workflow.NewTimer(ctx, 60*time.Second)
		})

		selector.Select(ctx)

		if state.State == "STOPPED" {
			return nil
		}
	}
}

type SWIFTHealthResult struct {
	Healthy   bool      `json:"healthy"`
	CheckedAt time.Time `json:"checked_at"`
}

func SWIFTConnectActivity(ctx context.Context, input SWIFTChannelLifecycleInput) error {
	url := fmt.Sprintf("%s/channels/%s/connect", input.AdminURL, input.ChannelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Swift-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotImplemented {
		activity.GetLogger(ctx).Info("SWIFTConnectActivity: 501 - no-op")
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connect: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func SWIFTDisconnectActivity(ctx context.Context, input SWIFTChannelLifecycleInput) error {
	url := fmt.Sprintf("%s/channels/%s/disconnect", input.AdminURL, input.ChannelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Swift-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("disconnect: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func SWIFTChannelLivenessCheckActivity(ctx context.Context, input SWIFTChannelLifecycleInput) (SWIFTHealthResult, error) {
	url := fmt.Sprintf("%s/channels/%s/health", input.AdminURL, input.ChannelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return SWIFTHealthResult{}, err
	}
	req.Header.Set("X-Swift-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SWIFTHealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SWIFTHealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}

	var parsed struct {
		Healthy   bool      `json:"healthy"`
		CheckedAt time.Time `json:"checked_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return SWIFTHealthResult{Healthy: false, CheckedAt: time.Now().UTC()}, nil
	}
	return SWIFTHealthResult{Healthy: parsed.Healthy, CheckedAt: parsed.CheckedAt}, nil
}

func SWIFTReconnectActivity(ctx context.Context, input SWIFTChannelLifecycleInput) error {
	if err := SWIFTDisconnectActivity(ctx, input); err != nil {
		activity.GetLogger(ctx).Warn("SWIFTReconnectActivity: disconnect failed", "err", err)
	}
	time.Sleep(2 * time.Second)
	return SWIFTConnectActivity(ctx, input)
}
