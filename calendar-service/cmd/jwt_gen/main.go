package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims matches the middleware claims structure
type JWTClaims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	TenantID  string   `json:"tenant_id"`
	TenantIDs []string `json:"tenant_ids"`
	jwt.RegisteredClaims
}

func main() {
	secret := "dev-jwt-secret-key-change-in-production"
	userID := "test-user-phase5-2"
	tenantID := "test-tenant"

	if len(os.Args) > 1 {
		secret = os.Args[1]
	}
	if len(os.Args) > 2 {
		userID = os.Args[2]
	}
	if len(os.Args) > 3 {
		tenantID = os.Args[3]
	}

	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     userID + "@example.com",
		TenantID:  tenantID,
		TenantIDs: []string{tenantID},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(tokenString)
}
