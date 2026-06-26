// Minimal localization map for API error codes.
// Extend this as needed. Keys are error_code values returned by the backend.
const messages: Record<string, string> = {
  'validation_failed': 'Validation failed. Please check your inputs.',
  'datasource_not_found': 'Datasource not found. Please select a valid datasource.',
  'no_tables_selected': 'Please select at least one table to generate a model.',
  'internal_server_error': 'An unexpected server error occurred. Try again later.',
};

export function tError(code?: string, fallback?: string) {
  if (!code) return fallback || 'Request failed';
  return messages[code] || fallback || 'Request failed';
}

export default tError;
