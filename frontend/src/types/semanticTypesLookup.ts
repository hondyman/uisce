/**
 * Semantic Types Configuration
 * 
 * This file provides TypeScript types and constants for the semantic_types lookup table.
 * Used to type-safely reference semantic type combinations for nodes and edges.
 */

/** Semantic type category */
export enum SemanticTypeCategory {
  DIMENSION = 'Dimension',
  MEASURE = 'Measure',
  TIME = 'Time',
}

/** Data types that can be used in semantic types */
export enum DataType {
  STRING = 'string',
  NUMBER = 'number',
  BOOLEAN = 'boolean',
  TIME = 'time',
  GEO = 'geo',
  NUMBER_AGG = 'number_agg',
  COUNT = 'count',
  COUNT_DISTINCT = 'count_distinct',
  COUNT_DISTINCT_APPROX = 'count_distinct_approx',
  SUM = 'sum',
  AVG = 'avg',
  MIN = 'min',
  MAX = 'max',
}

/** Format types for display and behavior */
export enum Format {
  DEFAULT = 'default',
  IMAGE_URL = 'imageUrl',
  LINK = 'link',
  CURRENCY = 'currency',
  PERCENT = 'percent',
  ID = 'id',
}

/** Semantic type value identifiers */
export enum SemanticTypeValue {
  // Dimensions - String
  DIMENSION_STRING_DEFAULT = 'dimension_string_default',
  DIMENSION_STRING_IMAGEURL = 'dimension_string_imageurl',
  DIMENSION_STRING_LINK = 'dimension_string_link',
  DIMENSION_STRING_CURRENCY = 'dimension_string_currency',
  DIMENSION_STRING_PERCENT = 'dimension_string_percent',

  // Dimensions - Number
  DIMENSION_NUMBER_DEFAULT = 'dimension_number_default',
  DIMENSION_NUMBER_ID = 'dimension_number_id',
  DIMENSION_NUMBER_CURRENCY = 'dimension_number_currency',
  DIMENSION_NUMBER_PERCENT = 'dimension_number_percent',

  // Dimensions - Boolean
  DIMENSION_BOOLEAN_DEFAULT = 'dimension_boolean_default',

  // Dimensions - Time
  DIMENSION_TIME_DEFAULT = 'dimension_time_default',

  // Dimensions - Geo
  DIMENSION_GEO_DEFAULT = 'dimension_geo_default',

  // Measures - Simple Types
  MEASURE_STRING_DEFAULT = 'measure_string_default',
  MEASURE_TIME_DEFAULT = 'measure_time_default',
  MEASURE_BOOLEAN_DEFAULT = 'measure_boolean_default',

  // Measures - Number
  MEASURE_NUMBER_DEFAULT = 'measure_number_default',
  MEASURE_NUMBER_PERCENT = 'measure_number_percent',
  MEASURE_NUMBER_CURRENCY = 'measure_number_currency',

  // Measures - Aggregations
  MEASURE_NUMBER_AGG_DEFAULT = 'measure_number_agg_default',
  MEASURE_NUMBER_AGG_PERCENT = 'measure_number_agg_percent',
  MEASURE_NUMBER_AGG_CURRENCY = 'measure_number_agg_currency',

  MEASURE_COUNT_DEFAULT = 'measure_count_default',
  MEASURE_COUNT_DISTINCT_DEFAULT = 'measure_count_distinct_default',
  MEASURE_COUNT_DISTINCT_APPROX_DEFAULT = 'measure_count_distinct_approx_default',

  MEASURE_SUM_DEFAULT = 'measure_sum_default',
  MEASURE_SUM_CURRENCY = 'measure_sum_currency',

  MEASURE_AVG_DEFAULT = 'measure_avg_default',
  MEASURE_MIN_DEFAULT = 'measure_min_default',
  MEASURE_MAX_DEFAULT = 'measure_max_default',

  // Time
  TIME_TIME_DEFAULT = 'time_time_default',
}

/** Semantic type metadata structure */
export interface SemanticTypeMetadata {
  semantic_type: SemanticTypeCategory;
  data_type: DataType;
  format: Format;
  notes: string;
}

/** Lookup value entry from API */
export interface SemanticTypeLookupValue {
  id: string;
  lookup_id: string;
  tenant_id: string;
  value: SemanticTypeValue;
  label: string;
  metadata: SemanticTypeMetadata;
  parent_id?: string | null;
  created_at?: string;
}

/** Lookup entry from API */
export interface SemanticTypeLookup {
  id: string;
  tenant_id: string;
  name: 'semantic_types';
  description: string;
  created_at?: string;
  updated_at?: string;
}

/** Helper type for property definition with semantic_types lookup */
export interface SemanticTypeProperty {
  name: string;
  label: string;
  lookup_id: string; // References the semantic_types lookup
  data_type: 'string';
  required?: boolean;
  default?: SemanticTypeValue;
}

/** Pre-defined semantic type groups for common use cases */
export const SEMANTIC_TYPE_GROUPS = {
  dimensions: {
    string: [
      SemanticTypeValue.DIMENSION_STRING_DEFAULT,
      SemanticTypeValue.DIMENSION_STRING_IMAGEURL,
      SemanticTypeValue.DIMENSION_STRING_LINK,
      SemanticTypeValue.DIMENSION_STRING_CURRENCY,
      SemanticTypeValue.DIMENSION_STRING_PERCENT,
    ],
    number: [
      SemanticTypeValue.DIMENSION_NUMBER_DEFAULT,
      SemanticTypeValue.DIMENSION_NUMBER_ID,
      SemanticTypeValue.DIMENSION_NUMBER_CURRENCY,
      SemanticTypeValue.DIMENSION_NUMBER_PERCENT,
    ],
    boolean: [SemanticTypeValue.DIMENSION_BOOLEAN_DEFAULT],
    time: [SemanticTypeValue.DIMENSION_TIME_DEFAULT],
    geo: [SemanticTypeValue.DIMENSION_GEO_DEFAULT],
  },
  measures: {
    simple: [
      SemanticTypeValue.MEASURE_STRING_DEFAULT,
      SemanticTypeValue.MEASURE_TIME_DEFAULT,
      SemanticTypeValue.MEASURE_BOOLEAN_DEFAULT,
    ],
    number: [
      SemanticTypeValue.MEASURE_NUMBER_DEFAULT,
      SemanticTypeValue.MEASURE_NUMBER_PERCENT,
      SemanticTypeValue.MEASURE_NUMBER_CURRENCY,
    ],
    aggregations: [
      SemanticTypeValue.MEASURE_NUMBER_AGG_DEFAULT,
      SemanticTypeValue.MEASURE_NUMBER_AGG_PERCENT,
      SemanticTypeValue.MEASURE_NUMBER_AGG_CURRENCY,
      SemanticTypeValue.MEASURE_COUNT_DEFAULT,
      SemanticTypeValue.MEASURE_COUNT_DISTINCT_DEFAULT,
      SemanticTypeValue.MEASURE_COUNT_DISTINCT_APPROX_DEFAULT,
      SemanticTypeValue.MEASURE_SUM_DEFAULT,
      SemanticTypeValue.MEASURE_SUM_CURRENCY,
      SemanticTypeValue.MEASURE_AVG_DEFAULT,
      SemanticTypeValue.MEASURE_MIN_DEFAULT,
      SemanticTypeValue.MEASURE_MAX_DEFAULT,
    ],
  },
  time: [SemanticTypeValue.TIME_TIME_DEFAULT],
};

/** Utility functions */

/**
 * Get semantic type metadata from value
 */
export function getSemanticTypeMetadata(value: SemanticTypeValue): SemanticTypeMetadata | null {
  const metadata: Record<SemanticTypeValue, SemanticTypeMetadata> = {
    // Dimensions - String
    [SemanticTypeValue.DIMENSION_STRING_DEFAULT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.STRING,
      format: Format.DEFAULT,
      notes: '',
    },
    [SemanticTypeValue.DIMENSION_STRING_IMAGEURL]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.STRING,
      format: Format.IMAGE_URL,
      notes: 'Dimension Format',
    },
    [SemanticTypeValue.DIMENSION_STRING_LINK]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.STRING,
      format: Format.LINK,
      notes: 'Dimension Format',
    },
    [SemanticTypeValue.DIMENSION_STRING_CURRENCY]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.STRING,
      format: Format.CURRENCY,
      notes: 'Dimension Format (If underlying type is number and formatted as string in SQL)',
    },
    [SemanticTypeValue.DIMENSION_STRING_PERCENT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.STRING,
      format: Format.PERCENT,
      notes: 'Dimension Format (If underlying type is number and formatted as string in SQL)',
    },

    // Dimensions - Number
    [SemanticTypeValue.DIMENSION_NUMBER_DEFAULT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.NUMBER,
      format: Format.DEFAULT,
      notes: '',
    },
    [SemanticTypeValue.DIMENSION_NUMBER_ID]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.NUMBER,
      format: Format.ID,
      notes: 'Dimension Format',
    },
    [SemanticTypeValue.DIMENSION_NUMBER_CURRENCY]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.NUMBER,
      format: Format.CURRENCY,
      notes: 'Dimension Format',
    },
    [SemanticTypeValue.DIMENSION_NUMBER_PERCENT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.NUMBER,
      format: Format.PERCENT,
      notes: 'Dimension Format',
    },

    // Dimensions - Boolean
    [SemanticTypeValue.DIMENSION_BOOLEAN_DEFAULT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.BOOLEAN,
      format: Format.DEFAULT,
      notes: '',
    },

    // Dimensions - Time
    [SemanticTypeValue.DIMENSION_TIME_DEFAULT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.TIME,
      format: Format.DEFAULT,
      notes: '',
    },

    // Dimensions - Geo
    [SemanticTypeValue.DIMENSION_GEO_DEFAULT]: {
      semantic_type: SemanticTypeCategory.DIMENSION,
      data_type: DataType.GEO,
      format: Format.DEFAULT,
      notes: '',
    },

    // Measures - Simple Types
    [SemanticTypeValue.MEASURE_STRING_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.STRING,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_TIME_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.TIME,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_BOOLEAN_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.BOOLEAN,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },

    // Measures - Number
    [SemanticTypeValue.MEASURE_NUMBER_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_NUMBER_PERCENT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER,
      format: Format.PERCENT,
      notes: 'Measure Format',
    },
    [SemanticTypeValue.MEASURE_NUMBER_CURRENCY]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER,
      format: Format.CURRENCY,
      notes: 'Measure Format',
    },

    // Measures - Aggregations
    [SemanticTypeValue.MEASURE_NUMBER_AGG_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER_AGG,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_NUMBER_AGG_PERCENT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER_AGG,
      format: Format.PERCENT,
      notes: 'Measure Format',
    },
    [SemanticTypeValue.MEASURE_NUMBER_AGG_CURRENCY]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.NUMBER_AGG,
      format: Format.CURRENCY,
      notes: 'Measure Format',
    },
    [SemanticTypeValue.MEASURE_COUNT_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.COUNT,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_COUNT_DISTINCT_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.COUNT_DISTINCT,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_COUNT_DISTINCT_APPROX_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.COUNT_DISTINCT_APPROX,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_SUM_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.SUM,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_SUM_CURRENCY]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.SUM,
      format: Format.CURRENCY,
      notes: 'Measure Format',
    },
    [SemanticTypeValue.MEASURE_AVG_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.AVG,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_MIN_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.MIN,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },
    [SemanticTypeValue.MEASURE_MAX_DEFAULT]: {
      semantic_type: SemanticTypeCategory.MEASURE,
      data_type: DataType.MAX,
      format: Format.DEFAULT,
      notes: 'Measure Type',
    },

    // Time
    [SemanticTypeValue.TIME_TIME_DEFAULT]: {
      semantic_type: SemanticTypeCategory.TIME,
      data_type: DataType.TIME,
      format: Format.DEFAULT,
      notes: 'Dedicated Semantic Time Object',
    },
  };

  return metadata[value] || null;
}

/**
 * Check if a semantic type is a dimension
 */
export function isDimension(value: SemanticTypeValue): boolean {
  return Object.values(SEMANTIC_TYPE_GROUPS.dimensions)
    .flat()
    .includes(value);
}

/**
 * Check if a semantic type is a measure
 */
export function isMeasure(value: SemanticTypeValue): boolean {
  return Object.values(SEMANTIC_TYPE_GROUPS.measures)
    .flat()
    .includes(value);
}

/**
 * Check if a semantic type is a time type
 */
export function isTimeType(value: SemanticTypeValue): boolean {
  return SEMANTIC_TYPE_GROUPS.time.includes(value);
}

/**
 * Get the category of a semantic type
 */
export function getCategory(value: SemanticTypeValue): SemanticTypeCategory | null {
  const metadata = getSemanticTypeMetadata(value);
  return metadata?.semantic_type || null;
}

/**
 * Filter semantic types by category
 */
export function filterByCategory(
  values: SemanticTypeValue[],
  category: SemanticTypeCategory
): SemanticTypeValue[] {
  return values.filter(v => getCategory(v) === category);
}

/**
 * Filter semantic types by data type
 */
export function filterByDataType(values: SemanticTypeValue[], dataType: DataType): SemanticTypeValue[] {
  return values.filter(v => {
    const metadata = getSemanticTypeMetadata(v);
    return metadata?.data_type === dataType;
  });
}

/**
 * Filter semantic types by format
 */
export function filterByFormat(values: SemanticTypeValue[], format: Format): SemanticTypeValue[] {
  return values.filter(v => {
    const metadata = getSemanticTypeMetadata(v);
    return metadata?.format === format;
  });
}
