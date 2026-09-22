package scanner

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
)

// Foreign keys must be read from pg_constraint. The information_schema views joined by constraint name ran for
// many minutes on a database with several schemas (the scan sat at "Extracting metadata" indefinitely) and could
// pair same-named constraints of different tables. This pins the catalog query, the schema filter and the
// handling of one relationship.
func TestProcessForeignKeys_ReadsPgConstraintNotInformationSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ds := uuid.New()
	s := &AnsiScanner{
		sourceDB: db, tenantDatasourceId: ds, sourceSystem: "crims",
		schemaWhitelist: []string{"mdm", "orm"},
		columnMap:       map[uuid.UUID]*models.CatalogNode{},
	}
	col := &models.CatalogNode{QualifiedPath: "/mdm/security_type_mapping/source_system_id", Properties: json.RawMessage(`{"data_type":"uuid"}`)}
	s.columnMap[generateID(ds.String(), "crims", NODE_TYPE_COLUMN.String(), col.QualifiedPath)] = col

	// a regexp that fails if the slow views come back
	mock.ExpectQuery(`(?s)^\s*SELECT\s+con\.conname.*FROM pg_catalog\.pg_constraint con.*con\.contype = 'f'.*sn\.nspname IN \(\$1, \$2\)`).
		WithArgs("mdm", "orm").
		WillReturnRows(sqlmock.NewRows([]string{
			"constraint_name", "constraint_schema", "source_schema", "source_table", "source_column",
			"target_schema", "target_table", "target_column", "on_update", "on_delete", "is_deferrable", "initially_deferred", "ordinal_position",
		}).AddRow("fk_stm_source", "mdm", "mdm", "security_type_mapping", "source_system_id",
			"mdm", "source_system", "id", "NO ACTION", "NO ACTION", "NO", "NO", 1))

	if err := s.processForeignKeys(); err != nil {
		t.Fatalf("processForeignKeys: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	var p map[string]interface{}
	if err := json.Unmarshal(col.Properties, &p); err != nil {
		t.Fatal(err)
	}
	if p["is_foreign_key"] != true || p["foreign_key_target_table"] != "mdm.source_system" || p["foreign_key_target_column"] != "id" {
		t.Errorf("column not marked as a foreign key to mdm.source_system.id: %s", col.Properties)
	}
	if p["data_type"] != "uuid" {
		t.Errorf("existing properties lost: %s", col.Properties)
	}
}
