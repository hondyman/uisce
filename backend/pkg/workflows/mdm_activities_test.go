package workflows

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/mdm"
)

func newTestMDMActivities(t *testing.T) (*MDMActivities, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	engine := mdm.NewUniversalMasteringEngine(sqlx.NewDb(db, "postgres"))
	return NewMDMActivities(engine), mock
}

func TestActivityValidateGoldenRecord_RequiresTenantID(t *testing.T) {
	a, _ := newTestMDMActivities(t)
	_, err := a.ActivityValidateGoldenRecord(context.Background(), map[string]interface{}{
		"entity_type": "Counterparty", "entity_id": "CP-123",
	}, nil)
	if err == nil {
		t.Fatal("expected an error when tenant_id is missing")
	}
}

func TestActivityValidateGoldenRecord_RequiresEntityTypeAndID(t *testing.T) {
	a, _ := newTestMDMActivities(t)
	_, err := a.ActivityValidateGoldenRecord(context.Background(), map[string]interface{}{
		"tenant_id": uuid.New().String(),
	}, nil)
	if err == nil {
		t.Fatal("expected an error when entity_type/entity_id are missing")
	}
}

func TestActivityValidateGoldenRecord_NoGoldenRecordIsNotAViolation(t *testing.T) {
	a, mock := newTestMDMActivities(t)
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT golden_attributes`)).
		WillReturnError(sql.ErrNoRows)

	result, err := a.ActivityValidateGoldenRecord(context.Background(), map[string]interface{}{
		"tenant_id": tenantID.String(), "entity_type": "Counterparty", "entity_id": "CP-999",
	}, map[string]interface{}{"attributes": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["validation_status"] != "NO_GOLDEN_RECORD" {
		t.Errorf("expected NO_GOLDEN_RECORD, got %v", result)
	}
}

func TestActivityValidateGoldenRecord_MatchPasses(t *testing.T) {
	a, mock := newTestMDMActivities(t)
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT golden_attributes`)).
		WillReturnRows(sqlmock.NewRows([]string{"golden_attributes"}).
			AddRow([]byte(`{"risk_rating":"HIGH","country":"US"}`)))

	result, err := a.ActivityValidateGoldenRecord(context.Background(), map[string]interface{}{
		"tenant_id": tenantID.String(), "entity_type": "Counterparty", "entity_id": "CP-123",
	}, map[string]interface{}{"attributes": map[string]interface{}{"risk_rating": "HIGH", "country": "US"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["validation_status"] != "MATCH" {
		t.Errorf("expected MATCH, got %v", result)
	}
}

func TestActivityValidateGoldenRecord_DriftIsBlocked(t *testing.T) {
	a, mock := newTestMDMActivities(t)
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT golden_attributes`)).
		WillReturnRows(sqlmock.NewRows([]string{"golden_attributes"}).
			AddRow([]byte(`{"risk_rating":"HIGH"}`)))

	_, err := a.ActivityValidateGoldenRecord(context.Background(), map[string]interface{}{
		"tenant_id": tenantID.String(), "entity_type": "Counterparty", "entity_id": "CP-123",
	}, map[string]interface{}{"attributes": map[string]interface{}{"risk_rating": "LOW"}})
	if err == nil {
		t.Fatal("expected GOLDEN_RECORD_VIOLATION error for drifted attribute")
	}
}
