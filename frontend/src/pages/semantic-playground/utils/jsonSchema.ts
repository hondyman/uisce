// JSON Schema for SemanticQuery validation in Monaco editor

export const semanticQueryJsonSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  type: "object",
  title: "Semantic Query",
  description: "A query in the semantic layer",
  required: ["datasource", "version", "select"],
  properties: {
    datasource: {
      type: "string",
      description: "The datasource/business object ID",
      examples: ["customers", "orders", "products"],
    },
    version: {
      type: "string",
      description: "The bundle version",
      examples: ["v1", "v2", "latest"],
    },
    select: {
      type: "array",
      description: "Fields to select (semantic names)",
      items: {
        type: "string",
      },
      examples: [["customer_id", "customer_name", "loyalty_points"]],
    },
    filters: {
      type: "array",
      description: "Filter conditions",
      items: {
        type: "object",
        required: ["field", "op", "value"],
        properties: {
          field: {
            type: "string",
            description: "Field name (semantic)",
          },
          op: {
            type: "string",
            enum: ["=", "!=", ">", "<", ">=", "<=", "in", "not_in", "like"],
            description: "Comparison operator",
          },
          value: {
            description: "Filter value (can be string, number, array, etc)",
          },
        },
      },
      examples: [
        [
          {
            field: "country",
            op: "=",
            value: "US",
          },
          {
            field: "loyalty_points",
            op: ">",
            value: 100,
          },
        ],
      ],
    },
    order_by: {
      type: "array",
      description: "Order by clauses",
      items: {
        type: "object",
        required: ["field", "direction"],
        properties: {
          field: {
            type: "string",
            description: "Field name (semantic)",
          },
          direction: {
            type: "string",
            enum: ["asc", "desc"],
          },
        },
      },
      examples: [[{ field: "created_at", direction: "desc" }]],
    },
    limit: {
      type: "integer",
      description: "Result limit",
      minimum: 1,
      maximum: 10000,
      examples: [20, 100],
    },
    offset: {
      type: "integer",
      description: "Result offset for pagination",
      minimum: 0,
      examples: [0, 100],
    },
    aggregations: {
      type: "object",
      description: "Aggregation functions",
      examples: [
        {
          total_revenue: {
            function: "sum",
            field: "revenue",
          },
          avg_order_value: {
            function: "avg",
            field: "order_value",
          },
        },
      ],
    },
  },
};

// Monaco editor configuration for JSON with IntelliSense
export const monacoEditorOptions = {
  language: "json",
  theme: "vs-dark",
  fontSize: 13,
  fontFamily: "'Fira Code', 'Consolas', monospace",
  lineNumbers: "on",
  lineNumbersMinChars: 3,
  automaticLayout: true,
  minimap: {
    enabled: false,
  },
  formatOnPaste: true,
  formatOnType: true,
  wordWrap: "on",
  padding: {
    top: 12,
    bottom: 12,
  },
  scrollBeyondLastLine: false,
  folding: true,
  foldingHighlight: true,
};

// SQL syntax highlighting configuration
export const sqlEditorOptions = {
  language: "sql",
  theme: "vs-dark",
  fontSize: 13,
  fontFamily: "'Fira Code', 'Consolas', monospace",
  lineNumbers: "on",
  automaticLayout: true,
  readOnly: true,
  minimap: {
    enabled: false,
  },
  wordWrap: "on",
  padding: {
    top: 12,
    bottom: 12,
  },
  scrollBeyondLastLine: false,
  lineDecorationsWidth: 0,
};
