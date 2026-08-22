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
export function dedupeFields<T extends { key?: string; technicalName?: string; id?: string }>(
  fields: T[]
): T[] {
  const seen = new Set<string>();
  return fields.filter((f) => {
    const identifier = f.key || f.technicalName || f.id;
    if (!identifier || seen.has(identifier)) return false;
    seen.add(identifier);
    return true;
  });
}
