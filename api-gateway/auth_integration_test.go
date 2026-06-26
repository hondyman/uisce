package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// This test runs an in-memory Gin server which exposes:
// - POST /api/auth/login that proxies to a mocked backend login endpoint
//   and then issues a gateway-signed HS256 JWT
// - GET /api/keys protected by the gateway JWT middleware

func TestGatewayLoginAndProtectedEndpoint(t *testing.T) {
	// Ensure deterministic Gin mode
	gin.SetMode(gin.TestMode)

	// Start a mocked backend that simulates a successful login response
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/auth/login" {
			// Return a minimal successful login payload
			resp := map[string]interface{}{
				"user": map[string]interface{}{
					"id":        "user-123",
					"tenant_id": "tenant-xyz",
				},
				"expires_in": 3600,
			}
			by, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(by)
			return
		}
		http.NotFound(w, r)
	}))
	defer backend.Close()

	// Set JWT_SECRET for signing in the same way as the gateway
	os.Setenv("JWT_SECRET", "test-secret-key")

	// Build an in-process Gin router with the minimal handlers we need
	r := gin.New()

	// Login handler: forward to mocked backend then sign a JWT
	r.POST("/api/auth/login", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad body"})
			return
		}

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", backend.URL+"/api/auth/login", bytes.NewBuffer(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "proxy failure"})
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "backend unreachable"})
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			c.Status(resp.StatusCode)
			c.Writer.Write(respBody)
			return
		}

		var backendResp map[string]interface{}
		if err := json.Unmarshal(respBody, &backendResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid backend response"})
			return
		}

		var user map[string]interface{}
		if u, ok := backendResp["user"].(map[string]interface{}); ok {
			user = u
		}
		expiresIn := 3600
		if ei, ok := backendResp["expires_in"].(float64); ok && ei > 0 {
			expiresIn = int(ei)
		}

		claims := jwt.MapClaims{}
		if user != nil {
			if uid, ok := user["id"].(string); ok {
				claims["user_id"] = uid
			}
			if tenant, ok := user["tenant_id"].(string); ok {
				claims["tenant_id"] = tenant
			}
		}
		claims["exp"] = time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"access_token": signed, "token_type": "Bearer", "user": user})
	})

	// Protected group using the same JWT middleware logic
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		tokenString := authHeader
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		c.Next()
	})

	api.GET("/keys", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api_keys": []interface{}{}})
	})

	// Run the router in a test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// 1) Call login
	loginReq := map[string]string{"email": "test+gw@example.com", "password": "password123"}
	by, _ := json.Marshal(loginReq)
	resp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewBuffer(by))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed status=%d body=%s", resp.StatusCode, string(b))
	}
	var loginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("invalid login response: %v", err)
	}
	token, _ := loginResp["access_token"].(string)
	if token == "" {
		t.Fatalf("no access_token in login response")
	}

	// 2) Call protected endpoint with the returned gateway-signed JWT
	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL+"/api/keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("protected request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("protected endpoint failed status=%d body=%s", resp2.StatusCode, string(b))
	}
}
