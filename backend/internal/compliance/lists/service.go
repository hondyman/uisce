package lists

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IngestListRequest contains the parameters to create or refresh a restricted list.
type IngestListRequest struct {
	TenantID       uuid.UUID            `json:"tenant_id"`
	ListCode       string               `json:"list_code"`
	ListName       string               `json:"list_name"`
	ListType       ListType             `json:"list_type"`
	SourceFeedName string               `json:"source_feed_name"`
	Items          []RestrictedListItem `json:"items"`
}

// ListService manages the ingestion, versioning, temporal lifecycle, and restriction lookups for Class B lists.
type ListService struct {
	db *sql.DB
}

// NewListService constructs a ListService backed by Postgres.
func NewListService(db *sql.DB) *ListService {
	return &ListService{db: db}
}

// IngestList processes a versioned list ingestion in a transaction, generating a canonical content hash.
func (s *ListService) IngestList(ctx context.Context, req IngestListRequest) (*RestrictedList, error) {
	if req.ListCode == "" || req.ListName == "" {
		return nil, fmt.Errorf("list_code and list_name are required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Fence tenant context for RLS
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", req.TenantID.String())
	if err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	// 1. Sort items deterministically to produce an immutable content hash
	sortedItems := append([]RestrictedListItem(nil), req.Items...)
	sort.Slice(sortedItems, func(i, j int) bool {
		if sortedItems[i].IssuerID != sortedItems[j].IssuerID {
			return sortedItems[i].IssuerID < sortedItems[j].IssuerID
		}
		if sortedItems[i].SecurityID != sortedItems[j].SecurityID {
			return sortedItems[i].SecurityID < sortedItems[j].SecurityID
		}
		return sortedItems[i].MatchKey < sortedItems[j].MatchKey
	})

	hasher := sha256.New()
	for _, it := range sortedItems {
		hasher.Write([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s;",
			it.IssuerID, it.SecurityID, it.IssuerName, it.MatchKey, it.MatchType, it.RestrictionScope, it.RestrictionReason)))
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// 2. Upsert Restricted List header
	var list RestrictedList
	feedName := req.SourceFeedName
	if feedName == "" {
		feedName = "MANUAL_IMPORT"
	}

	now := time.Now().UTC()
	err = tx.QueryRowContext(ctx, `
		INSERT INTO compliance.compliance_restricted_list (
			tenant_id, list_code, list_name, list_type, is_active, version,
			source_feed_name, last_refreshed_at, item_count, content_hash, updated_at
		) VALUES (
			$1, $2, $3, $4, true, 1,
			$5, $6, $7, $8, $6
		)
		ON CONFLICT (tenant_id, list_code) DO UPDATE SET
			list_name = EXCLUDED.list_name,
			list_type = EXCLUDED.list_type,
			is_active = true,
			version = compliance.compliance_restricted_list.version + 1,
			source_feed_name = EXCLUDED.source_feed_name,
			last_refreshed_at = EXCLUDED.last_refreshed_at,
			item_count = EXCLUDED.item_count,
			content_hash = EXCLUDED.content_hash,
			updated_at = EXCLUDED.updated_at
		RETURNING id, tenant_id, list_code, list_name, list_type, is_active, version,
		          source_feed_name, last_refreshed_at, item_count, content_hash, created_at, updated_at
	`, req.TenantID, req.ListCode, req.ListName, string(req.ListType),
		feedName, now, len(sortedItems), contentHash,
	).Scan(
		&list.ID, &list.TenantID, &list.ListCode, &list.ListName, &list.ListType,
		&list.IsActive, &list.Version, &list.SourceFeedName, &list.LastRefreshedAt,
		&list.ItemCount, &list.ContentHash, &list.CreatedAt, &list.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert restricted list: %w", err)
	}

	// 3. Clear existing active items for this list and insert newly ingested snapshot
	_, err = tx.ExecContext(ctx, "DELETE FROM compliance.compliance_restricted_list_item WHERE list_id = $1", list.ID)
	if err != nil {
		return nil, fmt.Errorf("clear existing list items: %w", err)
	}

	for _, it := range sortedItems {
		matchType := it.MatchType
		if matchType == "" {
			matchType = MatchTypeExact
		}
		scope := it.RestrictionScope
		if scope == "" {
			scope = RestrictionScopeAllTrading
		}
		effFrom := it.EffectiveFrom
		if effFrom.IsZero() {
			effFrom = now
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_restricted_list_item (
				id, tenant_id, list_id, security_id, issuer_id, issuer_name,
				match_key, match_type, restriction_reason, restriction_scope,
				effective_from, effective_to, created_at
			) VALUES (
				gen_random_uuid(), $1, $2, NULLIF($3, ''), $4, $5,
				$6, $7, $8, $9,
				$10, $11, now()
			)
		`, req.TenantID, list.ID, it.SecurityID, it.IssuerID, it.IssuerName,
			it.MatchKey, string(matchType), it.RestrictionReason, string(scope),
			effFrom, it.EffectiveTo,
		)
		if err != nil {
			return nil, fmt.Errorf("insert list item for issuer %s: %w", it.IssuerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &list, nil
}

// CheckRestriction looks up whether a target security, issuer, or match key is currently restricted.
func (s *ListService) CheckRestriction(
	ctx context.Context,
	tenantID uuid.UUID,
	listCode string,
	securityID string,
	issuerID string,
	matchKey string,
) (*RestrictedListItem, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT 
			i.id, i.tenant_id, i.list_id, COALESCE(i.security_id, ''), i.issuer_id, COALESCE(i.issuer_name, ''),
			COALESCE(i.match_key, ''), i.match_type, i.restriction_reason, i.restriction_scope,
			i.effective_from, i.effective_to, i.created_at
		FROM compliance.compliance_restricted_list_item i
		JOIN compliance.compliance_restricted_list l ON i.list_id = l.id
		WHERE (l.tenant_id = $1 OR l.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid)
		  AND l.list_code = $2
		  AND l.is_active = true
		  AND i.effective_from <= now()
		  AND (i.effective_to IS NULL OR i.effective_to > now())
		  AND (
		      (i.issuer_id != '' AND i.issuer_id = $3)
		      OR (i.security_id != '' AND i.security_id = $4)
		      OR (i.match_key != '' AND LOWER(i.match_key) = LOWER($5))
		  )
		ORDER BY i.created_at DESC
		LIMIT 1
	`, tenantID, listCode, issuerID, securityID, strings.TrimSpace(matchKey))

	var item RestrictedListItem
	var effTo sql.NullTime
	err := row.Scan(
		&item.ID, &item.TenantID, &item.ListID, &item.SecurityID, &item.IssuerID, &item.IssuerName,
		&item.MatchKey, &item.MatchType, &item.RestrictionReason, &item.RestrictionScope,
		&item.EffectiveFrom, &effTo, &item.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("check restriction: %w", err)
	}
	if effTo.Valid {
		item.EffectiveTo = &effTo.Time
	}

	return &item, true, nil
}

// CheckListStaleness detects active lists that have exceeded the allowable refresh age.
func (s *ListService) CheckListStaleness(
	ctx context.Context,
	tenantID uuid.UUID,
	maxAge time.Duration,
) ([]StaleListAlert, error) {
	threshold := time.Now().UTC().Add(-maxAge)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, list_code, list_name, last_refreshed_at,
		       EXTRACT(EPOCH FROM (now() - last_refreshed_at)) / 3600.0 AS age_hours
		FROM compliance.compliance_restricted_list
		WHERE (tenant_id = $1 OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid)
		  AND is_active = true
		  AND last_refreshed_at < $2
		ORDER BY last_refreshed_at ASC
	`, tenantID, threshold)
	if err != nil {
		return nil, fmt.Errorf("query stale lists: %w", err)
	}
	defer rows.Close()

	var alerts []StaleListAlert
	for rows.Next() {
		var a StaleListAlert
		if err := rows.Scan(&a.ListID, &a.ListCode, &a.ListName, &a.LastRefreshedAt, &a.AgeHours); err != nil {
			return nil, fmt.Errorf("scan stale list: %w", err)
		}
		a.Severity = "WARNING"
		a.Message = fmt.Sprintf("Restricted list %s (%s) is stale: last refreshed %.1f hours ago (threshold: %.1f hours)",
			a.ListCode, a.ListName, a.AgeHours, maxAge.Hours())
		alerts = append(alerts, a)
	}

	return alerts, nil
}
