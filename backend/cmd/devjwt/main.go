package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: devjwt <user-id> <roles comma> <tenant-ids comma>")
		os.Exit(1)
	}

	userID := strings.TrimSpace(os.Args[1])
	roles := splitList(os.Args[2])
	tenantIDs := splitList(os.Args[3])

	claims := jwt.MapClaims{
		"sub":        userID,
		"roles":      roles,
		"tenant_ids": tenantIDs,
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-jwt-secret-key-change-in-production"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to sign token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(signed)
}

func splitList(raw string) []string {
	items := strings.Split(raw, ",")
	result := []string{}
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
