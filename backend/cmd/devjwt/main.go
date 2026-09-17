// Command devjwt forges HS256 JWTs accepted by the uisce API's
// AuthContextMiddleware. It is a development-only auth-bypass kit:
// anyone with JWT_SECRET and this binary can mint tokens for any
// tenant. Do not ship or invoke it against production.
//
// Scope: ENVIRONMENT must be development, local, or test; JWT_SECRET
// must be set (fail closed — no alternate config paths). Signing goes
// through services.SecurityManager.MintDevToken (the single backend
// forge path). Token is written to stdout only; secrets and errors go
// to stderr.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/services"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if err := requireDevEnvironment(); err != nil {
		return err
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is not set (refuse to invent a secret or read alternate config)")
	}
	if len(args) < 3 {
		return fmt.Errorf("usage: devjwt <user-id> <roles comma> <tenant-ids comma>")
	}

	sm := services.NewSecurityManager(nil, nil, []byte(secret))
	token, err := sm.MintDevToken(services.DevTokenInput{
		UserID:    strings.TrimSpace(args[0]),
		Roles:     splitList(args[1]),
		TenantIDs: splitList(args[2]),
	})
	if err != nil {
		return fmt.Errorf("mint: %w", err)
	}
	fmt.Println(token)
	return nil
}

func requireDevEnvironment() error {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	switch env {
	case "development", "local", "test":
		return nil
	case "":
		return fmt.Errorf("ENVIRONMENT is unset; set ENVIRONMENT=development|local|test to use devjwt")
	default:
		return fmt.Errorf("devjwt refused: ENVIRONMENT=%q is not development|local|test", env)
	}
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
