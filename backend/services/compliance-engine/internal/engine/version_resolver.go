package engine

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// VersionResolver resolves which policy version to use based on trade date
type VersionResolver struct {
	db *sql.DB
}

// NewVersionResolver creates a new version resolver
func NewVersionResolver(db *sql.DB) *VersionResolver {
	return &VersionResolver{db: db}
}

// ResolveVersion determines which policy version applies for a given trade date
func (vr *VersionResolver) ResolveVersion(ctx context.Context, tradeDate time.Time, ruleType string) (string, error) {
	query := `
		SELECT version_tag 
		FROM compliance_policies
		WHERE rule_type = $1
		  AND effective_start_date <= $2
		  AND (effective_end_date IS NULL OR effective_end_date >= $2)
		ORDER BY effective_start_date DESC
		LIMIT 1
	`

	var versionTag string
	err := vr.db.QueryRowContext(ctx, query, ruleType, tradeDate).Scan(&versionTag)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback to latest version
			return vr.getLatestVersion(ctx, ruleType)
		}
		return "", fmt.Errorf("failed to resolve version: %w", err)
	}

	return versionTag, nil
}

// getLatestVersion returns the most recent policy version
func (vr *VersionResolver) getLatestVersion(ctx context.Context, ruleType string) (string, error) {
	query := `
		SELECT version_tag 
		FROM compliance_policies
		WHERE rule_type = $1
		ORDER BY effective_start_date DESC
		LIMIT 1
	`

	var versionTag string
	err := vr.db.QueryRowContext(ctx, query, ruleType).Scan(&versionTag)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no policies found for rule type %s", ruleType)
		}
		return "", fmt.Errorf("failed to get latest version: %w", err)
	}

	return versionTag, nil
}

// GetVersionForDate is a convenience method that parses a date string
func (vr *VersionResolver) GetVersionForDate(ctx context.Context, dateStr string, ruleType string) (string, error) {
	tradeDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	return vr.ResolveVersion(ctx, tradeDate, ruleType)
}
