package mining

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type JoinCandidate struct {
	LeftTable  string `json:"left_table"`
	LeftCol    string `json:"left_column"`
	RightTable string `json:"right_table"`
	RightCol   string `json:"right_column"`
}

type MetricCandidate struct {
	AliasName     string   `json:"alias_name"`
	FormulaAST    string   `json:"formula_ast"`
	SourceColumns []string `json:"source_columns"`
}

type QueryMiningDaemon struct {
	db *sql.DB
}

func NewQueryMiningDaemon(db *sql.DB) *QueryMiningDaemon {
	return &QueryMiningDaemon{db: db}
}

// ProcessQueryLogEntry parses a historical query and registers candidate patterns
func (d *QueryMiningDaemon) ProcessQueryLogEntry(
	ctx context.Context,
	tenantID uuid.UUID,
	dialect, rawSQL string,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	upperSQL := strings.ToUpper(rawSQL)

	// 1. Detect Equi-Joins via regex/heuristics
	if strings.Contains(upperSQL, "JOIN") && strings.Contains(upperSQL, " ON ") {
		d.stageCandidateJoin(ctx, tenantID, dialect, rawSQL)
	}

	// 2. Detect Arithmetic Metric Calculations
	if strings.Contains(upperSQL, "SELECT") && (strings.Contains(upperSQL, "*") || strings.Contains(upperSQL, "+") || strings.Contains(upperSQL, "-")) {
		if strings.Contains(upperSQL, " AS ") {
			d.stageCandidateMetric(ctx, tenantID, dialect, "mined_metric", rawSQL)
		}
	}

	return nil
}

func (d *QueryMiningDaemon) stageCandidateMetric(
	ctx context.Context,
	tenantID uuid.UUID,
	dialect, aliasName, formula string,
) {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s", aliasName, formula)))
	sig := hex.EncodeToString(h.Sum(nil))

	if d.db != nil {
		query := `
			INSERT INTO catalog_mining.discovered_patterns (
				tenant_id, pattern_type, pattern_signature, source_dialect,
				raw_expression, normalized_ast, execution_frequency, last_detected_at
			) VALUES ($1, 'METRIC_FORMULA', $2, $3, $4, $5, 1, NOW())
			ON CONFLICT (tenant_id, pattern_signature) DO UPDATE SET
				execution_frequency = catalog_mining.discovered_patterns.execution_frequency + 1,
				last_detected_at = NOW();`

		astPayload, _ := json.Marshal(map[string]string{"alias": aliasName, "formula": formula})
		_, _ = d.db.ExecContext(ctx, query, tenantID, sig, dialect, formula, astPayload)
	}
}

func (d *QueryMiningDaemon) stageCandidateJoin(
	ctx context.Context,
	tenantID uuid.UUID,
	dialect, joinSQL string,
) {
	h := sha256.New()
	h.Write([]byte(joinSQL))
	sig := hex.EncodeToString(h.Sum(nil))

	if d.db != nil {
		query := `
			INSERT INTO catalog_mining.discovered_patterns (
				tenant_id, pattern_type, pattern_signature, source_dialect,
				raw_expression, normalized_ast, execution_frequency, last_detected_at
			) VALUES ($1, 'IMPLICIT_JOIN', $2, $3, $4, $5, 1, NOW())
			ON CONFLICT (tenant_id, pattern_signature) DO UPDATE SET
				execution_frequency = catalog_mining.discovered_patterns.execution_frequency + 1,
				last_detected_at = NOW();`

		joinJSON, _ := json.Marshal(map[string]string{"raw_join": joinSQL})
		_, _ = d.db.ExecContext(ctx, query, tenantID, sig, dialect, joinSQL, joinJSON)
	}
}
