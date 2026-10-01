/**
 * Splits ids into request-sized chunks.
 *
 * The glossary preview endpoint caps column_ids per request (backend
 * previewCap in internal/api/service.go). A datasource can hold more columns
 * than that, so callers send the list in chunks and merge the results.
 */
export function chunkIds<T>(ids: T[], size: number): T[][] {
  if (size <= 0) throw new Error('chunk size must be positive');
  const out: T[][] = [];
  for (let i = 0; i < ids.length; i += size) {
    out.push(ids.slice(i, i + size));
  }
  return out;
}
