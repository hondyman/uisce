// Shared relationship-cardinality vocabulary, consumed by the query builder,
// page designer, report designer, and relationship UI so every surface
// branches on the same value instead of re-deriving/guessing it.
//
// Mirrors backend/internal/models/cardinality.go's Display() wire format.
export type Cardinality = '1:1' | '1:M' | 'M:1' | 'M:M';

/**
 * Normalizes the many loose strings different backend paths have written
 * historically ('one-to-many', '1:N', 'ONE_TO_MANY', lowercase, etc.) into
 * the canonical wire format. Returns undefined for unrecognized input
 * rather than guessing.
 */
export function normalizeCardinality(raw: string | undefined | null): Cardinality | undefined {
  if (!raw) return undefined;
  const c = raw.toUpperCase().replace(/[-\s]/g, '_');

  switch (c) {
    case 'ONE_TO_ONE':
    case '1:1':
    case '1_1':
      return '1:1';
    case 'ONE_TO_MANY':
    case '1:M':
    case '1:N':
    case '1_M':
    case '1_N':
      return '1:M';
    case 'MANY_TO_ONE':
    case 'M:1':
    case 'N:1':
    case 'M_1':
    case 'N_1':
      return 'M:1';
    case 'MANY_TO_MANY':
    case 'M:M':
    case 'N:M':
    case 'M:N':
    case 'N:N':
    case 'M_M':
    case 'N_M':
    case 'M_N':
    case 'N_N':
      return 'M:M';
    default:
      return undefined;
  }
}

/**
 * Whether the "many" side is on the target of the relationship — the
 * signal designers use to decide between an embedded/collection UI
 * (page designer's RelatedObjectsPalette renders these as draggable grid
 * widgets) and a single reference/lookup control.
 */
export function isToMany(cardinality: Cardinality | string | undefined | null): boolean {
  const normalized = typeof cardinality === 'string' ? normalizeCardinality(cardinality) : cardinality;
  return normalized === '1:M' || normalized === 'M:M';
}
