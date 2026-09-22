package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// A table can have several foreign keys to the same table (source_system_id_a and _b both reference
// mdm.source_system), and each relationship has its own stable edge id. The edge merge has to match stored edges
// by that id. Matching by (source table, target table) made two source rows hit one stored row, Postgres refused
// with "MERGE command cannot affect row a second time", and the whole scan transaction rolled back.
func TestMergeCatalogData_EdgesMatchByEdgeIDNotTablePair(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`MERGE INTO public\.catalog_node`).WillReturnResult(sqlmock.NewResult(0, 5))
	// The edge merge must key on target.id = source.id, and must not go back to the table pair.
	mock.ExpectExec(`(?s)MERGE INTO public\.catalog_edge.*ON target\.tenant_datasource_id = source\.tenant_datasource_id::text\s+AND target\.id = source\.id\s`).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`UPDATE public\.catalog_node n`).WillReturnResult(sqlmock.NewResult(0, 0))

	tx, err := sqlx.NewDb(sqlDB, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := MergeCatalogData(tx, uuid.New()); err != nil {
		t.Fatalf("MergeCatalogData: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
