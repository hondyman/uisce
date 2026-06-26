package financial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/financial"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolRegistry_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := financial.NewSQLToolRepository(db)
	registry := financial.NewToolRegistry(repo)
	ctx := context.Background()

	t.Run("Get Built-in Tool", func(t *testing.T) {
		tool, found := registry.Get(ctx, "calculate_time_weighted_return")
		assert.True(t, found)
		assert.NotNil(t, tool)
		assert.Equal(t, "calculate_time_weighted_return", tool.Name())
	})

	t.Run("Get DB Tool", func(t *testing.T) {
		toolName := "custom_tool"
		toolID := uuid.New()
		
		rows := sqlmock.NewRows([]string{"id", "name", "description", "parameters_schema", "handler_type", "handler_config", "created_at", "updated_at"}).
			AddRow(toolID, toolName, "Custom Tool", []byte(`{}`), "script", []byte(`{}`), time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM financial_tools WHERE name = \$1`).
			WithArgs(toolName).
			WillReturnRows(rows)

		tool, found := registry.Get(ctx, toolName)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
		require.True(t, found)
		require.NotNil(t, tool)
		assert.Equal(t, toolName, tool.Name())

		// Verify Execute
		res, err := tool.Execute(ctx, json.RawMessage(`{"foo":"bar"}`))
		require.NoError(t, err)
		resMap, ok := res.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "executed", resMap["status"])
	})
}
