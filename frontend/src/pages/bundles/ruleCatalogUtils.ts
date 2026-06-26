/**
 * Rules Catalog Utilities
 *
 * Utility functions for the Rules Catalog component
 */

export const getSeverityColor = (severity: string): string => {
  switch (severity) {
    case 'BLOCK':
      return '#EF4444';
    case 'WARNING':
      return '#F59E0B';
    case 'INFO':
      return '#3B82F6';
    default:
      return '#6B7280';
  }
};

export const getSeverityIcon = (severity: string): string => {
  switch (severity) {
    case 'BLOCK':
      return '🛑';
    case 'WARNING':
      return '⚠️';
    case 'INFO':
      return 'ℹ️';
    default:
      return '•';
  }
};

export const getSeverityOrder = (severity: string): number => {
  const severityOrder = { BLOCK: 0, WARNING: 1, INFO: 2 };
  return severityOrder[severity as keyof typeof severityOrder] || 99;
};