package interpreter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.temporal.io/sdk/activity"
)

type InterpreterActivities struct {
	client *http.Client
}

func NewInterpreterActivities() *InterpreterActivities {
	return &InterpreterActivities{
		client: &http.Client{},
	}
}

// HTTPArgs defines the arguments for the generic HTTP activity
type HTTPArgs struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

// ExecuteHTTP is a generic activity to make HTTP requests
func (a *InterpreterActivities) ExecuteHTTP(ctx context.Context, args HTTPArgs) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing HTTP Activity", "url", args.URL, "method", args.Method)

	var bodyReader io.Reader
	if args.Body != nil {
		jsonBody, err := json.Marshal(args.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, args.Method, args.URL, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}
	if args.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	// Try to parse JSON, otherwise return generic text
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = map[string]interface{}{
			"raw_response": string(respBody),
		}
	}
	
	result["status_code"] = resp.StatusCode
	return result, nil
}

// LogMessage is a simple activity for debugging or audit
func (a *InterpreterActivities) LogMessage(ctx context.Context, message string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Workflow Log", "message", message)
	return nil
}
