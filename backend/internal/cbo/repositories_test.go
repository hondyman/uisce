package cbo

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestDBPreAggRepository_ListForBO_FiltersByRegion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewDBPreAggRepository(sqlxDB)

	tenID := uuid.New()
	boName := "orders"
	region := "eu-west"

	props := map[string]interface{}{"region": region, "size_bytes": 12345}
	propsBytes, _ := json.Marshal(props)
	config := map[string]interface{}{"materialization": map[string]interface{}{"target_name": "preagg_orders"}, "group_by": []string{"country"}, "measures": []string{"total"}}
	configBytes, _ := json.Marshal(config)

	rows := sqlmock.NewRows([]string{"node_name", "properties", "config"}).
		AddRow("orders_by_country", driver.Value(string(propsBytes)), driver.Value(string(configBytes)))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT n.node_name, n.properties, n.config")).
		WithArgs(tenID, boName, region).
		WillReturnRows(rows)

	res, err := repo.ListForBO(context.Background(), "prod", &tenID, boName, region)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 pre-agg, got %d", len(res))
	}

	if res[0].Region != region {
		t.Fatalf("expected region %s, got %s", region, res[0].Region)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
