package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestWebSocketToken_MintAndValidateUnit(t *testing.T) {
	srv := &Server{}

	// 1. Unset WS_TOKEN_SECRET in prod -> fails hard
	t.Run("Missing WS_TOKEN_SECRET outside dev fails", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("WS_TOKEN_SECRET", "")

		body := bytes.NewBufferString(`{"jobId": "job-1", "purpose": "profiler", "ttl_seconds": 60}`)
		req := httptest.NewRequest("POST", "/api/ws/token", body)
		w := httptest.NewRecorder()

		srv.getWsToken(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 2. Mint with WS_TOKEN_SECRET and validate successfully
	t.Run("Mint and validate with dedicated secret", func(t *testing.T) {
		secret := "test-ws-token-secret-very-secure-32b"
		t.Setenv("APP_ENV", "production")
		t.Setenv("WS_TOKEN_SECRET", secret)

		body := bytes.NewBufferString(`{"jobId": "job-abc", "purpose": "profiler", "ttl_seconds": 120}`)
		req := httptest.NewRequest("POST", "/api/ws/token", body)
		w := httptest.NewRecorder()

		srv.getWsToken(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		tokenStr := resp["token"]
		require.NotEmpty(t, tokenStr)

		// Validate with matching jobId
		claims, err := srv.validateWsToken(tokenStr, "job-abc")
		require.NoError(t, err)
		require.Equal(t, "job-abc", claims["job_id"])
		require.Equal(t, "profiler", claims["purpose"])

		// Validate with mismatched jobId -> fails
		_, err = srv.validateWsToken(tokenStr, "wrong-job")
		require.Error(t, err)
		require.Contains(t, err.Error(), "token not valid for job_id")
	})

	// 3. Token signed with old/leaked JWT_SECRET -> rejected
	t.Run("Token signed with different secret is rejected", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("WS_TOKEN_SECRET", "the-real-ws-secret")

		// Hand-mint token signed with an old/different secret
		claims := jwt.MapClaims{
			"job_id":  "job-spoof",
			"purpose": "profiler",
			"iat":     time.Now().Unix(),
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		forgedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := forgedToken.SignedString([]byte("leaked-jwt-secret-value"))
		require.NoError(t, err)

		_, err = srv.validateWsToken(signed, "job-spoof")
		require.Error(t, err)
		require.Contains(t, err.Error(), "signature is invalid")
	})
}
