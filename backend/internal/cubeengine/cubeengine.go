package cubeengine

import (
	"context"
	"database/sql"

	"github.com/hondyman/uisce/backend/internal/cube"
	"github.com/hondyman/uisce/backend/internal/telemetry/optimize"
)

type Table struct {
	Name    string
	Schema  string
	Columns []Column
	FKs     []FK
}

type Column struct {
	Name     string
	DataType string
}

type FK struct {
	Name     string
	FromCols []string
	ToCols   []string
	ToSchema string
	ToTable  string
}

type Catalog struct {
	Cubes  map[string]cube.Cube
	Views  map[string]cube.ViewMeta
	Tables []Table
}

type Engine struct {
	catalog    *cube.Catalog
	db         *sql.DB
	optService *optimize.Service
}

func NewEngine(catalog *cube.Catalog, db *sql.DB, optService *optimize.Service) *Engine {
	return &Engine{
		catalog:    catalog,
		db:         db,
		optService: optService,
	}
}

type EmittedSQL struct {
	SQL                string
	Params             []any
	UsedPreAggregation struct {
		Name string
	}
}

func (e *Engine) Compile(ctx context.Context, req cube.QueryRequest) (*EmittedSQL, error) {
	return &EmittedSQL{
		SQL: "SELECT * FROM mock_table",
	}, nil
}
