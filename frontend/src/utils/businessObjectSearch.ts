export interface SearchableBusinessObject {
  id?: string | null;
  name?: string | null;
  display_name?: string | null;
  description?: string | null;
  driver_table_name?: string | null;
  category?: string | null;
}

/**
 * Type-ahead filter for the Business Objects list. Every space-separated word in the query has to appear
 * (case-insensitive) somewhere in the object's name, display name, description, driving table or category,
 * so "mdm party" finds the BO driven by /mdm/party. Missing fields are ignored rather than throwing.
 * An empty query returns the list unchanged.
 */
export function filterBusinessObjectsBySearch<T extends SearchableBusinessObject>(items: T[], query: string): T[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) return items;
  return items.filter((item) => {
    const haystack = [item.name, item.display_name, item.description, item.driver_table_name, item.category]
      .filter((v): v is string => typeof v === 'string' && v.length > 0)
      .join(' ')
      .toLowerCase();
    return words.every((w) => haystack.includes(w));
  });
}
