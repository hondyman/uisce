package lakehouse

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type ColumnDef struct {
	Name    string
	Type    DataType
	Comment string
}

type DDLExecutor struct {
	client QueryExecutor
	logger *zap.Logger
}

func NewDDLExecutor(client QueryExecutor, logger *zap.Logger) *DDLExecutor {
	return &DDLExecutor{client: client, logger: logger}
}

func (d *DDLExecutor) EnsureSchema(ctx context.Context, schema string) error {
	q := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS iceberg.%s", quoteIdent(schema))
	d.logger.Debug("DDL EnsureSchema", zap.String("schema", schema), zap.String("sql", q))
	return d.exec(ctx, q)
}

func (d *DDLExecutor) EnsureTable(ctx context.Context, schema, table string, cols []ColumnDef) error {
	if len(cols) == 0 {
		cols = append(cols, defaultIngestionColumns()...)
	} else {
		cols = append(cols, defaultIngestionColumns()...)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS iceberg.%s.%s (\n",
		quoteIdent(schema), quoteIdent(table)))

	for i, col := range cols {
		if i > 0 {
			sb.WriteString(",\n")
		}
		line := fmt.Sprintf("    %s %s", quoteIdent(col.Name), col.Type)
		if col.Comment != "" {
			line += fmt.Sprintf(" COMMENT '%s'", escapeSingleQuote(col.Comment))
		}
		sb.WriteString(line)
	}

	sb.WriteString("\n) WITH (\n")
	sb.WriteString("    partitioning = ARRAY['day(_ingested_at)']\n")
	sb.WriteString(")")

	q := sb.String()
	d.logger.Debug("DDL EnsureTable",
		zap.String("schema", schema),
		zap.String("table", table),
		zap.String("sql", q))

	return d.exec(ctx, q)
}

func (d *DDLExecutor) EnsureColumn(ctx context.Context, schema, table string, col ColumnDef) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ALTER TABLE iceberg.%s.%s ADD COLUMN %s %s",
		quoteIdent(schema), quoteIdent(table),
		quoteIdent(col.Name), col.Type))
	if col.Comment != "" {
		sb.WriteString(fmt.Sprintf(" COMMENT '%s'", escapeSingleQuote(col.Comment)))
	}
	q := sb.String()
	d.logger.Debug("DDL EnsureColumn",
		zap.String("schema", schema),
		zap.String("table", table),
		zap.String("column", col.Name),
		zap.String("sql", q))
	return d.exec(ctx, q)
}

func (d *DDLExecutor) EnsureWarehouseTable(ctx context.Context, table string, cols []ColumnDef, partitionExpr string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS iceberg.uisce_warehouse.%s (\n", quoteIdent(table)))

	for i, col := range cols {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("    %s %s", quoteIdent(col.Name), col.Type))
		if col.Comment != "" {
			sb.WriteString(fmt.Sprintf(" COMMENT '%s'", escapeSingleQuote(col.Comment)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n) WITH (\n    partitioning = ARRAY['%s']\n)", partitionExpr))
	return d.exec(ctx, sb.String())
}

func (d *DDLExecutor) TableExists(ctx context.Context, schema, table string) bool {
	rows, err := d.client.Query(ctx,
		fmt.Sprintf("SELECT count(*) FROM iceberg.information_schema.tables WHERE table_schema = '%s' AND table_name = '%s'",
			escapeSingleQuote(schema), escapeSingleQuote(table)))
	if err != nil {
		return false
	}
	defer rows.Close()

	if rows.Next() {
		var count int64
		if err := rows.Scan(&count); err == nil {
			return count > 0
		}
	}
	return false
}

func (d *DDLExecutor) exec(ctx context.Context, query string) error {
	_, err := d.client.Execute(ctx, query)
	if err != nil {
		if isAlreadyExistsError(err) {
			d.logger.Debug("DDL already-exists (treated as success)", zap.String("sql", truncate(query, 120)))
			return nil
		}
		return fmt.Errorf("datafusion DDL failed: %w\nSQL: %s", err, truncate(query, 300))
	}
	return nil
}

func defaultIngestionColumns() []ColumnDef {
	return []ColumnDef{
		{Name: "_ingested_at", Type: TIMESTAMP, Comment: "LPS ingestion timestamp (partition key)"},
		{Name: "_source_tenant", Type: VARCHAR, Comment: "Obfuscated tenant hash (not raw UUID)"},
		{Name: "_source_ds", Type: VARCHAR, Comment: "Obfuscated datasource hash"},
		{Name: "_row_hash", Type: VARCHAR, Comment: "SHA-256 of the source row for deduplication"},
	}
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "schema already exists")
}
