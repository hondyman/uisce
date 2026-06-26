/**
 * Utility functions for validating view names to prevent invalid API calls
 */

export const isInvalidViewName = (name: string): boolean => {
  if (!name || typeof name !== 'string') {
    return true;
  }

  const trimmed = name.trim().toLowerCase();
  
  // Empty or whitespace-only names
  if (!trimmed) {
    return true;
  }

  // Check for obvious placeholder or test values
  const invalidPatterns = [
    /^[x]{4,}$/,           // "xxxx", "xxxxxxxx", etc.
    /^test[0-9]*$/i,       // "test", "test123", etc.
    /^placeholder$/i,      // "placeholder"
    /^example$/i,          // "example"
    /^sample$/i,           // "sample"
    /^dummy$/i,            // "dummy"
    /^undefined$/i,        // "undefined"
    /^null$/i,             // "null"
    /^[0-9]+$/,            // Pure numbers "123", "456"
  ];
  
  return invalidPatterns.some(pattern => pattern.test(trimmed));
};

export const validateViewName = (name: string): { valid: boolean; error?: string } => {
  if (isInvalidViewName(name)) {
    return {
      valid: false,
      error: `Invalid view name: "${name}". Please provide a valid view name that is not a placeholder or test value.`
    };
  }

  return { valid: true };
};

export const assertValidViewName = (name: string): void => {
  const validation = validateViewName(name);
  if (!validation.valid) {
    throw new Error(validation.error);
  }
};