package semantic

SemanticTerm: {
  id: string & =~"^[a-z]+\\.[a-z0-9_]+$"
  term_type: "physical" | "calculated"
  data_type: "string" | "number" | "boolean" | "date" | "json"
  owner: string
  steward: string
  status: "draft" | "in_review" | "approved" | "published" | "deprecated"
  version: int
  physical_table?: string
  physical_column?: string
  expression?: string
  tie_breaker?: {
    strategy: "precedence" | "latest_timestamp" | "custom"
    precedence?: [...string]
    description?: string
  }
  materialization?: "virtual" | "view" | "materialized_table"
}

holding_market_value_raw: SemanticTerm & {
  id: "holding.market_value_raw"
  term_type: "physical"
  data_type: "number"
  physical_table: "holdings"
  physical_column: "market_value"
  owner: "Data Engineering"
  steward: "Ops Analytics"
  status: "published"
  version: 1
}

holding_market_value_resolved: SemanticTerm & {
  id: "holding.market_value_resolved"
  term_type: "calculated"
  data_type: "number"
  expression: """
  CASE
    WHEN holding_type = 'SETTLED' THEN market_value
    WHEN holding_type = 'EOD' THEN market_value
    WHEN holding_type = 'SOD' THEN market_value
    ELSE market_value
  END
  """
  tie_breaker: {
    strategy: "precedence"
    precedence: ["SETTLED","EOD","SOD"]
    description: "Prefer SETTLED when present, otherwise EOD, otherwise SOD"
  }
  materialization: "materialized_table"
  owner: "Data Engineering"
  steward: "Ops Analytics"
  status: "draft"
  version: 1
}
