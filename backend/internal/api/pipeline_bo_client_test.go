package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
)

const pipeTenant = "00000000-0000-0000-0000-000000000001"

func expectFundContract(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WithArgs("fund", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("mdm.fund", "id"))
	mock.ExpectQuery("SELECT column_name FROM information_schema.columns").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("name").AddRow("aum").AddRow("tenant_id"))
}

func TestPipelineBOClient_ReadIsTenantScopedAndFiltered(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	expectFundContract(mock)
	mock.ExpectQuery(`SELECT \* FROM mdm.fund WHERE tenant_id = \$1 AND "aum" BETWEEN \$2 AND \$3 AND "name" ILIKE \$4 ORDER BY "id" ASC LIMIT \$5 OFFSET \$6`).
		WithArgs(sqlmock.AnyArg(), 1.0, 9.0, "%fund\\_a%", 50, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Fund_A"))

	c := NewPipelineBOClient(NewBOCRUDHandler(sqlxDB, nil, nil))
	rows, err := c.ReadPage(context.Background(), pipeTenant, "fund", []datapipeline.Condition{
		{Field: "aum", Operator: "between", Value: []interface{}{1.0, 9.0}},
		{Field: "name", Operator: "contains", Value: "fund_a"},
	}, 100, 50)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPipelineBOClient_ReadRejectsUnknownColumnAndOperator(t *testing.T) {
	for _, f := range []datapipeline.Condition{
		{Field: "salary; DROP TABLE x", Operator: "equals", Value: 1},
		{Field: "aum", Operator: "1=1 --", Value: 1},
	} {
		db, mock, _ := sqlmock.New()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		expectFundContract(mock)
		_, err := NewPipelineBOClient(NewBOCRUDHandler(sqlxDB, nil, nil)).ReadPage(context.Background(), pipeTenant, "fund", []datapipeline.Condition{f}, 0, 10)
		assert.Error(t, err, "filter %+v", f)
		db.Close()
	}
}

func TestPipelineBOClient_WritesThroughEnforcer(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	enf := &txEnforcer{db: sqlxDB, reject: map[int]bool{1: true}}
	expectFundContract(mock)
	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`INSERT INTO mdm.fund`).WithArgs(pipeTenant, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
	}
	mock.ExpectCommit()

	out, err := NewPipelineBOClient(NewBOCRUDHandler(sqlxDB, nil, enf)).WriteBatch(context.Background(), pipeTenant, "fund",
		datapipeline.BOWriteRequest{Records: []map[string]any{{"name": "A", "aum": 1}, {"name": "B", "aum": 2}}})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Written)
	require.Len(t, out.Failed, 1)
	assert.Equal(t, []string{"notional within limit"}, out.Failed[0].Rules)
	assert.Equal(t, []string{"fund"}, enf.boKeys)
}
