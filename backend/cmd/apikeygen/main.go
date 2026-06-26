package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type apiKeyFileEntry struct {
	Key       string   `json:"key"`
	UserID    string   `json:"user_id"`
	TenantIDs []string `json:"tenant_ids,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

func main() {
	userID := flag.String("user", "", "user id for the API key")
	rolesArg := flag.String("roles", "", "comma-separated roles")
	tenantsArg := flag.String("tenants", "", "comma-separated tenant ids")
	filePath := flag.String("file", defaultKeyFilePath(), "path to api key file")
	dbDSN := flag.String("db", "", "database connection string (uses DATABASE_URL if empty)")
	name := flag.String("name", "", "optional key name")
	description := flag.String("description", "", "optional description")
	createdBy := flag.String("created-by", "", "creator user id (UUID for DB mode)")
	expiresAt := flag.String("expires", "", "expiration timestamp (RFC3339)")
	flag.Parse()

	if strings.TrimSpace(*userID) == "" {
		fmt.Println("usage: apikeygen -user <user-id> -roles role1,role2 -tenants t1,t2 [-file path]")
		os.Exit(1)
	}

	roles := splitList(*rolesArg)
	tenantIDs := splitList(*tenantsArg)
	if len(tenantIDs) == 0 {
		fmt.Println("error: at least one tenant id is required")
		os.Exit(1)
	}

	if key, ok := createKeyInDatabase(*dbDSN, strings.TrimSpace(*userID), *createdBy, *name, *description, roles, tenantIDs, *expiresAt); ok {
		fmt.Println(key)
		return
	}

	key, err := generateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate key: %v\n", err)
		os.Exit(1)
	}

	entries, err := readKeyFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read key file: %v\n", err)
		os.Exit(1)
	}

	entries = append(entries, apiKeyFileEntry{
		Key:       key,
		UserID:    strings.TrimSpace(*userID),
		TenantIDs: tenantIDs,
		Roles:     roles,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if err := writeKeyFile(*filePath, entries); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write key file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(key)
}

func defaultKeyFilePath() string {
	if value := strings.TrimSpace(os.Getenv("API_KEYS_FILE")); value != "" {
		return value
	}
	return "config/api_keys.json"
}

func generateKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
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

func readKeyFile(path string) ([]apiKeyFileEntry, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []apiKeyFileEntry{}, nil
		}
		return nil, err
	}
	if len(payload) == 0 {
		return []apiKeyFileEntry{}, nil
	}

	entries := []apiKeyFileEntry{}
	if err := json.Unmarshal(payload, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func writeKeyFile(path string, entries []apiKeyFileEntry) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}

	payload, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o600)
}

func createKeyInDatabase(dsn, userID, createdBy, name, description string, roles []string, tenantIDs []string, expiresAt string) (string, bool) {
	resolvedDSN := strings.TrimSpace(dsn)
	if resolvedDSN == "" {
		resolvedDSN = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if resolvedDSN == "" {
		return "", false
	}

	expiration, err := parseExpiresAt(expiresAt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid expires value: %v\n", err)
		os.Exit(1)
	}

	creator := strings.TrimSpace(createdBy)
	if creator == "" {
		creator = strings.TrimSpace(userID)
	}
	if creator == "" {
		fmt.Fprintln(os.Stderr, "created-by is required for DB mode")
		os.Exit(1)
	}

	db, err := sqlx.Connect("pgx", resolvedDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	store := services.NewDBAPIKeyStore(db)
	key, _, err := store.CreateKey(context.Background(), services.APIKeyCreateRequest{
		UserID:      strings.TrimSpace(userID),
		TenantIDs:   tenantIDs,
		Roles:       roles,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		CreatedBy:   creator,
		ExpiresAt:   expiration,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create api key: %v\n", err)
		os.Exit(1)
	}

	return key, true
}

func parseExpiresAt(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
