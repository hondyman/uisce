package analytics

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSaveExtensionModelRejectsSelfExtend(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")
	svc := NewSemanticModelService(sqlxDB)

	dsID := uuid.New()
	model := map[string]any{
		"base_model_key": "/base",
		"model_key":      "/base",
	}

	err = svc.SaveExtensionModel(context.Background(), dsID, model)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
