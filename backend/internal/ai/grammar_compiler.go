package ai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type GrammarCompiler struct {
	db *sql.DB
}

func NewGrammarCompiler(db *sql.DB) *GrammarCompiler {
	return &GrammarCompiler{db: db}
}

// CompileTenantASTGrammar builds an EBNF grammar restricting JSON output to valid catalog entities
func (g *GrammarCompiler) CompileTenantASTGrammar(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var terms []string
	var boKeys []string

	if g.db != nil {
		query := `
			SELECT DISTINCT node_key 
			FROM public.catalog_node 
			WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
			  AND is_active = TRUE
			ORDER BY node_key ASC;`

		rows, err := g.db.QueryContext(ctx, query, tenantID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var key string
				if err := rows.Scan(&key); err == nil {
					terms = append(terms, fmt.Sprintf(`"%s"`, key))
				}
			}
		}

		boQuery := `
			SELECT DISTINCT key 
			FROM public.business_objects 
			WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000')
			ORDER BY key ASC;`

		boRows, err := g.db.QueryContext(ctx, boQuery, tenantID)
		if err == nil {
			defer boRows.Close()
			for boRows.Next() {
				var k string
				if err := boRows.Scan(&k); err == nil {
					boKeys = append(boKeys, fmt.Sprintf(`"%s"`, k))
				}
			}
		}
	}

	if len(terms) == 0 {
		terms = []string{`"customer_country"`, `"order_date"`, `"order_total"`, `"market_value_usd"`, `"px_last"`}
	}

	if len(boKeys) == 0 {
		boKeys = []string{`"wealth.portfolio"`, `"oms.order"`, `"master.customer"`}
	}

	var sb strings.Builder
	sb.WriteString("root ::= ObjectPayload\n")
	sb.WriteString("ObjectPayload ::= \"{\" ws \"\\\"business_object\\\":\" ws BOKeys \",\" ws \"\\\"dimensions\\\":\" ws DimensionArray \",\" ws \"\\\"measures\\\":\" ws MeasureArray ws \"}\"\n")
	sb.WriteString(fmt.Sprintf("BOKeys ::= %s\n", strings.Join(boKeys, " | ")))
	sb.WriteString("DimensionArray ::= \"[\" ws (ValidTerm (ws \",\" ws ValidTerm)*)? ws \"]\"\n")
	sb.WriteString("MeasureArray ::= \"[\" ws (MeasureObject (ws \",\" ws MeasureObject)*)? ws \"]\"\n")
	sb.WriteString("MeasureObject ::= \"{\" ws \"\\\"term\\\":\" ws ValidTerm \",\" ws \"\\\"aggregation\\\":\" ws AggOp ws \"}\"\n")
	sb.WriteString(fmt.Sprintf("ValidTerm ::= %s\n", strings.Join(terms, " | ")))
	sb.WriteString("AggOp ::= \"\\\"SUM\\\"\" | \"\\\"AVG\\\"\" | \"\\\"COUNT\\\"\" | \"\\\"MIN\\\"\" | \"\\\"MAX\\\"\" | \"\\\"XIRR_KERNEL\\\"\"\n")
	sb.WriteString("ws ::= [ \\t\\n\\r]*\n")

	return sb.String(), nil
}
