package api

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/semlayer/backend/internal/region"
)

func TestDBAllowedRegionsProvider_JSONBColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"allowed"}).AddRow("[\"eu-west\",\"us-east\"]")
	mock.ExpectQuery("SELECT COALESCE\\(allowed_regions.*").WithArgs("tenant-1").WillReturnRows(rows)

	p := region.NewDBAllowedRegionsProvider(db)
	list, err := p.GetAllowedRegions("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 || list[0] != "eu-west" || list[1] != "us-east" {
		t.Fatalf("unexpected list: %v", list)
	}
}

func TestDBAllowedRegionsProvider_MetadataFallback_Comma(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"allowed"}).AddRow("eu-west, us-east")
	mock.ExpectQuery("SELECT COALESCE\\(allowed_regions.*").WithArgs("tenant-2").WillReturnRows(rows)

	p := region.NewDBAllowedRegionsProvider(db)
	list, err := p.GetAllowedRegions("tenant-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 || list[0] != "eu-west" || list[1] != "us-east" {
		t.Fatalf("unexpected list: %v", list)
	}
}

func TestDBAllowedRegionsProvider_EmptyOrNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"allowed"}).AddRow(driver.Value(nil))
	mock.ExpectQuery("SELECT COALESCE\\(allowed_regions.*").WithArgs("tenant-3").WillReturnRows(rows)

	p := region.NewDBAllowedRegionsProvider(db)
	list, err := p.GetAllowedRegions("tenant-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got: %v", list)
	}
}

func TestDBAllowedRegionsProvider_JSONTextInMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	jsonArr, _ := json.Marshal([]string{"eu-west"})
	rows := sqlmock.NewRows([]string{"allowed"}).AddRow(string(jsonArr))
	mock.ExpectQuery("SELECT COALESCE\\(allowed_regions.*").WithArgs("tenant-4").WillReturnRows(rows)

	p := region.NewDBAllowedRegionsProvider(db)
	list, err := p.GetAllowedRegions("tenant-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0] != "eu-west" {
		t.Fatalf("unexpected list: %v", list)
	}
}
