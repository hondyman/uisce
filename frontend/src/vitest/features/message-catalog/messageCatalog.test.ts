import { describe, expect, it } from 'vitest';
import { effective, formatMessage, MessageView, placeholders, samePlaceholders } from '../../../features/message-catalog/api';
import { CatalogError, parseCatalogError } from '../../../utils/catalogError';
import en from '../../../locales/en.json';
import es from '../../../locales/es.json';
import fr from '../../../locales/fr.json';

describe('catalog error envelope', () => {
  const body = {
    error: "No se encontró el conjunto de mensajes 4242.",
    code: 404,
    error_code: '9100-7',
    severity: 'Error',
    language: 'es',
    correlation_id: 'req-123',
  };

  it('recognises the backend msgcat envelope', () => {
    const parsed = parseCatalogError(JSON.stringify(body));
    expect(parsed?.error_code).toBe('9100-7');
    const err = new CatalogError(parsed!, 404);
    expect(err.message).toBe(body.error);
    expect(err.code).toBe('9100-7');
    expect(err.correlationId).toBe('req-123');
  });

  it('leaves other error bodies alone', () => {
    expect(parseCatalogError('Forbidden')).toBeNull();
    expect(parseCatalogError(JSON.stringify({ error: 'x', code: 400 }))).toBeNull();
    expect(parseCatalogError(JSON.stringify({ error: 'x', error_code: 'not_found' }))).toBeNull();
  });
});

describe('message parameters (twin of backend msgcat/params.go)', () => {
  it('formats in one pass', () => {
    expect(formatMessage('Run %1 failed: %2.', ['R-7', 'bad %1'])).toBe('Run R-7 failed: bad %1.');
    expect(formatMessage('%2 then %1 and %3', ['a', 'b'])).toBe('b then a and ');
  });

  it('compares parameter sets', () => {
    expect(placeholders('%2 of %1, again %2')).toEqual(['%1', '%2']);
    expect(samePlaceholders('Run %1 failed: %2', 'Échec %2 de %1')).toBe(true);
    expect(samePlaceholders('%1', '%1 %2')).toBe(false);
  });
});

describe('effective text', () => {
  const entry = (language: string, text: string) => ({
    set_nbr: 1, message_nbr: 6, language, severity: 'Error' as const, text, updated_at: '',
  });
  const m: MessageView = {
    set_nbr: 1, message_nbr: 6, code: '1-6', severity: 'Error', pending: 0,
    core: { en: entry('en', '%1 was not found.'), fr: entry('fr', '%1 est introuvable.') },
    tenant: { en: entry('en', 'Northwind could not find %1.') },
  };

  it("prefers the user's language, then the tenant's text, then English", () => {
    expect(effective(m, 'en')).toMatchObject({ source: 'tenant', entry: { text: 'Northwind could not find %1.' } });
    expect(effective(m, 'fr')).toMatchObject({ source: 'core', entry: { text: '%1 est introuvable.' } });
    expect(effective(m, 'de')).toMatchObject({ source: 'fallback', entry: { text: 'Northwind could not find %1.' } });
  });
});

describe('catalog page strings', () => {
  const flat = (o: Record<string, unknown>, p = ''): string[] =>
    Object.entries(o).flatMap(([k, v]) => (v && typeof v === 'object' ? flat(v as Record<string, unknown>, `${p}${k}.`) : [`${p}${k}`]));
  const base = (keys: string[]) => new Set(keys.map((k) => k.replace(/_(one|many|other)$/, '')));

  it('exist in every shipped language', () => {
    const want = base(flat((en as Record<string, unknown>).messageCatalog as Record<string, unknown>));
    for (const [name, loc] of [['es', es], ['fr', fr]] as const) {
      const got = base(flat((loc as Record<string, unknown>).messageCatalog as Record<string, unknown>));
      expect([...want].filter((k) => !got.has(k)), name).toEqual([]);
    }
  });
});
