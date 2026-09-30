package mdm

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleResolver_ResolveEntityRules(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resolver := NewRuleResolver(db)
	ctx := context.Background()

	clientTenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	coreTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	termAccountName := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	termDomicile := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	termStatus := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	t.Run("per-attribute merge with core fallback and tenant override", func(t *testing.T) {
		// Mock core tenant lookup
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true LIMIT 1`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(coreTenantID))

		// Mock rules query
		ruleCols := []string{
			"id", "tenant_id", "entity_type", "semantic_term_id",
			"attribute_name", "strategy", "priority_order", "max_stale_seconds", "is_active",
		}

		rows := sqlmock.NewRows(ruleCols).
			// Core baseline rules (account_name, domicile, status)
			AddRow(uuid.New(), coreTenantID, "ACCOUNT", termAccountName, "account_name", "SOURCE_PRIORITY", pq.StringArray{"GOLDENSOURCE", "INTERNAL"}, 86400, true).
			AddRow(uuid.New(), coreTenantID, "ACCOUNT", termDomicile, "domicile", "SOURCE_PRIORITY", pq.StringArray{"MARKET_EDM", "BLOOMBERG"}, 43200, true).
			AddRow(uuid.New(), coreTenantID, "ACCOUNT", termStatus, "status_cd", "MOST_RECENT", pq.StringArray{}, 0, true).
			// Tenant delta override (only for domicile)
			AddRow(uuid.New(), clientTenantID, "ACCOUNT", termDomicile, "domicile", "SOURCE_PRIORITY", pq.StringArray{"BLOOMBERG", "REFINITIV"}, 7200, true)

		mock.ExpectQuery(`SELECT (.+) FROM public\.semantic_survivorship_rules r`).
			WithArgs("ACCOUNT", clientTenantID, coreTenantID).
			WillReturnRows(rows)

		rules, err := resolver.ResolveEntityRules(ctx, clientTenantID, "ACCOUNT")
		require.NoError(t, err)
		require.Len(t, rules, 3)

		// 1. account_name should be inherited from Core
		accRule := rules["account_name"]
		assert.Equal(t, ResolutionTierCoreInherited, accRule.ResolutionTier)
		assert.Equal(t, "SOURCE_PRIORITY", accRule.Strategy)
		assert.Equal(t, []string{"GOLDENSOURCE", "INTERNAL"}, accRule.PriorityOrder)
		assert.Equal(t, 86400, accRule.MaxStaleSeconds)

		// 2. domicile should be overridden by Client Tenant
		domRule := rules["domicile"]
		assert.Equal(t, ResolutionTierTenantOverride, domRule.ResolutionTier)
		assert.Equal(t, "SOURCE_PRIORITY", domRule.Strategy)
		assert.Equal(t, []string{"BLOOMBERG", "REFINITIV"}, domRule.PriorityOrder)
		assert.Equal(t, 7200, domRule.MaxStaleSeconds)

		// 3. status_cd should be inherited from Core
		statusRule := rules["status_cd"]
		assert.Equal(t, ResolutionTierCoreInherited, statusRule.ResolutionTier)
		assert.Equal(t, "MOST_RECENT", statusRule.Strategy)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("core tenant self-resolution short-circuits", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id FROM public\.tenants WHERE gold_copy = true LIMIT 1`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(coreTenantID))

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "entity_type", "semantic_term_id",
			"attribute_name", "strategy", "priority_order", "max_stale_seconds", "is_active",
		}).AddRow(uuid.New(), coreTenantID, "ACCOUNT", termAccountName, "account_name", "SOURCE_PRIORITY", pq.StringArray{"GOLDENSOURCE"}, 86400, true)

		mock.ExpectQuery(`SELECT (.+) FROM public\.semantic_survivorship_rules r (.+) WHERE r\.entity_type = \$1 AND r\.is_active = true AND r\.tenant_id = \$2`).
			WithArgs("ACCOUNT", coreTenantID).
			WillReturnRows(rows)

		rules, err := resolver.ResolveEntityRules(ctx, coreTenantID, "ACCOUNT")
		require.NoError(t, err)
		require.Len(t, rules, 1)
		assert.Equal(t, ResolutionTierCoreInherited, rules["account_name"].ResolutionTier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation failures", func(t *testing.T) {
		_, err := resolver.ResolveEntityRules(ctx, uuid.Nil, "ACCOUNT")
		assert.Error(t, err)

		_, err = resolver.ResolveEntityRules(ctx, clientTenantID, "")
		assert.Error(t, err)
	})
}
