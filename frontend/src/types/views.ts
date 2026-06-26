export interface View {
  id?: string;
  name: string;
  title?: string;
  description?: string;
  cube_count?: number;
  folder_count?: number;
  modified_at?: string;
  etag?: string;
  tags?: string[];
  is_core?: boolean;
  isCore?: boolean;
  // optional canonical extends reference (UUID) or legacy name
  extends?: string;
  /**
   * Human-friendly display for the extends target.
   *
   * When the backend or a normalization step resolves a view's `extends` (which may be a UUID
   * or legacy name), the UI stores a readable form in `extends_display` (preferably the target
   * view's `title`, falling back to its `name`, and finally to the raw `extends`).
   *
   * This field is computed on the client when listing views and may be undefined for older
   * backend responses that do not include an `extends` or when the referenced view cannot
   * be resolved.
   */
  extends_display?: string;
}

export type MinimalView = Pick<View, 'name' | 'title' | 'description' | 'cube_count' | 'folder_count'>;

export function getViewIdentifier(v: Partial<View> | undefined): string {
  if (!v) return '';
  return (v as any).id || v.name || '';
}

/**
 * Returns a human-friendly text for a view's `extends` property.
 *
 * @param v - the view object (may already include a computed `extends_display`)
 * @param lookup - optional map keyed by lower-cased id or name to a View object used to resolve a UUID
 * @returns a display string (title/name/raw extends) or undefined when not available
 */
export function getViewExtendsDisplay(v: Partial<View> | undefined, lookup?: Record<string, Partial<View>>): string | undefined {
  if (!v) return undefined;
  // prefer explicit extends_display if computed upstream
  if ((v as any).extends_display) return String((v as any).extends_display);

  const ext = (v as any).extends;
  if (!ext || typeof ext !== 'string') return undefined;

  const extKey = String(ext).toLowerCase();
  if (lookup) {
    const target = lookup[extKey] || lookup[String(ext).toLowerCase()];
    if (target) return String(target.title || target.name || ext);
  }

  // no lookup available or not found; return raw extends string
  return String(ext);
}
