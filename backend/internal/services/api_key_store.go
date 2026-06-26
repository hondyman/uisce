package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type APIKeyStore interface {
	FindByKey(ctx context.Context, rawKey string) (*APIKey, error)
	CreateKey(ctx context.Context, req APIKeyCreateRequest) (string, *APIKey, error)
}

type APIKeyCreateRequest struct {
	UserID      string
	TenantID    string
	TenantIDs   []string
	Roles       []string
	Name        string
	Description string
	CreatedBy   string
	ExpiresAt   *time.Time
}

type DBAPIKeyStore struct {
	db *sqlx.DB
}

func NewDBAPIKeyStore(db *sqlx.DB) *DBAPIKeyStore {
	return &DBAPIKeyStore{db: db}
}

func (s *DBAPIKeyStore) FindByKey(ctx context.Context, rawKey string) (*APIKey, error) {
	trimmed := strings.TrimSpace(rawKey)
	if trimmed == "" {
		return nil, sql.ErrNoRows
	}

	hash := sha256.Sum256([]byte(trimmed))
	keyHash := hex.EncodeToString(hash[:])

	var userID, tenantID string
	roles := []string{}
	tenantIDs := []string{}
	var isActive bool
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime

	query := `
		SELECT user_id, tenant_id, roles, tenant_ids, is_active, expires_at, last_used_at
		FROM public.api_keys
		WHERE key_hash = $1
	`
	if err := s.db.QueryRowxContext(ctx, query, keyHash).Scan(
		&userID,
		&tenantID,
		pq.Array(&roles),
		pq.Array(&tenantIDs),
		&isActive,
		&expiresAt,
		&lastUsedAt,
	); err != nil {
		return nil, err
	}

	if !isActive {
		return nil, sql.ErrNoRows
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return nil, sql.ErrNoRows
	}

	if err := s.touchLastUsed(ctx, keyHash); err != nil {
		return nil, err
	}

	apiKey := &APIKey{
		Key:       trimmed,
		UserID:    strings.TrimSpace(userID),
		TenantID:  strings.TrimSpace(tenantID),
		TenantIDs: normalizeStringList(tenantIDs),
		Roles:     normalizeStringList(roles),
		Active:    true,
	}
	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		apiKey.LastUsedAt = &lastUsedAt.Time
	}

	return apiKey, nil
}

func (s *DBAPIKeyStore) CreateKey(ctx context.Context, req APIKeyCreateRequest) (string, *APIKey, error) {
	if s == nil || s.db == nil {
		return "", nil, fmt.Errorf("api key store not configured")
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		return "", nil, fmt.Errorf("created_by is required")
	}
	if _, err := uuid.Parse(createdBy); err != nil {
		return "", nil, fmt.Errorf("created_by must be a valid UUID")
	}

	tenantID := strings.TrimSpace(req.TenantID)
	tenantIDs := normalizeStringList(req.TenantIDs)
	if tenantID == "" && len(tenantIDs) > 0 {
		tenantID = tenantIDs[0]
	}
	if tenantID == "" {
		return "", nil, fmt.Errorf("tenant_id is required")
	}
	if len(tenantIDs) == 0 {
		tenantIDs = []string{tenantID}
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("key-%s", time.Now().UTC().Format("20060102-150405"))
	}

	key := generateSecureKey()
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	roles := normalizeStringList(req.Roles)
	permissions := map[string]interface{}{
		"roles":      roles,
		"tenant_ids": tenantIDs,
	}
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return "", nil, err
	}

	description := strings.TrimSpace(req.Description)
	descriptionValue := sql.NullString{String: description, Valid: description != ""}

	query := `
		INSERT INTO public.api_keys
			(tenant_id, name, key_hash, description, permissions, expires_at, is_active, created_by, user_id, roles, tenant_ids)
		VALUES
			($1, $2, $3, $4, $5, $6, true, $7, $8, $9, $10)
	`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		tenantID,
		name,
		keyHash,
		descriptionValue,
		permissionsJSON,
		req.ExpiresAt,
		createdBy,
		strings.TrimSpace(req.UserID),
		pq.Array(roles),
		pq.Array(tenantIDs),
	); err != nil {
		return "", nil, err
	}

	apiKey := &APIKey{
		Key:       key,
		UserID:    strings.TrimSpace(req.UserID),
		TenantID:  tenantID,
		TenantIDs: tenantIDs,
		Roles:     roles,
		Active:    true,
		ExpiresAt: req.ExpiresAt,
	}

	return key, apiKey, nil
}

func (s *DBAPIKeyStore) touchLastUsed(ctx context.Context, keyHash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE public.api_keys SET last_used_at = NOW() WHERE key_hash = $1`, keyHash)
	return err
}

func normalizeStringList(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
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
