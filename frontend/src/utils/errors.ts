import tError from './locales';

export function getErrorMessage(e: unknown, fallback = 'Request failed'): string {
  if (!e) return fallback;
  // Try to handle Fetch / plain Error responses that contain structured JSON
  // Expected backend shape: { error: string, code: number, error_code?: string, details?: any }
  try {
    const maybeResponse = e as any;
    // If the error is an instance of Response or contains response-like body text
    if (maybeResponse?.json && typeof maybeResponse.json === 'function') {
      // In some callers we pass the Response object itself; leave to caller to await .text()/.json()
    }

    // Common axios-like wrapper
    if (maybeResponse?.response?.data) {
      const data = maybeResponse.response.data;
      if (data.error_code) return tError(data.error_code, data.error || fallback);
      if (data.error) return String(data.error);
      if (data.message) return String(data.message);
    }

    // If thrown error contains parsed body fields
    if (maybeResponse?.error_code || maybeResponse?.error) {
      return tError(maybeResponse.error_code, maybeResponse.error || fallback);
    }

    if (typeof maybeResponse.message === 'string') return maybeResponse.message;
    if (typeof maybeResponse === 'string') return maybeResponse;
  } catch (err) {
    // fall through
  }

  try {
    return String(e);
  } catch {
    return fallback;
  }
}
export default getErrorMessage;
