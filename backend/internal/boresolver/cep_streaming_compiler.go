package boresolver

import (
	"fmt"
	"strings"
)

// StreamingGenerationRequest holds parameters specific to continuous stream processing
type StreamingGenerationRequest struct {
	SQLGenerationRequest
	Fields         []string // e.g., ["account_id", "margin_requirement"]
	TopicName      string   // e.g., "redpanda.tenant_123.market_ticks"
	WindowType     string   // e.g., "TUMBLE", "HOP"
	WindowInterval string   // e.g., "5 MINUTE", "10 SECOND"
	WatermarkDelay string   // e.g., "5 SECOND"
}

// CompileFlinkStreamingSQL translates the Semantic Graph into a Continuous Flink Query
func (g *BOSQLGenerator) CompileFlinkStreamingSQL(ctx *GenerationContext, req StreamingGenerationRequest) (string, error) {
	if req.TopicName == "" {
		return "", fmt.Errorf("TopicName is required for streaming compilation")
	}

	if len(req.Fields) == 0 {
		req.Fields = []string{"account_id", "margin_requirement"}
	}

	if req.WindowType == "" {
		req.WindowType = "TUMBLE"
	}

	if req.WindowInterval == "" {
		req.WindowInterval = "5 MINUTE"
	}

	// 1. Resolve Semantic Fields to Kafka/Redpanda JSON payload properties
	selects := make([]string, len(req.Fields))
	for i, f := range req.Fields {
		// Example: "account_id" -> "JSON_VALUE(payload, '$.account_id') AS account_id"
		selects[i] = fmt.Sprintf("JSON_VALUE(payload, '$.%s') AS %s", f, f)
	}

	// 2. Resolve Filters (AST to Flink SQL WHERE clauses)
	whereClause, err := g.ConvertFilters(ctx)
	if err != nil {
		whereClause = ""
	}

	// 3. 🚨 RULE 7 ENFORCEMENT (Data Layer Isolation)
	// Even in streaming, we must ensure the Flink job only reads the tenant's exact data.
	tenantFilter := fmt.Sprintf("JSON_VALUE(payload, '$.tenant_id') = '%s'", req.TenantID)
	if whereClause == "" {
		whereClause = tenantFilter
	} else {
		whereClause = fmt.Sprintf("(%s) AND %s", whereClause, tenantFilter)
	}

	// 4. Construct the Continuous Windowed Aggregation Query
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("SELECT %s, \n", strings.Join(selects, ", ")))

	// Add Flink Window Metadata
	builder.WriteString(fmt.Sprintf("  window_start, window_end \nFROM TABLE(\n  %s(TABLE %s, DESCRIPTOR(proctime), INTERVAL '%s')\n)\n",
		req.WindowType,
		req.TopicName,
		req.WindowInterval,
	))

	builder.WriteString(fmt.Sprintf("WHERE %s \n", whereClause))

	// Grouping for Continuous Aggregation
	builder.WriteString("GROUP BY window_start, window_end")
	for _, f := range req.Fields {
		builder.WriteString(fmt.Sprintf(", %s", f))
	}

	return builder.String(), nil
}
