package tiles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

type SenderConfig struct {
	AdminURL      string `json:"admin_url"` // "http://127.0.0.1:8982"
	AdminToken    string `json:"admin_token"`
	CustodianID   string `json:"custodian_id"`
	RetryAttempts int    `json:"retry_attempts"` // default 1
}

func NewSenderSink(cfg SenderConfig) TileFunc {
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = 1
	}

	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string

		for _, rec := range records {
			msgType, _ := rec["msg_type"].(string)

			var rawBytes []byte
			switch v := rec["outbound_raw"].(type) {
			case []byte:
				rawBytes = v
			case string:
				rawBytes = []byte(v)
			default:
				errs = append(errs, "sender: missing outbound_raw")
				continue
			}

			b64 := base64.StdEncoding.EncodeToString(rawBytes)
			hash := sha256.Sum256(rawBytes)
			idempKey := hex.EncodeToString(hash[:])

			payload := map[string]string{
				"msg_type": msgType,
				"raw":      b64,
			}
			body, _ := json.Marshal(payload)

			url := fmt.Sprintf("%s/channels/%s/send", cfg.AdminURL, cfg.CustodianID)

			success := false
			var lastErr error
			for i := 0; i < cfg.RetryAttempts; i++ {
				req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
				if err != nil {
					lastErr = err
					break
				}
				req.Header.Set("Content-Type", "application/json")
				if cfg.AdminToken != "" {
					req.Header.Set("Authorization", "Bearer "+cfg.AdminToken)
				}
				req.Header.Set("X-Idempotency-Key", idempKey)

				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						success = true
						break
					}
					lastErr = fmt.Errorf("status %d", resp.StatusCode)
				} else {
					lastErr = err
				}
			}

			if !success {
				errs = append(errs, fmt.Sprintf("sender: dispatch failed after %d attempts: %v", cfg.RetryAttempts, lastErr))
				continue
			}

			out = append(out, rec)
		}

		return out, errs, nil
	}
}
