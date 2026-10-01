/**
 * Deduplicates a list of fields by their stable technical identifier.
 *
 * Priority: key → technicalName → id
 * Fields that lack all three identifiers are excluded.
 *
 * This is the single source of truth for all field-list deduplication across
 * the app. Using name (display label) as a fallback is intentionally avoided —
 * distinct fields in financial domain objects can share similar display names
 * (e.g. "Account Number" vs "Account Name"), so deduping on name risks
 * dropping technically distinct fields.
 */
type FieldIdentity = { key?: string; technicalName?: string; id?: string };

// `T extends object` (not FieldIdentity): an all-optional constraint is a "weak
// type", so TS rejects inputs that share no property with it, including the
// identifier-less fields this function is documented to exclude.
export function dedupeFields<T extends object>(fields: T[]): T[] {
  const seen = new Set<string>();
  return fields.filter((f) => {
    const { key, technicalName, id } = f as FieldIdentity;
    const identifier = key || technicalName || id;
    if (!identifier || seen.has(identifier)) return false;
    seen.add(identifier);
    return true;
  });
}
