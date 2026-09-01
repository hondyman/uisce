package calcengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestValidateIdentifier_RejectsInjectionPayloads locks down the allowlist
// used before tableName/tenantID/datasourceID are interpolated into SQL via
// fmt.Sprintf, so a caller-influenceable value (tenantID/datasourceID in
// particular come from HTTP request context / JWT claims upstream) can't
// break out of the surrounding SQL text.
func TestValidateIdentifier_RejectsInjectionPayloads(t *testing.T) {
	malicious := []string{
		`positions'; DROP TABLE positions; --`,
		`positions" OR "1"="1`,
		"positions; DELETE FROM positions",
		"positions/*",
		"positions ",
		"",
		"tenant$1",
		"tenant.name",
	}
	for _, v := range malicious {
		t.Run(v, func(t *testing.T) {
			err := validateIdentifier("table_name", v)
			assert.Error(t, err, "expected %q to be rejected", v)
		})
	}
}

func TestValidateIdentifier_AllowsLegitimateValues(t *testing.T) {
	legit := []string{"positions", "tenant-alpha", "ds-postgres", "semantic_hot", "tenant_1", "a"}
	for _, v := range legit {
		t.Run(v, func(t *testing.T) {
			assert.NoError(t, validateIdentifier("table_name", v))
		})
	}
}

// TestBuildSafeQuery_RejectsMaliciousIdentifiers ensures the injection guard
// actually sits on the SQL-building path, not just the standalone helper.
func TestBuildSafeQuery_RejectsMaliciousIdentifiers(t *testing.T) {
	manager, mock := setupManager(t, PostgresQueryDialect{})

	cases := []struct {
		name string
		q    *TierQuery
	}{
		{
			name: "malicious table name",
			q: &TierQuery{
				TableName:    `positions'; DROP TABLE positions; --`,
				TenantID:     "tenant-alpha",
				DatasourceID: "ds-postgres",
				DateColumn:   "as_of_date",
				Mode:         HotOnly,
			},
		},
		{
			name: "malicious tenant id",
			q: &TierQuery{
				TableName:    "positions",
				TenantID:     `tenant'; DROP TABLE positions; --`,
				DatasourceID: "ds-postgres",
				DateColumn:   "as_of_date",
				Mode:         HotOnly,
			},
		},
		{
			name: "malicious datasource id",
			q: &TierQuery{
				TableName:    "positions",
				TenantID:     "tenant-alpha",
				DatasourceID: `ds'; DROP TABLE positions; --`,
				DateColumn:   "as_of_date",
				Mode:         HotOnly,
			},
		},
		{
			name: "malicious hot schema",
			q: &TierQuery{
				TableName:    "positions",
				TenantID:     "tenant-alpha",
				DatasourceID: "ds-postgres",
				DateColumn:   "as_of_date",
				Mode:         HotOnly,
				HotSchema:    `semantic_hot; DROP TABLE positions; --`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, args, err := manager.BuildSafeQuery(context.Background(), tc.q)
			assert.Error(t, err)
			assert.Empty(t, sql)
			assert.Nil(t, args)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "no watermark lookup should happen before validation")
}

// TestMigrateWithValidation_RejectsMaliciousIdentifiers confirms the
// exportToParquet/deleteFromHot path (raw fmt.Sprintf SQL, since StarRocks
// EXPORT/DELETE DDL doesn't support bind parameters for identifiers) is
// guarded at its entry point.
func TestMigrateWithValidation_RejectsMaliciousIdentifiers(t *testing.T) {
	manager, mock := setupManager(t, PostgresQueryDialect{})

	_, err := manager.MigrateWithValidation(context.Background(),
		`positions'; DROP TABLE positions; --`, "tenant-alpha", "ds-postgres", time.Now())
	assert.Error(t, err)

	_, err = manager.MigrateWithValidation(context.Background(),
		"positions", `tenant'; DROP TABLE positions; --`, "ds-postgres", time.Now())
	assert.Error(t, err)

	_, err = manager.MigrateWithValidation(context.Background(),
		"positions", "tenant-alpha", `ds'; DROP TABLE positions; --`, time.Now())
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should be issued before validation")
}

// TestValidateNoOverlap_RejectsMaliciousTableName confirms the raw
// fmt.Sprintf table-name interpolation in ValidateNoOverlap is guarded.
func TestValidateNoOverlap_RejectsMaliciousTableName(t *testing.T) {
	manager, mock := setupManager(t, PostgresQueryDialect{})

	err := manager.ValidateNoOverlap(context.Background(), `positions'; DROP TABLE positions; --`, "tenant-alpha", "ds-postgres")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestValidateTotalRowCount_RejectsMaliciousTableName confirms the raw
// fmt.Sprintf table-name interpolation reachable via countRows is guarded.
func TestValidateTotalRowCount_RejectsMaliciousTableName(t *testing.T) {
	manager, mock := setupManager(t, PostgresQueryDialect{})

	err := manager.ValidateTotalRowCount(context.Background(), `positions'; DROP TABLE positions; --`, "tenant-alpha", "ds-postgres", 10)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
