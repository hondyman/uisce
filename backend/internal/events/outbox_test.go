package events

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestPublishEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs("Order.Created", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := sqlxDB.BeginTxx(ctx, nil)
	assert.NoError(t, err)

	payload := map[string]string{"order_id": "123"}
	err = PublishEvent(ctx, tx, "Order.Created", payload)
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestProcessOutbox(t *testing.T) {
	// Placeholder for future test when Publisher interface is available
}
