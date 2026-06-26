package planner

import (
	"reflect"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

func TestSQLBuilder_ApplyHints(t *testing.T) {
	tests := []struct {
		name     string
		hints    domain.PruningHints
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "column pruning only",
			hints: domain.PruningHints{
				Columns: []string{"col1", "col2"},
			},
			wantSQL:  "SELECT col1, col2 FROM test_table",
			wantArgs: nil,
		},
		{
			name: "row filtering only",
			hints: domain.PruningHints{
				RowFilters: []string{"tenant_id = $1"},
				BindArgs:   []any{"acme"},
			},
			wantSQL:  "SELECT * FROM test_table WHERE tenant_id = $1",
			wantArgs: []any{"acme"},
		},
		{
			name: "both column and row pruning",
			hints: domain.PruningHints{
				Columns:    []string{"avg_order_value", "total_orders"},
				RowFilters: []string{"tenant_id = $1", "region = $2"},
				BindArgs:   []any{"acme", "us-west"},
			},
			wantSQL:  "SELECT avg_order_value, total_orders FROM test_table WHERE tenant_id = $1 AND region = $2",
			wantArgs: []any{"acme", "us-west"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSQLBuilder("test_table")
			builder.ApplyHints(tt.hints)
			gotSQL, gotArgs := builder.Build()

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL = %v, want %v", gotSQL, tt.wantSQL)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
