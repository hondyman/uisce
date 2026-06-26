package financial

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// ToolRepository defines the interface for tool storage
type ToolRepository interface {
	List(ctx context.Context) ([]FinancialTool, error)
	GetByName(ctx context.Context, name string) (*FinancialTool, error)
	Create(ctx context.Context, tool *FinancialTool) error
}

// SQLToolRepository implements ToolRepository using database/sql
type SQLToolRepository struct {
	DB           *sql.DB
	hasuraClient HasuraClient
}

func NewSQLToolRepository(db *sql.DB) *SQLToolRepository {
	return &SQLToolRepository{DB: db}
}

func NewSQLToolRepositoryWithHasura(db *sql.DB, hasura HasuraClient) *SQLToolRepository {
	return &SQLToolRepository{DB: db, hasuraClient: hasura}
}

func (r *SQLToolRepository) List(ctx context.Context) ([]FinancialTool, error) {
	return r.listToolsRecords(ctx)
}

func (r *SQLToolRepository) GetByName(ctx context.Context, name string) (*FinancialTool, error) {
	return r.getToolByNameRecord(ctx, name)
}

func (r *SQLToolRepository) Create(ctx context.Context, tool *FinancialTool) error {
	return r.createToolRecord(ctx, tool)
}

// listToolsRecords retrieves all financial tools from the database
// TODO: Replace with Hasura GraphQL query:
//
//	query { financial_tools(order_by: {name: asc}) { id name description parameters_schema handler_type handler_config created_at updated_at } }
//
// SQL fallback: SELECT all fields with ORDER BY name
func (r *SQLToolRepository) listToolsRecords(ctx context.Context) ([]FinancialTool, error) {
	query := `
		SELECT id, name, description, parameters_schema, handler_type, handler_config, created_at, updated_at
		FROM financial_tools
		ORDER BY name
	`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list financial tools: %w", err)
	}
	defer rows.Close()

	var tools []FinancialTool
	for rows.Next() {
		var t FinancialTool
		var params, config []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &params, &t.HandlerType, &config, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan financial tool: %w", err)
		}
		t.ParametersSchema = json.RawMessage(params)
		t.HandlerConfig = json.RawMessage(config)
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

// getToolByNameRecord retrieves a single financial tool by name
// TODO: Replace with Hasura GraphQL query:
//
//	query { financial_tools(where: {name: {_eq: $name}}) { id name description parameters_schema handler_type handler_config created_at updated_at } }
//
// SQL fallback: SELECT by name with QueryRowContext + Scan
func (r *SQLToolRepository) getToolByNameRecord(ctx context.Context, name string) (*FinancialTool, error) {
	query := `
		SELECT id, name, description, parameters_schema, handler_type, handler_config, created_at, updated_at
		FROM financial_tools
		WHERE name = $1
	`
	var t FinancialTool
	var params, config []byte
	err := r.DB.QueryRowContext(ctx, query, name).Scan(&t.ID, &t.Name, &t.Description, &params, &t.HandlerType, &config, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get financial tool: %w", err)
	}
	t.ParametersSchema = json.RawMessage(params)
	t.HandlerConfig = json.RawMessage(config)
	return &t, nil
}

// createToolRecord inserts a new financial tool
// TODO: Replace with Hasura GraphQL mutation:
//
//	mutation { insert_financial_tools_one(object: {id: $id, name: $name, description: $description, parameters_schema: $params, handler_type: $type, handler_config: $config}) { id } }
//	Note: JSONB fields parameters_schema and handler_config accept JSON objects directly
//
// SQL fallback: INSERT with 6 fields including JSON marshaling
func (r *SQLToolRepository) createToolRecord(ctx context.Context, tool *FinancialTool) error {
	query := `
		INSERT INTO financial_tools (id, name, description, parameters_schema, handler_type, handler_config)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	paramsJSON, err := json.Marshal(tool.ParametersSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters schema: %w", err)
	}
	configJSON, err := json.Marshal(tool.HandlerConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal handler config: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, tool.ID, tool.Name, tool.Description, paramsJSON, tool.HandlerType, configJSON)
	if err != nil {
		return fmt.Errorf("failed to create financial tool: %w", err)
	}
	return nil
}
