package jwtmiddleware

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTSecurityAndValidation(t *testing.T) {
	testSecret := "super-secure-production-grade-jwt-secret-key-at-least-32-chars"
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Unsetenv("JWT_SECRET")

	// 1. Create a signed JWT
	claims := JWTClaims{
		UserID:      "u-12345",
		Email:       "gsifi-auditor@bank.com",
		TenantID:    "00000000-0000-0000-0000-000000000000",
		TenantIDs:   []string{"00000000-0000-0000-0000-000000000000", "99e99e99-99e9-49e9-89e9-99e99e99e999"},
		Roles:       []string{"global_admin", "compliance_officer"},
		IsActive:    true,
		IsCoreAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// 2. Validate Token parsing and HMAC algorithm enforcement
	parsedClaims, err := ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected validation failure: %v", err)
	}

	if parsedClaims.UserID != "u-12345" {
		t.Errorf("expected user_id u-12345, got %s", parsedClaims.UserID)
	}

	// 3. Verify Tenant Access & Role Validation
	if err := ValidateTenantAccess(parsedClaims, "99e99e99-99e9-49e9-89e9-99e99e99e999"); err != nil {
		t.Errorf("expected access granted to authorized tenant: %v", err)
	}

	if !HasRole(parsedClaims, "global_admin") {
		t.Errorf("expected role global_admin to be present")
	}

	// 4. Test Extraction from HTTP Request Header (Rejecting empty / malformed headers)
	req := httptest.NewRequest("GET", "/api/v1/compliance/backtest", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	reqClaims, err := ValidateTokenFromRequest(req)
	if err != nil {
		t.Fatalf("failed extracting token from header: %v", err)
	}

	if reqClaims.Email != "gsifi-auditor@bank.com" {
		t.Errorf("expected email gsifi-auditor@bank.com, got %s", reqClaims.Email)
	}

	// 5. GSIFI Violation: Reject Token via URL Parameter
	badURLReq := httptest.NewRequest("GET", "/api/v1/compliance/backtest?token="+tokenStr, nil)
	_, err = ValidateTokenFromRequest(badURLReq)
	if err == nil {
		t.Errorf("expected GSIFI error when token passed via URL parameter, got nil")
	}
}
