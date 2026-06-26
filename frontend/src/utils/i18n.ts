// Minimal i18n helper with in-memory locale map.
// This will fallback to a local map but, if `i18next` is available,
// we'll prefer the configured translations (so we can migrate gradually).
const TRANSLATIONS: Record<string, string> = {
  // Relationships table
  'relationships.id': 'ID',
  'relationships.edge_type_name': 'Predicate',
  'relationships.description': 'Description',
  'relationships.subject': 'Subject Type',
  'relationships.object': 'Object Type',
  'relationships.active': 'Active',

  // Dialog
  'relationships.create': 'Create Relationship',
  'relationships.select_edge_type': 'Edge Type',
  'relationships.select_node': 'Select related node to link',

  // Node types (human readable)
  'node_type.semantic_term': 'Semantic Term',
  'node_type.business_term': 'Business Term',
  'node_type.semantic_column': 'Semantic Column',
  'node_type.database_column': 'Database Column',
  'node_type.semantic_model': 'Semantic Model',
};

import i18n from '../i18n';

export function t(key: string, fallback?: string) {
  // prefer i18n if initialized
  try {
    if (i18n && typeof i18n.t === 'function') {
      const result = i18n.t(key);
      // i18next returns the key if translation missing, so use fallback if it's the same
      if (result && result !== key) return result;
    }
    } catch (e) {
    // ignore – we'll fallback to the in-repo translations
  }

  return TRANSLATIONS[key] ?? fallback ?? key;
}

export function setLanguage(code: string) {
  try { localStorage.setItem('selected_language', code); } catch (e) { }
  if (i18n && typeof i18n.changeLanguage === 'function') {
    void i18n.changeLanguage(code);
  }
}

export default t;
