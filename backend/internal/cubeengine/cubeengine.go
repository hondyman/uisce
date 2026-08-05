package cubeengine

import (
	"context"
	"database/sql"

	"github.com/hondyman/uisce/backend/internal/telemetry/optimize"
	"github.com/hondyman/uisce/backend/models"
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
	Cubes  map[string]models.Cube
	Views  map[string]models.ViewMeta
	Tables []Table
}

type Engine struct {
	catalog    *models.Catalog
	db         *sql.DB
	optService *optimize.Service
}

func NewEngine(catalog *models.Catalog, db *sql.DB, optService *optimize.Service) *Engine {
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

func (e *Engine) Compile(ctx context.Context, req models.QueryRequest) (*EmittedSQL, error) {
	return &EmittedSQL{
		SQL: "SELECT * FROM mock_table",
	}, nil
}
