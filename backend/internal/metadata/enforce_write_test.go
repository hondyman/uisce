package metadata

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// A BLOCK rule requiring amount > 100, as a single tenant rule on BO "party".
func expectPartyRule(m sqlmock.Sqlmock) {
	const gold = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	m.ExpectQuery(`uisce_gold_copy_tenant_id`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	m.ExpectQuery(`FROM catalog_node n`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id"}).
			AddRow("00000000-0000-0000-0000-000000000001", "amount over 100", "",
				[]byte(`{"bo_name":"party","tenant_id":"t-1","severity":"BLOCK","timing":"pre_write"}`),
				[]byte(`{"rule_ast":{"type":"condition","field":"amount","fieldPath":"amount","operator":">","value":100,"valueType":"number"}}`),
				true, "t-1"))
	m.ExpectQuery(`FROM business_object_fields bf`).WillReturnRows(sqlmock.NewRows([]string{"field_name", "node_name"}))
}

func expectPartyBO(m sqlmock.Sqlmock) {
	m.ExpectQuery(`FROM public.business_objects bo`).
		WithArgs("party", "t-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}).
			AddRow("00000000-0000-0000-0000-0000000000b0", "party", "/mdm/party"))
	// party has no bound datasource: its records live in the metadata DB.
	m.ExpectQuery(`FROM public.business_object_binding`).WillReturnError(sql.ErrNoRows)
	// ...and no required fields.
	m.ExpectQuery(`COALESCE\(is_required`).WillReturnRows(sqlmock.NewRows([]string{"field_name", "label"}))
}

// party requires IssuerID, bound to column issuer_id.
func expectPartyBORequiringIssuer(m sqlmock.Sqlmock) {
	m.ExpectQuery(`FROM public.business_objects bo`).
		WithArgs("party", "t-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}).
			AddRow("00000000-0000-0000-0000-0000000000b0", "party", "/mdm/party"))
	m.ExpectQuery(`FROM public.business_object_binding`).WillReturnError(sql.ErrNoRows)
	m.ExpectQuery(`COALESCE\(is_required`).
		WillReturnRows(sqlmock.NewRows([]string{"field_name", "label"}).AddRow("IssuerID", "Issuer"))
	m.ExpectQuery(`MAPS_TO`).
		WillReturnRows(sqlmock.NewRows([]string{"field_name", "node_name"}).AddRow("IssuerID", "issuer_id"))
}

func newMockService(t *testing.T) (*BusinessObjectService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &BusinessObjectService{db: sqlx.NewDb(db, "postgres")}, mock
}

func TestEnforceWrite_BlockingRuleRollsBackAndReturnsTypedError(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "true")
	s, mock := newMockService(t)
	expectPartyBO(mock)
	mock.ExpectBegin()
	expectPartyRule(mock)
	mock.ExpectRollback()
	mock.ExpectExec(`INSERT INTO validation_rule_violations`).WillReturnResult(sqlmock.NewResult(1, 1))

	_, err := s.EnforceWrite(context.Background(), "t-1", "party", func(*sqlx.Tx) (map[string]interface{}, error) {
		return map[string]interface{}{"id": "r1", "amount": 50}, nil
	})
	var rej *RuleRejectionError
	if !errors.As(err, &rej) || len(rej.Rules) != 1 || rej.Rules[0] != "amount over 100" {
		t.Fatalf("err = %v; want RuleRejectionError naming the rule", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnforceWrite_ShadowModeCommitsAndStillPersistsViolation(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "")
	s, mock := newMockService(t)
	expectPartyBO(mock)
	mock.ExpectBegin()
	expectPartyRule(mock)
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO validation_rule_violations`).WillReturnResult(sqlmock.NewResult(1, 1))

	rec, err := s.EnforceWrite(context.Background(), "t-1", "party", func(*sqlx.Tx) (map[string]interface{}, error) {
		return map[string]interface{}{"id": "r1", "amount": 50}, nil
	})
	if err != nil || rec["id"] != "r1" {
		t.Fatalf("rec=%v err=%v; shadow mode must not block", rec, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnforceWrite_UnknownBOWritesWithoutEvaluation(t *testing.T) {
	s, mock := newMockService(t)
	mock.ExpectQuery(`FROM public.business_objects bo`).WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}))
	mock.ExpectBegin()
	mock.ExpectCommit()
	if _, err := s.EnforceWrite(context.Background(), "t-1", "oms.raw_table", func(*sqlx.Tx) (map[string]interface{}, error) {
		return map[string]interface{}{"id": "x"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err) // no rule queries may have run
	}
}

func TestEnforceWrite_WriteErrorRollsBack(t *testing.T) {
	s, mock := newMockService(t)
	expectPartyBO(mock)
	mock.ExpectBegin()
	mock.ExpectRollback()
	_, err := s.EnforceWrite(context.Background(), "t-1", "party", func(*sqlx.Tx) (map[string]interface{}, error) {
		return nil, ErrNoRowWritten
	})
	if !errors.Is(err, ErrNoRowWritten) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Row 0 violates the rule, row 1 passes, row 2's own write fails: only row 1
// survives, each failure is attributed, and only the rule violation is
// persisted - after the commit.
func TestEnforceWriteBatch_PerRowSavepoints(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "true")
	s, mock := newMockService(t)
	expectPartyBO(mock)
	mock.ExpectBegin()
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	expectPartyRule(mock)
	mock.ExpectExec(`ROLLBACK TO SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	expectPartyRule(mock)
	mock.ExpectExec(`RELEASE SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`ROLLBACK TO SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO validation_rule_violations`).WillReturnResult(sqlmock.NewResult(1, 1))

	amounts := []int{50, 150, 0}
	res, err := s.EnforceWriteBatch(context.Background(), "t-1", "party", 3, false, func(_ *sqlx.Tx, i int) (map[string]interface{}, error) {
		if i == 2 {
			return nil, errors.New("duplicate key")
		}
		return map[string]interface{}{"id": i, "amount": amounts[i]}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var rej *RuleRejectionError
	if !errors.As(res[0].Err, &rej) {
		t.Errorf("row 0: %v; want rule rejection", res[0].Err)
	}
	if res[1].Err != nil || res[1].Record == nil {
		t.Errorf("row 1: %+v; want written", res[1])
	}
	if res[2].Err == nil || errors.As(res[2].Err, &rej) {
		t.Errorf("row 2: %v; want the write's own error", res[2].Err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnforceWriteBatch_DryRunRollsBackAndPersistsNothing(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "true")
	s, mock := newMockService(t)
	expectPartyBO(mock)
	mock.ExpectBegin()
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	expectPartyRule(mock)
	mock.ExpectExec(`ROLLBACK TO SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	res, err := s.EnforceWriteBatch(context.Background(), "t-1", "party", 1, true, func(*sqlx.Tx, int) (map[string]interface{}, error) {
		return map[string]interface{}{"id": "r", "amount": 1}, nil
	})
	if err != nil || res[0].Err == nil {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err) // an unexpected violation INSERT or COMMIT would fail here
	}
}

// A required field left empty is refused before any rule runs, whatever the
// enforcement mode, and the write is rolled back.
func TestEnforceWrite_MissingRequiredFieldRollsBack(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "")
	s, mock := newMockService(t)
	expectPartyBORequiringIssuer(mock)
	mock.ExpectBegin()
	mock.ExpectRollback()

	_, err := s.EnforceWrite(context.Background(), "t-1", "party", func(*sqlx.Tx) (map[string]interface{}, error) {
		return map[string]interface{}{"id": "r1", "issuer_id": nil}, nil
	})
	var req *RequiredFieldsError
	if !errors.As(err, &req) || len(req.Fields) != 1 || req.Fields[0] != "Issuer (issuer_id)" {
		t.Fatalf("err = %v; want RequiredFieldsError naming Issuer (issuer_id)", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err) // no rule query may have run
	}
}

// Per row in a batch: the empty row is rolled back to its savepoint, the
// filled one - here filled by a database default, as the written row
// shows - goes on to the rules and commits.
func TestEnforceWriteBatch_MissingRequiredFieldPerRow(t *testing.T) {
	t.Setenv("VALIDATION_RULES_ENFORCE", "true")
	s, mock := newMockService(t)
	expectPartyBORequiringIssuer(mock)
	mock.ExpectBegin()
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`ROLLBACK TO SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	expectPartyRule(mock)
	mock.ExpectExec(`RELEASE SAVEPOINT bo_batch_row`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	rows := []map[string]interface{}{
		{"id": 0, "amount": 150, "issuer_id": "  "},
		{"id": 1, "amount": 150, "issuer_id": "ISS-DEFAULT"},
	}
	res, err := s.EnforceWriteBatch(context.Background(), "t-1", "party", 2, false, func(_ *sqlx.Tx, i int) (map[string]interface{}, error) {
		return rows[i], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var req *RequiredFieldsError
	if !errors.As(res[0].Err, &req) {
		t.Errorf("row 0: %v; want RequiredFieldsError", res[0].Err)
	}
	if res[1].Err != nil || res[1].Record == nil {
		t.Errorf("row 1: %+v; want written", res[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// If the required fields cannot be read, nothing is written.
func TestEnforceWrite_RequiredFieldsUnreadableFailsClosed(t *testing.T) {
	s, mock := newMockService(t)
	mock.ExpectQuery(`FROM public.business_objects bo`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "bo_key", "driver_table_name"}).
			AddRow("00000000-0000-0000-0000-0000000000b0", "party", "/mdm/party"))
	mock.ExpectQuery(`FROM public.business_object_binding`).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`COALESCE\(is_required`).WillReturnError(errors.New("connection reset"))
	wrote := false
	_, err := s.EnforceWrite(context.Background(), "t-1", "party", func(*sqlx.Tx) (map[string]interface{}, error) {
		wrote = true
		return nil, nil
	})
	if err == nil || wrote {
		t.Fatalf("err=%v wrote=%v; want refused before writing", err, wrote)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err) // no BEGIN
	}
}

func TestMissingRequired(t *testing.T) {
	f := []requiredField{{Label: "Issuer", Column: "issuer_id"}, {Label: "name", Column: "name"}, {Label: "Elsewhere", Column: "not_in_row"}}
	if m := missingRequired("party", f, map[string]interface{}{"issuer_id": []byte(" "), "name": "x"}); m == nil || m.Fields[0] != "Issuer (issuer_id)" || len(m.Fields) != 1 {
		t.Fatalf("got %+v", m)
	}
	if m := missingRequired("party", f, map[string]interface{}{"issuer_id": 0, "name": false}); m != nil {
		t.Fatalf("zero values are values: %+v", m)
	}
}
