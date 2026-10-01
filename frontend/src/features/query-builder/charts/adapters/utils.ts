import type { QueryResultsData, ChartConfig, QueryColumnMeta } from '../types';

export interface ExtractedShelves {
  dimCol: QueryColumnMeta | undefined;
  secDimCol: QueryColumnMeta | undefined;
  measureCols: QueryColumnMeta[];
}

export function extractShelves(results: QueryResultsData, config?: ChartConfig): ExtractedShelves {
  const columns = results.columns || [];
  if (columns.length === 0) {
    return { dimCol: undefined, secDimCol: undefined, measureCols: [] };
  }

  // Find explicit or inferred primary dimension
  let dimCol = config?.dimCol
    ? columns.find((c) => c.name === config.dimCol || c.alias === config.dimCol)
    : columns.find((c) => c.type === 'dimension') || columns[0];

  // Find secondary dimension
  let secDimCol: QueryColumnMeta | undefined;
  if (config?.secondaryDimCol) {
    secDimCol = columns.find((c) => c.name === config.secondaryDimCol || c.alias === config.secondaryDimCol);
  } else {
    const candidateDims = columns.filter((c) => c !== dimCol && c.type === 'dimension');
    if (candidateDims.length > 0) {
      secDimCol = candidateDims[0];
    }
  }

  // Find measure columns
  let measureCols: QueryColumnMeta[] = [];
  if (config?.measureCols && config.measureCols.length > 0) {
    measureCols = columns.filter((c) => config.measureCols!.includes(c.name) || config.measureCols!.includes(c.alias || ''));
  } else if (config?.measureCol) {
    const m = columns.find((c) => c.name === config.measureCol || c.alias === config.measureCol);
    if (m) measureCols = [m];
  }

  if (measureCols.length === 0) {
    measureCols = columns.filter((c) => c !== dimCol && c !== secDimCol);
  }

  return { dimCol, secDimCol, measureCols };
}
