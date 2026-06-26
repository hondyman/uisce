export interface Dimension {
  id: string;
  name: string;
  type: string;
  sql: string;
  title?: string;
  description?: string;
}

export interface Measure {
  id: string;
  name: string;
  type: string;
  sql: string;
  title?: string;
  description?: string;
  drillMembers?: string[];
  drillDown?: (drillDownLocator: DrillDownLocator, pivotConfig?: PivotConfig) => Query | null;
  pivot?: (pivotConfig?: PivotConfig) => PivotRow[];
}

export interface Join {
  id: string;
  name: string;
  sql: string;
  relationship: string;
}

// Optional cube-level options to be passed through to YAML when present
export interface CubeOptions {
  title?: string;
  description?: string;
  public?: boolean;
  meta?: Record<string, any>;
  refresh_key?: any;
  access_policy?: any;
  segments?: Array<Record<string, any>>;
  hierarchies?: Array<Record<string, any>>;
  sql_alias?: string;
  data_source?: string;
}

export interface SemanticModelConfig {
  core: {
    dimensions: Dimension[];
    measures: Measure[];
    joins?: Join[];
    options?: CubeOptions; // optional core cube-level options
  };
  custom: {
    dimensions: Dimension[];
    measures: Measure[];
    joins?: Join[];
    overrides?: {
      dimensions?: Record<string, Partial<Dimension>>;
      measures?: Record<string, Partial<Measure>>;
    };
    options?: CubeOptions; // optional final cube-level options (applied in generateFinalYAML)
  };
}

export interface DatabaseColumn {
  name: string;
  type: string;
  description: string;
}

export interface DatabaseTable {
  name: string;
  columns: DatabaseColumn[];
}

export interface DrillDownLocator {
  xValues: any[];
  yValues: any[];
}

export interface PivotConfig {
  x?: string[];
  y?: string[];
}

export interface Query {
  measures: string[];
  dimensions: string[];
  filters: any[];
  timeDimensions: any[];
}

export interface PivotRow {
  xValues: any[];
  yValuesArray: any[][];
}

export const DATABASE_SCHEMA: { tables: DatabaseTable[] } = {
  tables: [
    {
      name: "categories",
      columns: [
        { name: "category_id", type: "int2", description: "Primary key" },
        { name: "category_name", type: "varchar(15)", description: "Category name" },
        { name: "description", type: "text", description: "Category description" },
        { name: "picture", type: "bytea", description: "Category picture" }
      ]
    },
    {
      name: "customers",
      columns: [
        { name: "customer_id", type: "varchar(5)", description: "Primary key" },
        { name: "company_name", type: "varchar(40)", description: "Company name" },
        { name: "contact_name", type: "varchar(30)", description: "Contact person name" },
        { name: "contact_title", type: "varchar(30)", description: "Contact title" },
        { name: "address", type: "varchar(60)", description: "Street address" },
        { name: "city", type: "varchar(15)", description: "City" },
        { name: "region", type: "varchar(15)", description: "Region/State" },
        { name: "postal_code", type: "varchar(10)", description: "Postal code" },
        { name: "country", type: "varchar(15)", description: "Country" },
        { name: "phone", type: "varchar(24)", description: "Phone number" },
        { name: "fax", type: "varchar(24)", description: "Fax number" }
      ]
    },
  ]
};