import i18n from '../i18n';

/**
 * The body every backend error answered through the Message Catalog has
 * (backend/internal/msgcat ErrorBody). `error` is already in the user's
 * language; the internal cause is never included.
 */
export interface CatalogErrorBody {
  error: string;
  code: number; // HTTP status
  error_code: string; // catalog code, "set-nbr"
  severity: 'Message' | 'Warning' | 'Error' | 'Fatal';
  user_action?: string;
  language?: string;
  correlation_id: string;
}

export function parseCatalogError(text: string): CatalogErrorBody | null {
  try {
    const body = JSON.parse(text);
    if (body && typeof body.error === 'string' && typeof body.error_code === 'string' && /^\d+-\d+$/.test(body.error_code)) {
      return body as CatalogErrorBody;
    }
  } catch {
    // not JSON
  }
  return null;
}

/** An error the backend answered with a catalog message. */
export class CatalogError extends Error {
  readonly status: number;
  readonly code: string;
  readonly severity: CatalogErrorBody['severity'];
  readonly userAction?: string;
  readonly correlationId: string;

  constructor(body: CatalogErrorBody, status: number) {
    super(body.error);
    this.name = 'CatalogError';
    this.status = status;
    this.code = body.error_code;
    this.severity = body.severity;
    this.userAction = body.user_action;
    this.correlationId = body.correlation_id;
  }
}

/** The UI language, sent as Accept-Language so errors come back in it. */
export function acceptLanguage(): string {
  const lng = i18n?.language;
  return lng && lng !== 'en' ? `${lng}, en;q=0.5` : 'en';
}
