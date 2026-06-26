package profiler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/hondyman/semlayer/backend/internal/profiler/helpers"
)

type ProgressFunc func(current, total int, message string)

// ProfileTablesFunc is a package-level function variable that points to the
// real ProfileTables implementation. Tests may replace this variable to
// inject a fake profiler to avoid hitting real Postgres instances.
var ProfileTablesFunc = ProfileTables

// ProfileTables is a minimal, safe implementation that lists columns and
// performs a simple upsert for each column into sml.column_profiles. This
// implementation focuses on being correct and compilable; it intentionally
// stores minimal metadata (data_type=text, cardinality=0) so callers can read
// results later. The real sampling/analytics can be added iteratively.
func ProfileTables(ctx context.Context, logger *zap.Logger, alphaPool *pgxpool.Pool, tenantID string, datasourceID string, sourceDSN string, schema string, tables []string, sampleSize int, fpRate float64, batchSize int, progress ProgressFunc) error {
	// Connect to the source DB to inspect columns
	srcPool, err := pgxpool.New(ctx, sourceDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to source dsn: %w", err)
	}
	defer srcPool.Close()

	// sane defaults
	if sampleSize <= 0 {
		sampleSize = 1000
	}
	if fpRate <= 0 {
		fpRate = 0.01
	}

	if batchSize <= 0 {
		batchSize = 50
	}

	for ti, table := range tables {
		if progress != nil {
			progress(ti+1, len(tables), fmt.Sprintf("profiling table %s.%s", schema, table))
		}

		colRows, err := srcPool.Query(ctx, "SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2", schema, table)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to list columns", zap.String("schema", schema), zap.String("table", table), zap.Error(err))
			}
			continue
		}

		var cols []string
		for colRows.Next() {
			var col string
			if err := colRows.Scan(&col); err != nil {
				if logger != nil {
					logger.Warn("failed to scan column", zap.Error(err))
				}
				continue
			}
			cols = append(cols, col)
		}
		colRows.Close()

		// process columns in batches to reduce round-trips
		for i := 0; i < len(cols); i += batchSize {
			end := i + batchSize
			if end > len(cols) {
				end = len(cols)
			}
			batchCols := cols[i:end]

			b := &pgx.Batch{}

			for _, col := range batchCols {
				if progress != nil {
					progress(0, 0, fmt.Sprintf("profiling column %s", col))
				}

				// sample values using random ordering (portable across Postgres variants)
				quotedSchema := pgx.Identifier{schema}.Sanitize()
				quotedTable := pgx.Identifier{table}.Sanitize()
				quotedCol := pgx.Identifier{col}.Sanitize()
				q := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY random() LIMIT $1", quotedCol, quotedSchema, quotedTable)

				valRows, err := srcPool.Query(ctx, q, sampleSize)
				if err != nil {
					if logger != nil {
						logger.Warn("failed to sample values", zap.String("schema", schema), zap.String("table", table), zap.String("column", col), zap.Error(err))
					}
					// queue a minimal upsert so there's at least a record for the column
					// Try to resolve a catalog node id for this column; if not found, fall back to composite upsert
					colNodeID, errID := resolveColumnNodeID(ctx, alphaPool, datasourceID, schema, table, col)
					if errID == nil && colNodeID != "" {
						// minimal upsert using id primary key
						b.Queue(`
							INSERT INTO sml.column_profiles
								(id, tenant_datasource_id, data_type, cardinality, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6)
							ON CONFLICT (id) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, colNodeID, datasourceID, "text", 0, nil, time.Now())
					} else {
						b.Queue(`
							INSERT INTO sml.column_profiles
								(tenant_id, datasource_id, datasource, schema, table_name, column_name, data_type, cardinality, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
							ON CONFLICT (tenant_id, datasource_id, schema, table_name, column_name) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, tenantID, datasourceID, sourceDSN, schema, table, col, "text", 0, nil, time.Now())
					}
					continue
				}

				var values []interface{}
				for valRows.Next() {
					var v interface{}
					if err := valRows.Scan(&v); err != nil {
						if logger != nil {
							logger.Warn("failed to scan sample value", zap.Error(err))
						}
						continue
					}
					values = append(values, v)
				}
				valRows.Close()

				if len(values) == 0 {
					// No samples; behave like the earlier minimal case but prefer id-based upsert if possible
					colNodeID, errID := resolveColumnNodeID(ctx, alphaPool, datasourceID, schema, table, col)
					if errID == nil && colNodeID != "" {
						b.Queue(`
							INSERT INTO sml.column_profiles
								(id, tenant_datasource_id, data_type, cardinality, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6)
							ON CONFLICT (id) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, colNodeID, datasourceID, "text", 0, nil, time.Now())
					} else {
						b.Queue(`
							INSERT INTO sml.column_profiles
								(tenant_id, datasource_id, datasource, schema, table_name, column_name, data_type, cardinality, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
							ON CONFLICT (tenant_id, datasource_id, schema, table_name, column_name) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, tenantID, datasourceID, sourceDSN, schema, table, col, "text", 0, nil, time.Now())
					}
					continue
				}

				// compute profile using shared helpers
				prof := helpers.ComputeProfile(values)
				prof.DataSource = sourceDSN
				prof.Schema = schema
				prof.TableName = table
				prof.ColumnName = col
				prof.CreatedAt = time.Now()

				// create bloom filter
				bloomBytes, err := helpers.CreateBloomFilter(values, fpRate)
				if err != nil {
					if logger != nil {
						logger.Warn("failed to create bloom filter", zap.String("column", col), zap.Error(err))
					}
					// fall back to nil bloom; prefer id-based upsert when possible and store frequent/inferred under properties
					colNodeID, errID := resolveColumnNodeID(ctx, alphaPool, datasourceID, schema, table, col)
					propertiesJSON := fmt.Sprintf(`{"frequent_values": %s, "inferred_patterns": %s}`, toJSONStringArray(prof.FrequentValues), toJSONStringArray(prof.InferredPatterns))
					if errID == nil && colNodeID != "" {
						b.Queue(`
							INSERT INTO sml.column_profiles
								(id, tenant_datasource_id, data_type, cardinality, min_length, max_length, avg_length, properties, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
							ON CONFLICT (id) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								min_length = EXCLUDED.min_length,
								max_length = EXCLUDED.max_length,
								avg_length = EXCLUDED.avg_length,
								properties = EXCLUDED.properties,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, colNodeID, datasourceID, prof.DataType, prof.Cardinality, prof.MinLength, prof.MaxLength, prof.AvgLength, propertiesJSON, nil, time.Now())
					} else {
						b.Queue(`
							INSERT INTO sml.column_profiles
								(tenant_id, datasource_id, datasource, schema, table_name, column_name, data_type, cardinality, min_length, max_length, avg_length, properties, bloom_filter, created_at)
							VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
							ON CONFLICT (tenant_id, datasource_id, schema, table_name, column_name) DO UPDATE SET
								data_type = EXCLUDED.data_type,
								cardinality = EXCLUDED.cardinality,
								min_length = EXCLUDED.min_length,
								max_length = EXCLUDED.max_length,
								avg_length = EXCLUDED.avg_length,
								properties = EXCLUDED.properties,
								bloom_filter = EXCLUDED.bloom_filter,
								created_at = EXCLUDED.created_at
						`, tenantID, datasourceID, sourceDSN, schema, table, col, prof.DataType, prof.Cardinality, prof.MinLength, prof.MaxLength, prof.AvgLength, propertiesJSON, nil, time.Now())
					}
					continue
				}
				// Prefer id-based upsert when possible; pack frequent_values and inferred_patterns into properties jsonb
				colNodeID, errID := resolveColumnNodeID(ctx, alphaPool, datasourceID, schema, table, col)
				propertiesJSON := fmt.Sprintf(`{"frequent_values": %s, "inferred_patterns": %s}`, toJSONStringArray(prof.FrequentValues), toJSONStringArray(prof.InferredPatterns))
				if errID == nil && colNodeID != "" {
					b.Queue(`
						INSERT INTO sml.column_profiles
							(id, tenant_datasource_id, data_type, cardinality, min_length, max_length, avg_length, properties, bloom_filter, created_at)
						VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
						ON CONFLICT (id) DO UPDATE SET
							data_type = EXCLUDED.data_type,
							cardinality = EXCLUDED.cardinality,
							min_length = EXCLUDED.min_length,
							max_length = EXCLUDED.max_length,
							avg_length = EXCLUDED.avg_length,
							properties = EXCLUDED.properties,
							bloom_filter = EXCLUDED.bloom_filter,
							created_at = EXCLUDED.created_at
					`, colNodeID, datasourceID, prof.DataType, prof.Cardinality, prof.MinLength, prof.MaxLength, prof.AvgLength, propertiesJSON, bloomBytes, time.Now())
				} else {
					b.Queue(`
						INSERT INTO sml.column_profiles
							(tenant_id, datasource_id, datasource, schema, table_name, column_name, data_type, cardinality, min_length, max_length, avg_length, properties, bloom_filter, created_at)
						VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
						ON CONFLICT (tenant_id, datasource_id, schema, table_name, column_name) DO UPDATE SET
							data_type = EXCLUDED.data_type,
							cardinality = EXCLUDED.cardinality,
							min_length = EXCLUDED.min_length,
							max_length = EXCLUDED.max_length,
							avg_length = EXCLUDED.avg_length,
							properties = EXCLUDED.properties,
							bloom_filter = EXCLUDED.bloom_filter,
							created_at = EXCLUDED.created_at
					`, tenantID, datasourceID, sourceDSN, schema, table, col, prof.DataType, prof.Cardinality, prof.MinLength, prof.MaxLength, prof.AvgLength, propertiesJSON, bloomBytes, time.Now())
				}
			}

			if b.Len() == 0 {
				continue
			}

			br := alphaPool.SendBatch(ctx, b)
			// consume results for each queued statement
			for q := 0; q < b.Len(); q++ {
				if _, err := br.Exec(); err != nil {
					if logger != nil {
						logger.Warn("failed to execute upsert in batch", zap.Error(err))
					}
				}
			}
			br.Close()
		}
	}

	return nil
}

// resolveColumnNodeID attempts to find the catalog_node.id for the given
// tenant_datasource_id, schema, table, and column. It returns the id as a
// string (empty if not found) and any query error encountered.
func resolveColumnNodeID(ctx context.Context, alphaPool *pgxpool.Pool, tenantDatasourceID string, schema string, table string, column string) (string, error) {
	if alphaPool == nil {
		return "", fmt.Errorf("no alphaPool")
	}
	var id string
	// qualified path for column nodes in the catalog is typically /schema/table/column
	qualified := fmt.Sprintf("/%s/%s/%s", schema, table, column)
	// Try to query catalog_node for the column node id
	err := alphaPool.QueryRow(ctx, `SELECT id FROM public.catalog_node WHERE tenant_datasource_id = $1 AND qualified_path = $2 LIMIT 1`, tenantDatasourceID, qualified).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// toJSONStringArray converts a slice of strings into a JSON array literal
// suitable for embedding in a small JSON object. On error it returns '[]'.
func toJSONStringArray(arr []string) string {
	if arr == nil {
		return "[]"
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(b)
}
