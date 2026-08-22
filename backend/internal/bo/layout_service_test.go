package bo

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetResolvedBOLayout(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	svc := NewLayoutService(sqlxDB, nil)

	tenantID := uuid.New()
	boID := uuid.New()
	discField := "sec_typ_cd"
	subtypesJSON := []byte(`{"EQUITY":{"displayName":"Equity","color":"#10B981","icon":"trending-up"}}`)

	// 1. Expect BO core query
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT key, discriminator_field, COALESCE(subtypes_config, '{}'::jsonb) AS subtypes_config 
		FROM public.business_objects 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`)).
		WithArgs(boID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"key", "discriminator_field", "subtypes_config"}).
			AddRow("security", &discField, subtypesJSON))

	// 2. Expect taxonomy fields query
	fieldsJSON := []byte(`[{"fieldId":"` + uuid.NewString() + `","key":"ticker","displayName":"Ticker","role":"IDENTIFIER","dataType":"string","subtypeScope":"CORE","isRequired":true,"isGoverned":false,"formula":""}]`)
	mock.ExpectQuery(regexp.QuoteMeta(`WITH raw_fields AS`)).
		WithArgs(boID, tenantID, "ALL").
		WillReturnRows(sqlmock.NewRows([]string{"group_key", "group_name", "group_seq", "fields_json"}).
			AddRow("sec_id", "Security Identity", 10, fieldsJSON))

	// Run
	resp, err := svc.GetResolvedBOLayout(context.Background(), tenantID, boID, "ALL")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, boID, resp.BOID)
	require.Equal(t, "security", resp.BOKey)
	require.Equal(t, "sec_typ_cd", resp.DiscriminatorField)
	require.Equal(t, 1, len(resp.Groups))
	require.Equal(t, "Security Identity", resp.Groups[0].GroupName)
	require.Equal(t, 1, len(resp.Groups[0].Fields))
	require.Equal(t, "Ticker", resp.Groups[0].Fields[0].DisplayName)

	require.NoError(t, mock.ExpectationsWereMet())
}
