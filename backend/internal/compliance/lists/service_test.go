package lists

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
)

func TestListService_IngestAndVersionSnapshot(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	defer db.Close()

	svc := NewListService(db)
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO public.tenants (id, name, display_name) VALUES ($1, 'test_tenant', 'Test Tenant')", tenantID)
	require.NoError(t, err)

	req1 := IngestListRequest{
		TenantID:       tenantID,
		ListCode:       "RESTRICTED_MNPI",
		ListName:       "Restricted MNPI Trading List",
		ListType:       ListTypeRestrictedTrading,
		SourceFeedName: "LEGAL_PORTAL",
		Items: []RestrictedListItem{
			{
				IssuerID:          "ISS-AAPL",
				IssuerName:        "Apple Inc.",
				MatchKey:          "AAPL",
				MatchType:         MatchTypeExact,
				RestrictionReason: "Project Titan M&A Deal",
				RestrictionScope:  RestrictionScopeAllTrading,
			},
			{
				IssuerID:          "ISS-MSFT",
				IssuerName:        "Microsoft Corp.",
				MatchKey:          "MSFT",
				MatchType:         MatchTypeExact,
				RestrictionReason: "Q3 Earnings Quiet Period",
				RestrictionScope:  RestrictionScopeAllTrading,
			},
		},
	}

	list1, err := svc.IngestList(ctx, req1)
	require.NoError(t, err)
	require.Equal(t, 1, list1.Version)
	require.Equal(t, 2, list1.ItemCount)
	require.NotEmpty(t, list1.ContentHash)

	// Re-ingest with additional item -> Version 2 with distinct content hash
	req2 := req1
	req2.Items = append(req2.Items, RestrictedListItem{
		IssuerID:          "ISS-NVDA",
		IssuerName:        "NVIDIA Corp.",
		MatchKey:          "NVDA",
		MatchType:         MatchTypeExact,
		RestrictionReason: "Confidential Supplier Talks",
		RestrictionScope:  RestrictionScopeAllTrading,
	})

	list2, err := svc.IngestList(ctx, req2)
	require.NoError(t, err)
	require.Equal(t, 2, list2.Version)
	require.Equal(t, 3, list2.ItemCount)
	require.NotEqual(t, list1.ContentHash, list2.ContentHash)
}

func TestListService_ExactAndAliasMatching(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	defer db.Close()

	svc := NewListService(db)
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO public.tenants (id, name, display_name) VALUES ($1, 'test_tenant_2', 'Test Tenant 2')", tenantID)
	require.NoError(t, err)

	req := IngestListRequest{
		TenantID:       tenantID,
		ListCode:       "SANCTIONS_OFAC",
		ListName:       "OFAC Sanctions List",
		ListType:       ListTypeSanctionsLookthrough,
		SourceFeedName: "OFAC_SDN_FEED",
		Items: []RestrictedListItem{
			{
				SecurityID:        "SEC-SANCTION-1",
				IssuerID:          "ISS-SANCTION-1",
				IssuerName:        "Sanctioned Energy Bank",
				MatchKey:          "SEB_ALIAS_HOLDING",
				MatchType:         MatchTypeAlias,
				RestrictionReason: "Executive Order 14024",
				RestrictionScope:  RestrictionScopeAllTrading,
			},
		},
	}

	_, err = svc.IngestList(ctx, req)
	require.NoError(t, err)

	// 1. Match by SecurityID
	item, matched, err := svc.CheckRestriction(ctx, tenantID, "SANCTIONS_OFAC", "SEC-SANCTION-1", "", "")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, "ISS-SANCTION-1", item.IssuerID)

	// 2. Match by IssuerID
	item, matched, err = svc.CheckRestriction(ctx, tenantID, "SANCTIONS_OFAC", "", "ISS-SANCTION-1", "")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, "Sanctioned Energy Bank", item.IssuerName)

	// 3. Match by Case-Insensitive Alias MatchKey
	item, matched, err = svc.CheckRestriction(ctx, tenantID, "SANCTIONS_OFAC", "", "", "seb_alias_holding")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, "Executive Order 14024", item.RestrictionReason)

	// 4. Non-matching clean entity
	_, matched, err = svc.CheckRestriction(ctx, tenantID, "SANCTIONS_OFAC", "SEC-CLEAN", "ISS-CLEAN", "CLEAN_TICKER")
	require.NoError(t, err)
	require.False(t, matched)
}

func TestListService_TemporalExpiration(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	defer db.Close()

	svc := NewListService(db)
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO public.tenants (id, name, display_name) VALUES ($1, 'test_tenant_3', 'Test Tenant 3')", tenantID)
	require.NoError(t, err)

	expiredTime := time.Now().UTC().Add(-2 * time.Hour)
	req := IngestListRequest{
		TenantID:       tenantID,
		ListCode:       "DEAL_BLACKOUT",
		ListName:       "M&A Deal Blackout List",
		ListType:       ListTypeRestrictedTrading,
		SourceFeedName: "DEAL_SYSTEM",
		Items: []RestrictedListItem{
			{
				IssuerID:          "ISS-EXPIRED-DEAL",
				IssuerName:        "Target Co M&A",
				RestrictionReason: "Deal Announced / Window Closed",
				RestrictionScope:  RestrictionScopeAllTrading,
				EffectiveFrom:     time.Now().UTC().Add(-24 * time.Hour),
				EffectiveTo:       &expiredTime, // Expired 2 hours ago
			},
		},
	}

	_, err = svc.IngestList(ctx, req)
	require.NoError(t, err)

	// Lookup must return false because the restriction window expired
	_, matched, err := svc.CheckRestriction(ctx, tenantID, "DEAL_BLACKOUT", "", "ISS-EXPIRED-DEAL", "")
	require.NoError(t, err)
	require.False(t, matched, "Expired restricted item must not match active trading check")
}

func TestListService_StalenessDetection(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	defer db.Close()

	svc := NewListService(db)
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO public.tenants (id, name, display_name) VALUES ($1, 'test_tenant_4', 'Test Tenant 4')", tenantID)
	require.NoError(t, err)

	// Ingest a list
	req := IngestListRequest{
		TenantID:       tenantID,
		ListCode:       "SANCTIONS_WATCH",
		ListName:       "Daily Sanctions Watchlist",
		ListType:       ListTypeSanctionsLookthrough,
		SourceFeedName: "VENDOR_DAILY",
		Items:          []RestrictedListItem{},
	}
	list, err := svc.IngestList(ctx, req)
	require.NoError(t, err)

	// Back-date last_refreshed_at by 30 hours
	staleTime := time.Now().UTC().Add(-30 * time.Hour)
	_, err = db.ExecContext(ctx, "UPDATE compliance.compliance_restricted_list SET last_refreshed_at = $1 WHERE id = $2", staleTime, list.ID)
	require.NoError(t, err)

	// Check staleness with 24h threshold -> must alert
	alerts, err := svc.CheckListStaleness(ctx, tenantID, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	require.Equal(t, "SANCTIONS_WATCH", alerts[0].ListCode)
	require.GreaterOrEqual(t, alerts[0].AgeHours, 29.9)
	require.Contains(t, alerts[0].Message, "is stale")
}
