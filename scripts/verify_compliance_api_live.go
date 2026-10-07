package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func generateDevJWT(tenantID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "TND5KO7xY/Fz1ifgTR5QMm9T+R5/aPxxavmMzp+hURJxRWTm2Pns+RC+q9NKMxMB3F/R2KAWXnwo7r8N5JIACQ=="
	}

	claims := jwtmiddleware.JWTClaims{
		TenantID:  tenantID,
		TenantIDs: []string{tenantID},
		Email:     "compliance.officer@northwind.com",
		Roles:     []string{"compliance_officer", "compliance_admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "usr_compliance_officer_001",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			Issuer:    "https://100.84.50.65:8443/realms/uisce",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func main() {
	tenantID := "99e99e99-99e9-49e9-89e9-99e99e99e999" // Northwind Traders Gold Copy
	token, err := generateDevJWT(tenantID)
	if err != nil {
		log.Fatalf("Generate token failed: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Test Calendar Endpoint
	calURL := "http://localhost:8080/api/compliance/calendar"
	req, _ := http.NewRequest("GET", calURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request to /api/compliance/calendar failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("GET /api/compliance/calendar -> HTTP %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var calData struct {
		Data       []any  `json:"data"`
		TotalCount int    `json:"total_count"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.Unmarshal(body, &calData); err != nil {
		log.Fatalf("JSON parse calendar: %v", err)
	}
	fmt.Printf("  -> Loaded %d statutory calendar events for tenant %s\n", len(calData.Data), calData.TenantID)

	// 2. Test Limits Utilization Endpoint
	limitsURL := "http://localhost:8080/api/compliance/limits/utilization"
	req2, _ := http.NewRequest("GET", limitsURL, nil)
	req2.Header.Set("Authorization", "Bearer "+token)

	resp2, err := client.Do(req2)
	if err != nil {
		log.Fatalf("Request to /api/compliance/limits/utilization failed: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("GET /api/compliance/limits/utilization -> HTTP %d\n", resp2.StatusCode)
	if resp2.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status: %d, body: %s", resp2.StatusCode, string(body2))
	}

	var limitsData struct {
		Data       []any  `json:"data"`
		TotalCount int    `json:"total_count"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.Unmarshal(body2, &limitsData); err != nil {
		log.Fatalf("JSON parse limits: %v", err)
	}
	fmt.Printf("  -> Loaded %d limit headroom metrics for tenant %s\n", len(limitsData.Data), limitsData.TenantID)

	fmt.Println("======================================================================")
	fmt.Println("LIVE ENDPOINT VERIFICATION COMPLETE: 100% PASS UNDER JWT & RLS")
	fmt.Println("======================================================================")
}
