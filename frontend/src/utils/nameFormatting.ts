/**
 * Utility functions for converting between business names and technical names
 * Business Name: "Legal Name" (display-friendly)
 * Technical Name: "legal_name" (lowercase with underscores)
 */

/**
 * Convert business name to technical name
 * @param businessName - Display name (e.g., "Legal Name", "Date of Birth")
 * @returns Technical name (e.g., "legal_name", "date_of_birth")
 */
export const businessToTechnicalName = (businessName: string): string => {
  if (!businessName) return '';
  return businessName
    .toLowerCase() // Convert to lowercase
    .trim()
    .replace(/\s+/g, '_') // Replace spaces with underscores
    .replace(/[^\w_]/g, ''); // Remove special characters except underscores
};

/**
 * Convert technical name to business name (title case)
 * @param technicalName - Technical name (e.g., "legal_name", "date_of_birth")
 * @returns Business name (e.g., "Legal Name", "Date of Birth")
 */
export const technicalToBusinessName = (technicalName: string): string => {
  if (!technicalName) return '';
  return technicalName
    .replace(/_/g, ' ') // Replace underscores with spaces
    .split(' ')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
};

/**
 * Check if a technical name is valid
 * @param technicalName - Name to validate
 * @returns true if valid (lowercase, underscores only)
 */
export const isValidTechnicalName = (technicalName: string): boolean => {
  if (!technicalName) return false;
  return /^[a-z_][a-z0-9_]*$/.test(technicalName);
};

/**
 * Ensure both business and technical names are set consistently
 * If only businessName provided, generate technicalName
 * If only technicalName provided, generate businessName
 */
export const normalizeName = (
  businessName: string | undefined,
  technicalName: string | undefined
): { businessName: string; technicalName: string } => {
  let final_business = businessName || '';
  let final_technical = technicalName || '';

  // If both missing, return empty
  if (!final_business && !final_technical) {
    return { businessName: '', technicalName: '' };
  }

  // If only business name provided, generate technical
  if (final_business && !final_technical) {
    final_technical = businessToTechnicalName(final_business);
  }

  // If only technical name provided, generate business
  if (final_technical && !final_business) {
    final_business = technicalToBusinessName(final_technical);
  }

  return { businessName: final_business, technicalName: final_technical };
};
