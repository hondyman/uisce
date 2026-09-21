package scanner

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
)

func TestProcessUniqueKeys_MarksColumnsWithConstraintGroups(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ds := uuid.New()
	s := &AnsiScanner{sourceDB: db, tenantDatasourceId: ds, sourceSystem: "crims", columnMap: map[uuid.UUID]*models.CatalogNode{}}
	node := func(schema, table, col string) *models.CatalogNode {
		n := &models.CatalogNode{QualifiedPath: fmt.Sprintf("/%s/%s/%s", schema, table, col), Properties: json.RawMessage(`{"data_type":"text"}`)}
		s.columnMap[generateID(ds.String(), "crims", NODE_TYPE_COLUMN.String(), n.QualifiedPath)] = n
		return n
	}
	tenant, party, other := node("mdm", "party", "tenant_id"), node("mdm", "party", "party_cd"), node("mdm", "party", "legal_name")

	mock.ExpectQuery(`information_schema.table_constraints`).WillReturnRows(
		sqlmock.NewRows([]string{"table_schema", "table_name", "constraint_name", "column_name", "ordinal_position"}).
			AddRow("mdm", "party", "party_cd_key", "tenant_id", 1).
			AddRow("mdm", "party", "party_cd_key", "party_cd", 2).
			// a constraint on a column that was not scanned must be ignored, not fail
			AddRow("mdm", "ghost", "ghost_key", "x", 1))

	if err := s.processUniqueKeys(); err != nil {
		t.Fatalf("processUniqueKeys: %v", err)
	}

	for _, n := range []*models.CatalogNode{tenant, party} {
		var p map[string]interface{}
		if err := json.Unmarshal(n.Properties, &p); err != nil {
			t.Fatal(err)
		}
		if p["is_unique_key"] != true || p["data_type"] != "text" {
			t.Errorf("%s: props = %s; want is_unique_key kept alongside existing props", n.QualifiedPath, n.Properties)
		}
		g := p["unique_key_groups"].([]interface{})[0].(map[string]interface{})
		cols := g["columns"].([]interface{})
		if g["name"] != "party_cd_key" || len(cols) != 2 || cols[0] != "tenant_id" || cols[1] != "party_cd" {
			t.Errorf("%s: group = %v; want party_cd_key over [tenant_id party_cd]", n.QualifiedPath, g)
		}
	}
	if string(other.Properties) != `{"data_type":"text"}` {
		t.Errorf("column outside any unique constraint was modified: %s", other.Properties)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
